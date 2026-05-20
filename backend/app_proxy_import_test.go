package backend

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNormalizeClashSubscriptionContentDecodesBase64ShareSubscription(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"VMess 节点","add":"vmess.example.com","port":"443","id":"22222222-2222-4222-8222-222222222222","aid":"0","scy":"auto","net":"ws","type":"none","host":"vmess.example.com","path":"/ws","tls":"tls","sni":"vmess.example.com","fp":"chrome"}`
	vmess := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))
	vless := "vless://11111111-1111-4111-8111-111111111111@example.com:443?encryption=none&security=tls&sni=edge.example.com&fp=firefox&type=ws&host=edge.example.com&path=%2Fvless%3Fed%3D2560#US%20VLESS"
	trojan := "trojan://top-secret@example.net:443?security=tls&sni=trojan.example.net&type=ws&host=trojan.example.net&path=%2Ftrojan#Trojan%20Node"

	subscription := strings.Join([]string{vless, vmess, trojan}, "\n")
	encoded := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(subscription)), "=")

	content, payload, err := normalizeClashSubscriptionContent([]byte(encoded))
	if err != nil {
		t.Fatalf("normalizeClashSubscriptionContent returned error: %v", err)
	}
	if !strings.Contains(content, "proxies:") {
		t.Fatalf("normalized content should be Clash YAML, got: %s", content)
	}
	if got := clashProxyCount(payload); got != 3 {
		t.Fatalf("proxy count = %d, want 3", got)
	}

	root := toStringMap(payload)
	if root == nil {
		t.Fatal("payload root is not a map")
	}
	proxies, ok := root["proxies"].([]interface{})
	if !ok || len(proxies) != 3 {
		t.Fatalf("payload proxies length = %d, want 3", len(proxies))
	}

	first := toStringMap(proxies[0])
	if first == nil {
		t.Fatal("first proxy is not a map")
	}
	if got := getMapString(first, "type"); got != "vless" {
		t.Fatalf("first proxy type = %q, want vless", got)
	}
	if got := getMapString(first, "name"); got != "US VLESS" {
		t.Fatalf("first proxy name = %q, want US VLESS", got)
	}
	wsOpts := toStringMap(first["ws-opts"])
	if wsOpts == nil || getMapString(wsOpts, "path") != "/vless?ed=2560" {
		t.Fatalf("first proxy ws path not decoded correctly: %#v", wsOpts)
	}

	second := toStringMap(proxies[1])
	if second == nil {
		t.Fatal("second proxy is not a map")
	}
	if got := getMapString(second, "type"); got != "vmess" {
		t.Fatalf("second proxy type = %q, want vmess", got)
	}
	if got := getMapString(second, "name"); got != "VMess 节点" {
		t.Fatalf("second proxy name = %q, want VMess 节点", got)
	}
}

