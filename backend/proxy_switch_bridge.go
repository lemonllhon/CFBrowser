package backend

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/logger"
	"ant-chrome/backend/internal/proxy"
	xproxy "golang.org/x/net/proxy"
)

type switchingProxyBridge struct {
	app       *App
	profileID string
	groupName string
	mode      string
	interval  time.Duration

	server   *http.Server
	listener net.Listener

	mu        sync.RWMutex
	current   config.BrowserProxy
	hasCurrent bool
	recent    []string
	stopped   chan struct{}
	stopOnce  sync.Once
	rng       *rand.Rand
}

func newSwitchingProxyBridge(app *App, profile *BrowserProfile) *switchingProxyBridge {
	intervalM := profile.AutoProxySwitchIntervalM
	if intervalM <= 0 {
		intervalM = 5
	}
	if intervalM > 24*60 {
		intervalM = 24 * 60
	}
	return &switchingProxyBridge{
		app:       app,
		profileID: strings.TrimSpace(profile.ProfileId),
		groupName: strings.TrimSpace(profile.AutoProxySwitchGroupName),
		mode:      normalizeProxySwitchMode(profile.AutoProxySwitchMode),
		interval:  time.Duration(intervalM) * time.Minute,
		stopped:   make(chan struct{}),
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *switchingProxyBridge) start() (string, error) {
	if err := b.switchNow(); err != nil {
		return "", err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("启动代理中转端口失败: %w", err)
	}
	b.listener = ln
	b.server = &http.Server{Handler: b}
	go func() {
		if err := b.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.New("ProxySwitch").Error("代理自动切换中转异常退出", logger.F("profile_id", b.profileID), logger.F("error", err.Error()))
		}
	}()
	if b.mode == proxySwitchModeInterval {
		go b.rotationLoop()
	}
	return "http://" + ln.Addr().String(), nil
}

func (b *switchingProxyBridge) stop() {
	b.stopOnce.Do(func() {
		close(b.stopped)
		if b.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = b.server.Shutdown(ctx)
			cancel()
		}
	})
}

func (b *switchingProxyBridge) rotationLoop() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := b.switchNow(); err != nil {
				logger.New("ProxySwitch").Warn("代理自动切换失败", logger.F("profile_id", b.profileID), logger.F("error", err.Error()))
			}
		case <-b.stopped:
			return
		}
	}
}

func (b *switchingProxyBridge) switchNow() error {
	candidates := b.candidateProxies()
	if len(candidates) == 0 {
		return fmt.Errorf("代理池分组 %q 没有可用代理", b.groupName)
	}
	candidates = preferHealthyProxyCandidates(candidates)

	b.mu.Lock()
	defer b.mu.Unlock()

	currentID := ""
	if b.hasCurrent {
		currentID = b.current.ProxyId
	}
	filtered := make([]config.BrowserProxy, 0, len(candidates))
	recent := make(map[string]struct{}, len(b.recent)+1)
	if currentID != "" {
		recent[currentID] = struct{}{}
	}
	for _, id := range b.recent {
		recent[id] = struct{}{}
	}
	for _, item := range candidates {
		if _, ok := recent[item.ProxyId]; !ok {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 && len(candidates) > 1 {
		for _, item := range candidates {
			if item.ProxyId != currentID {
				filtered = append(filtered, item)
			}
		}
	}
	if len(filtered) == 0 {
		filtered = candidates
	}
	next := filtered[b.rng.Intn(len(filtered))]
	if b.hasCurrent && b.current.ProxyId != "" {
		b.recent = append(b.recent, b.current.ProxyId)
		limit := len(candidates) - 1
		if limit > 20 {
			limit = 20
		}
		if limit < 1 {
			limit = 1
		}
		if len(b.recent) > limit {
			b.recent = b.recent[len(b.recent)-limit:]
		}
	}
	b.current = next
	b.hasCurrent = true
	logger.New("ProxySwitch").Info("代理自动切换出口已更新",
		logger.F("profile_id", b.profileID),
		logger.F("proxy_id", next.ProxyId),
		logger.F("proxy_name", next.ProxyName),
		logger.F("group", b.groupName),
		logger.F("mode", b.mode),
	)
	return nil
}

func (b *switchingProxyBridge) candidateProxies() []config.BrowserProxy {
	proxies := b.app.getLatestProxies()
	items := make([]config.BrowserProxy, 0, len(proxies))
	for _, item := range proxies {
		if strings.TrimSpace(item.ProxyId) == "" || strings.TrimSpace(item.ProxyConfig) == "" {
			continue
		}
		if strings.EqualFold(item.ProxyId, "__direct__") || strings.EqualFold(strings.TrimSpace(item.ProxyConfig), "direct://") {
			continue
		}
		if b.groupName != "" && item.GroupName != b.groupName {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (b *switchingProxyBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		b.handleConnect(w, r)
		return
	}
	b.handleHTTP(w, r)
}

func (b *switchingProxyBridge) handleConnect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dst, err := b.dialTarget(ctx, r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		_ = dst.Close()
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}
	client, _, err := hj.Hijack()
	if err != nil {
		_ = dst.Close()
		return
	}
	_, _ = client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	go tunnel(dst, client)
	go tunnel(client, dst)
}

func (b *switchingProxyBridge) handleHTTP(w http.ResponseWriter, r *http.Request) {
	resp, release, err := b.roundTripHTTP(r)
	if err != nil && b.canFailoverOnRequestError() && shouldRetryProxyRequest(r) {
		firstErr := err
		if switchErr := b.switchNow(); switchErr == nil {
			resp, release, err = b.roundTripHTTP(r)
			if err != nil {
				err = fmt.Errorf("%v; 切换后仍失败: %w", firstErr, err)
			}
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer release()
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (b *switchingProxyBridge) dialTarget(ctx context.Context, target string) (net.Conn, error) {
	conn, err := b.dialTargetOnce(ctx, target)
	if err == nil {
		return conn, nil
	}

	firstErr := err
	if !b.canFailoverOnRequestError() {
		return nil, firstErr
	}
	if switchErr := b.switchNow(); switchErr != nil {
		return nil, firstErr
	}
	conn, err = b.dialTargetOnce(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("%v; 切换后仍失败: %w", firstErr, err)
	}
	return conn, nil
}

func (b *switchingProxyBridge) dialTargetOnce(ctx context.Context, target string) (net.Conn, error) {
	upstream, release, err := b.effectiveProxyURL()
	if err != nil {
		return nil, err
	}
	if upstream == "" {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", target)
		if err != nil {
			release()
			return nil, err
		}
		return newReleaseOnCloseConn(conn, release), nil
	}
	u, err := url.Parse(upstream)
	if err != nil {
		release()
		return nil, err
	}
	var conn net.Conn
	switch strings.ToLower(u.Scheme) {
	case "socks5":
		conn, err = socksDialContext(u)(ctx, "tcp", target)
	case "http", "https":
		conn, err = dialHTTPProxyConnect(ctx, u, target)
	default:
		release()
		return nil, fmt.Errorf("不支持的中转上游协议: %s", u.Scheme)
	}
	if err != nil {
		release()
		return nil, err
	}
	return newReleaseOnCloseConn(conn, release), nil
}

func (b *switchingProxyBridge) roundTripHTTP(r *http.Request) (*http.Response, func(), error) {
	upstream, release, err := b.effectiveProxyURL()
	if err != nil {
		return nil, nil, err
	}
	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	if upstream != "" {
		u, err := url.Parse(upstream)
		if err != nil {
			release()
			return nil, nil, err
		}
		switch strings.ToLower(u.Scheme) {
		case "http", "https":
			transport.Proxy = http.ProxyURL(u)
		case "socks5":
			transport.Proxy = nil
			transport.DialContext = socksDialContext(u)
		}
	}
	req := r.Clone(r.Context())
	req.RequestURI = ""
	removeHopHeaders(req.Header)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		release()
		return nil, nil, err
	}
	return resp, release, nil
}

func (b *switchingProxyBridge) effectiveProxyURL() (string, func(), error) {
	noopRelease := func() {}
	b.mu.RLock()
	current := b.current
	hasCurrent := b.hasCurrent
	b.mu.RUnlock()
	if !hasCurrent {
		if err := b.switchNow(); err != nil {
			return "", nil, err
		}
		b.mu.RLock()
		current = b.current
		b.mu.RUnlock()
	}
	proxies := b.app.getLatestProxies()
	src := strings.TrimSpace(current.ProxyConfig)
	if supported, errorMsg := proxy.ValidateProxyConfig(src, proxies, current.ProxyId); !supported {
		return "", nil, fmt.Errorf("%s", errorMsg)
	}
	if strings.EqualFold(src, "direct://") {
		return "", noopRelease, nil
	}
	if proxy.IsSingBoxProtocol(src) {
		if b.app.singboxMgr == nil {
			return "", nil, fmt.Errorf("sing-box 管理器未初始化")
		}
		socksURL, err := b.app.singboxMgr.EnsureBridge(src, proxies, current.ProxyId)
		return socksURL, noopRelease, err
	}
	if proxy.RequiresBridge(src, proxies, current.ProxyId) {
		if b.app.xrayMgr == nil {
			return "", nil, fmt.Errorf("xray 管理器未初始化")
		}
		socksURL, bridgeKey, err := b.app.xrayMgr.AcquireBridge(src, proxies, current.ProxyId)
		if err != nil {
			return "", nil, err
		}
		return socksURL, func() {
			if bridgeKey != "" {
				b.app.xrayMgr.ReleaseBridge(bridgeKey)
			}
		}, nil
	}
	standard, _, err := proxy.ParseProxyNode(src)
	if err != nil {
		return "", nil, err
	}
	if standard == "" {
		return "", nil, fmt.Errorf("代理节点无法转换为标准中转地址")
	}
	return standard, noopRelease, nil
}

func (b *switchingProxyBridge) canFailoverOnRequestError() bool {
	return b != nil && b.mode == proxySwitchModeInterval
}

type releaseOnCloseConn struct {
	net.Conn
	release func()
	once    sync.Once
}

func newReleaseOnCloseConn(conn net.Conn, release func()) net.Conn {
	if conn == nil || release == nil {
		return conn
	}
	return &releaseOnCloseConn{Conn: conn, release: release}
}

func (c *releaseOnCloseConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

func socksDialContext(u *url.URL) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		var auth *xproxy.Auth
		if u.User != nil {
			auth = &xproxy.Auth{User: u.User.Username()}
			auth.Password, _ = u.User.Password()
		}
		dialer, err := xproxy.SOCKS5(network, u.Host, auth, xproxy.Direct)
		if err != nil {
			return nil, err
		}
		type contextDialer interface {
			DialContext(context.Context, string, string) (net.Conn, error)
		}
		if cd, ok := dialer.(contextDialer); ok {
			return cd.DialContext(ctx, network, address)
		}
		done := make(chan struct{})
		var conn net.Conn
		var dialErr error
		go func() {
			conn, dialErr = dialer.Dial(network, address)
			close(done)
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-done:
			return conn, dialErr
		}
	}
}

func dialHTTPProxyConnect(ctx context.Context, u *url.URL, target string) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(u.Scheme, "https") {
		conn = tls.Client(conn, &tls.Config{ServerName: u.Hostname()})
	}
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: target},
		Host:   target,
		Header: make(http.Header),
	}
	if u.User != nil {
		pass, _ := u.User.Password()
		req.Header.Set("Proxy-Authorization", "Basic "+basicAuth(u.User.Username(), pass))
	}
	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_ = conn.Close()
		return nil, fmt.Errorf("上游 HTTP 代理 CONNECT 失败: %s", resp.Status)
	}
	return conn, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func tunnel(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	_, _ = io.Copy(dst, src)
}

func removeHopHeaders(h http.Header) {
	for _, key := range []string{"Proxy-Connection", "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade"} {
		h.Del(key)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func shouldRetryProxyRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func preferHealthyProxyCandidates(candidates []config.BrowserProxy) []config.BrowserProxy {
	healthy := make([]config.BrowserProxy, 0, len(candidates))
	for _, item := range candidates {
		if item.LastTestOk {
			healthy = append(healthy, item)
		}
	}
	if len(healthy) == 0 {
		return candidates
	}
	return healthy
}

const (
	proxySwitchModeInterval = "interval"
	proxySwitchModeManual   = "manual"
)

func normalizeProxySwitchMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case proxySwitchModeManual:
		return proxySwitchModeManual
	default:
		return proxySwitchModeInterval
	}
}
