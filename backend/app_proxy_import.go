package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	maxClashSubscriptionBytes = 8 * 1024 * 1024
	clashSubscriptionTimeout  = 25 * time.Second
)

// BrowserProxyFetchClashByURL 拉取 Clash 订阅 URL，并返回可直接导入的 YAML 文本与建议配置。
func (a *App) BrowserProxyFetchClashByURL(rawURL string) (map[string]interface{}, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("订阅 URL 不能为空")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Host == "" {
		return nil, fmt.Errorf("URL 格式无效")
	}
	scheme := strings.ToLower(strings.TrimSpace(parsedURL.Scheme))
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("仅支持 http/https URL")
	}

	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "clash-verge/2.0 ant-chrome/1.0")
	req.Header.Set("Accept", "application/yaml,text/yaml,text/plain,*/*")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{
		Timeout: clashSubscriptionTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("拉取订阅失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("拉取订阅失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxClashSubscriptionBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取订阅内容失败: %w", err)
	}
	if len(body) > maxClashSubscriptionBytes {
		return nil, fmt.Errorf("订阅内容过大（超过 8MB）")
	}

	content, payload, err := normalizeClashSubscriptionContent(body)
	if err != nil {
		return nil, err
	}

	proxyCount := clashProxyCount(payload)
	if proxyCount <= 0 {
		return nil, fmt.Errorf("未检测到可导入的 proxies 节点")
	}

	dnsYAML := extractClashDNSYAML(payload)
	suggestedGroup := suggestClashGroupName(payload, parsedURL.Hostname())

	return map[string]interface{}{
		"url":            parsedURL.String(),
		"content":        content,
		"proxyCount":     proxyCount,
		"dnsServers":     dnsYAML,
		"suggestedGroup": suggestedGroup,
	}, nil
}

func normalizeClashSubscriptionContent(body []byte) (string, interface{}, error) {
	baseText := strings.TrimSpace(strings.ReplaceAll(string(body), "\r\n", "\n"))
	if baseText == "" {
		return "", nil, fmt.Errorf("订阅内容为空")
	}

	tryTexts := collectClashImportTextVariants(baseText)

	for _, text := range tryTexts {
		payload, ok := parseClashPayload(text)
		if ok && clashProxyCount(payload) > 0 {
			return text, payload, nil
		}

		if payload, ok := parseShareSubscriptionPayload(text); ok {
			content, err := marshalClashPayload(payload)
			if err != nil {
				return "", nil, err
			}
			return content, payload, nil
		}
	}

	return "", nil, fmt.Errorf("URL 内容不是有效 Clash YAML 或 Base64 分享订阅（需包含可导入节点）")
}

func collectClashImportTextVariants(baseText string) []string {
	tryTexts := make([]string, 0, 6)
	seen := map[string]struct{}{}
	appendText := func(text string) {
		text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
		if text == "" {
			return
		}
		if _, exists := seen[text]; exists {
			return
		}
		seen[text] = struct{}{}
		tryTexts = append(tryTexts, text)
	}

	appendText(baseText)
	if unescaped, err := url.QueryUnescape(baseText); err == nil {
		appendText(unescaped)
	}

	for _, text := range append([]string(nil), tryTexts...) {
		if decoded, ok := decodeBase64Text(text); ok {
			appendText(decoded)
		}
	}

	return tryTexts
}

func decodeBase64Text(raw string) (string, bool) {
	candidate := strings.Join(strings.Fields(raw), "")
	if candidate == "" {
		return "", false
	}
	// 一些订阅会返回 URL-safe base64 或缺少 padding，这里都尝试一遍。
	padded := candidate
	if mod := len(padded) % 4; mod != 0 {
		padded += strings.Repeat("=", 4-mod)
	}

	encoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encoders {
		if data, err := enc.DecodeString(candidate); err == nil {
			decoded := strings.TrimSpace(strings.ReplaceAll(string(data), "\r\n", "\n"))
			if decoded != "" {
				return decoded, true
			}
		}
		if data, err := enc.DecodeString(padded); err == nil {
			decoded := strings.TrimSpace(strings.ReplaceAll(string(data), "\r\n", "\n"))
			if decoded != "" {
				return decoded, true
			}
		}
	}
	return "", false
}

func marshalClashPayload(payload interface{}) (string, error) {
	data, err := yaml.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("订阅内容转换失败: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func parseClashPayload(text string) (interface{}, bool) {
	var payload interface{}
	if err := yaml.Unmarshal([]byte(text), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func parseShareSubscriptionPayload(text string) (interface{}, bool) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	proxies := make([]interface{}, 0, len(lines))
	for _, line := range lines {
		node := strings.TrimSpace(line)
		if node == "" || strings.HasPrefix(node, "#") {
			continue
		}
		proxy, ok := parseShareURIToClashProxy(node, len(proxies))
		if ok {
			proxies = append(proxies, proxy)
		}
	}
	if len(proxies) == 0 {
		return nil, false
	}
	return map[string]interface{}{
		"proxies": proxies,
	}, true
}

func parseShareURIToClashProxy(raw string, index int) (map[string]interface{}, bool) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.HasPrefix(lower, "vmess://"):
		return parseVmessShareToClashProxy(raw, index)
	case strings.HasPrefix(lower, "vless://"):
		return parseUserInfoShareToClashProxy(raw, index, "vless")
	case strings.HasPrefix(lower, "trojan://"):
		return parseUserInfoShareToClashProxy(raw, index, "trojan")
	case strings.HasPrefix(lower, "ss://"):
		return parseSSShareToClashProxy(raw, index)
	default:
		return nil, false
	}
}

func parseVmessShareToClashProxy(raw string, index int) (map[string]interface{}, bool) {
	encoded := trimSharePrefix(raw, "vmess://")
	decoded, ok := decodeBase64Text(encoded)
	if !ok {
		return nil, false
	}

	var item map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &item); err != nil {
		return nil, false
	}

	server := getMapString(item, "add")
	port := getMapInt(item, "port")
	uuid := getMapString(item, "id")
	if server == "" || port <= 0 || uuid == "" {
		return nil, false
	}

	node := map[string]interface{}{
		"name":    firstNonEmpty(getMapString(item, "ps"), fmt.Sprintf("导入代理 %d", index+1)),
		"type":    "vmess",
		"server":  server,
		"port":    port,
		"uuid":    uuid,
		"alterId": getMapInt(item, "aid"),
		"cipher":  firstNonEmpty(getMapString(item, "scy"), getMapString(item, "cipher"), "auto"),
	}

	if tlsMode := strings.ToLower(getMapString(item, "tls")); tlsMode != "" && tlsMode != "none" {
		node["tls"] = true
		putStringIfNotEmpty(node, "servername", firstNonEmpty(getMapString(item, "sni"), getMapString(item, "host")))
	}
	putStringIfNotEmpty(node, "client-fingerprint", getMapString(item, "fp"))
	applyClashTransportOptions(node, firstNonEmpty(getMapString(item, "net"), getMapString(item, "network")), getMapString(item, "path"), getMapString(item, "host"), getMapString(item, "serviceName"))
	return node, true
}

func parseUserInfoShareToClashProxy(raw string, index int, protocol string) (map[string]interface{}, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}

	server := parsed.Hostname()
	port, _ := strconv.Atoi(parsed.Port())
	user := parsed.User.Username()
	if server == "" || port <= 0 || user == "" {
		return nil, false
	}

	q := parsed.Query()
	node := map[string]interface{}{
		"name":   shareNodeName(parsed, index),
		"type":   protocol,
		"server": server,
		"port":   port,
	}
	if protocol == "vless" {
		node["uuid"] = user
		putStringIfNotEmpty(node, "flow", q.Get("flow"))
	} else {
		node["password"] = user
	}

	security := strings.ToLower(q.Get("security"))
	if security == "tls" || security == "reality" {
		node["tls"] = true
		putStringIfNotEmpty(node, "servername", firstNonEmpty(q.Get("sni"), q.Get("peer"), q.Get("servername")))
		if security == "reality" {
			realityOpts := map[string]interface{}{}
			putStringIfNotEmpty(realityOpts, "public-key", q.Get("pbk"))
			putStringIfNotEmpty(realityOpts, "short-id", q.Get("sid"))
			if len(realityOpts) > 0 {
				node["reality-opts"] = realityOpts
			}
		}
	}
	putStringIfNotEmpty(node, "client-fingerprint", q.Get("fp"))
	if isTruthyQueryValue(q.Get("allowInsecure")) || isTruthyQueryValue(q.Get("skip-cert-verify")) {
		node["skip-cert-verify"] = true
	}

	network := firstNonEmpty(q.Get("type"), q.Get("network"))
	applyClashTransportOptions(node, network, q.Get("path"), q.Get("host"), q.Get("serviceName"))
	return node, true
}

func parseSSShareToClashProxy(raw string, index int) (map[string]interface{}, bool) {
	body := trimSharePrefix(raw, "ss://")
	fragment := ""
	if hashIndex := strings.Index(body, "#"); hashIndex >= 0 {
		fragment = body[hashIndex+1:]
		body = body[:hashIndex]
	}
	if queryIndex := strings.Index(body, "?"); queryIndex >= 0 {
		body = body[:queryIndex]
	}

	var method, password, server string
	var port int
	if strings.Contains(body, "@") {
		at := strings.LastIndex(body, "@")
		userPart := body[:at]
		hostPart := body[at+1:]
		if decoded, ok := decodeBase64Text(userPart); ok {
			userPart = decoded
		} else if unescaped, err := url.QueryUnescape(userPart); err == nil {
			userPart = unescaped
		}
		parts := strings.SplitN(userPart, ":", 2)
		if len(parts) != 2 {
			return nil, false
		}
		method = parts[0]
		password = parts[1]
		parsedHost, err := url.Parse("ss://" + hostPart)
		if err != nil {
			return nil, false
		}
		server = parsedHost.Hostname()
		port, _ = strconv.Atoi(parsedHost.Port())
	} else {
		decoded, ok := decodeBase64Text(body)
		if !ok {
			return nil, false
		}
		at := strings.LastIndex(decoded, "@")
		if at < 0 {
			return nil, false
		}
		parts := strings.SplitN(decoded[:at], ":", 2)
		if len(parts) != 2 {
			return nil, false
		}
		method = parts[0]
		password = parts[1]
		parsedHost, err := url.Parse("ss://" + decoded[at+1:])
		if err != nil {
			return nil, false
		}
		server = parsedHost.Hostname()
		port, _ = strconv.Atoi(parsedHost.Port())
	}

	if method == "" || password == "" || server == "" || port <= 0 {
		return nil, false
	}
	name := strings.TrimSpace(fragment)
	if unescaped, err := url.QueryUnescape(name); err == nil {
		name = strings.TrimSpace(unescaped)
	}
	if name == "" {
		name = fmt.Sprintf("导入代理 %d", index+1)
	}
	return map[string]interface{}{
		"name":     name,
		"type":     "ss",
		"server":   server,
		"port":     port,
		"cipher":   method,
		"password": password,
	}, true
}

func trimSharePrefix(raw string, prefix string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= len(prefix) && strings.EqualFold(raw[:len(prefix)], prefix) {
		return strings.TrimSpace(raw[len(prefix):])
	}
	return strings.TrimSpace(strings.TrimPrefix(raw, prefix))
}

func shareNodeName(parsed *url.URL, index int) string {
	name := strings.TrimSpace(parsed.Fragment)
	if unescaped, err := url.QueryUnescape(name); err == nil {
		name = strings.TrimSpace(unescaped)
	}
	if name == "" {
		name = fmt.Sprintf("导入代理 %d", index+1)
	}
	return name
}

func applyClashTransportOptions(node map[string]interface{}, network string, path string, host string, serviceName string) {
	network = strings.TrimSpace(strings.ToLower(network))
	if network == "" {
		return
	}
	node["network"] = network
	switch network {
	case "ws":
		wsOpts := map[string]interface{}{}
		putStringIfNotEmpty(wsOpts, "path", path)
		if strings.TrimSpace(host) != "" {
			wsOpts["headers"] = map[string]interface{}{
				"Host": strings.TrimSpace(host),
			}
		}
		if len(wsOpts) > 0 {
			node["ws-opts"] = wsOpts
		}
	case "grpc":
		grpcOpts := map[string]interface{}{}
		putStringIfNotEmpty(grpcOpts, "grpc-service-name", serviceName)
		if len(grpcOpts) > 0 {
			node["grpc-opts"] = grpcOpts
		}
	}
}

func putStringIfNotEmpty(m map[string]interface{}, key string, value string) {
	if strings.TrimSpace(value) != "" {
		m[key] = strings.TrimSpace(value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isTruthyQueryValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "1" || value == "true" || value == "yes"
}

func clashProxyCount(payload interface{}) int {
	if m := toStringMap(payload); m != nil {
		if arr, ok := m["proxies"].([]interface{}); ok {
			return len(arr)
		}
		if arr, ok := m["proxy"].([]interface{}); ok {
			return len(arr)
		}
		if arr, ok := m["Proxy"].([]interface{}); ok {
			return len(arr)
		}
	}
	if arr, ok := payload.([]interface{}); ok {
		return len(arr)
	}
	return 0
}

func extractClashDNSYAML(payload interface{}) string {
	m := toStringMap(payload)
	if m == nil {
		return ""
	}
	dnsRaw, exists := m["dns"]
	if !exists || dnsRaw == nil {
		return ""
	}
	data, err := yaml.Marshal(map[string]interface{}{
		"dns": dnsRaw,
	})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func suggestClashGroupName(payload interface{}, fallbackHost string) string {
	fallbackHost = strings.TrimSpace(fallbackHost)
	m := toStringMap(payload)
	if m != nil {
		if groups, ok := m["proxy-groups"].([]interface{}); ok {
			for _, item := range groups {
				if groupMap := toStringMap(item); groupMap != nil {
					if name := strings.TrimSpace(getMapString(groupMap, "name")); name != "" {
						return name
					}
				}
			}
		}
	}
	if strings.HasPrefix(strings.ToLower(fallbackHost), "www.") {
		fallbackHost = fallbackHost[4:]
	}
	return fallbackHost
}

func toStringMap(value interface{}) map[string]interface{} {
	switch m := value.(type) {
	case map[string]interface{}:
		return m
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, v := range m {
			key := fmt.Sprint(k)
			out[key] = v
		}
		return out
	default:
		return nil
	}
}

func getMapString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func getMapInt(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	value, ok := m[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(v)))
		return n
	}
}
