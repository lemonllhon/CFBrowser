package backend

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type authenticatedProxyBridge struct {
	upstream string
	server   *http.Server
	listener net.Listener
	stopOnce sync.Once
}

func proxyNeedsBrowserAuthBridge(proxyURL string) bool {
	u, err := url.Parse(strings.TrimSpace(proxyURL))
	if err != nil || u == nil || u.User == nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5":
		return u.User.Username() != ""
	default:
		return false
	}
}

func newAuthenticatedProxyBridge(upstream string) *authenticatedProxyBridge {
	return &authenticatedProxyBridge{upstream: strings.TrimSpace(upstream)}
}

func (b *authenticatedProxyBridge) start() (string, error) {
	if !proxyNeedsBrowserAuthBridge(b.upstream) {
		return b.upstream, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("启动代理认证中转端口失败: %w", err)
	}
	b.listener = ln
	b.server = &http.Server{Handler: b}
	go func() {
		_ = b.server.Serve(ln)
	}()
	return "http://" + ln.Addr().String(), nil
}

func (b *authenticatedProxyBridge) stop() {
	b.stopOnce.Do(func() {
		if b.server == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = b.server.Shutdown(ctx)
		cancel()
	})
}

func (b *authenticatedProxyBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		b.handleConnect(w, r)
		return
	}
	b.handleHTTP(w, r)
}

func (b *authenticatedProxyBridge) handleConnect(w http.ResponseWriter, r *http.Request) {
	dst, err := b.dialTarget(r.Context(), r.Host)
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

func (b *authenticatedProxyBridge) handleHTTP(w http.ResponseWriter, r *http.Request) {
	upstream, err := url.Parse(b.upstream)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	transport := &http.Transport{DisableKeepAlives: true}
	switch strings.ToLower(upstream.Scheme) {
	case "http", "https":
		transport.Proxy = http.ProxyURL(upstream)
	case "socks5":
		transport.Proxy = nil
		transport.DialContext = socksDialContext(upstream)
	default:
		http.Error(w, "unsupported upstream proxy", http.StatusBadGateway)
		return
	}
	req := r.Clone(r.Context())
	req.RequestURI = ""
	removeHopHeaders(req.Header)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (b *authenticatedProxyBridge) dialTarget(ctx context.Context, target string) (net.Conn, error) {
	upstream, err := url.Parse(b.upstream)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(upstream.Scheme) {
	case "socks5":
		return socksDialContext(upstream)(ctx, "tcp", target)
	case "http", "https":
		return dialHTTPProxyConnect(ctx, upstream, target)
	default:
		return nil, fmt.Errorf("不支持的代理认证中转上游协议: %s", upstream.Scheme)
	}
}
