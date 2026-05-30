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
	hysteria2 := "hy2://hy-secret@hy.example.org:8443?sni=hy.example.org&insecure=1&obfs-password=obfs-secret&upmbps=50&downmbps=100#HY2%20Node"
	anytls := "anytls://any-secret@any.example.org:443?sni=any.example.org&insecure=1#AnyTLS%20Node"

	subscription := strings.Join([]string{vless, vmess, trojan, hysteria2, anytls}, "\n")
	encoded := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(subscription)), "=")

	content, payload, err := normalizeClashSubscriptionContent([]byte(encoded))
	if err != nil {
		t.Fatalf("normalizeClashSubscriptionContent returned error: %v", err)
	}
	if !strings.Contains(content, "proxies:") {
		t.Fatalf("normalized content should be Clash YAML, got: %s", content)
	}
	if got := clashProxyCount(payload); got != 5 {
		t.Fatalf("proxy count = %d, want 5", got)
	}

	root := toStringMap(payload)
	if root == nil {
		t.Fatal("payload root is not a map")
	}
	proxies, ok := root["proxies"].([]interface{})
	if !ok || len(proxies) != 5 {
		t.Fatalf("payload proxies length = %d, want 5", len(proxies))
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

	fourth := toStringMap(proxies[3])
	if fourth == nil {
		t.Fatal("fourth proxy is not a map")
	}
	if got := getMapString(fourth, "type"); got != "hysteria2" {
		t.Fatalf("fourth proxy type = %q, want hysteria2", got)
	}
	if got := getMapString(fourth, "name"); got != "HY2 Node" {
		t.Fatalf("fourth proxy name = %q, want HY2 Node", got)
	}
	if got := getMapString(fourth, "password"); got != "hy-secret" {
		t.Fatalf("fourth proxy password = %q, want hy-secret", got)
	}
	if got := getMapString(fourth, "obfs-password"); got != "obfs-secret" {
		t.Fatalf("fourth proxy obfs-password = %q, want obfs-secret", got)
	}

	fifth := toStringMap(proxies[4])
	if fifth == nil {
		t.Fatal("fifth proxy is not a map")
	}
	if got := getMapString(fifth, "type"); got != "anytls" {
		t.Fatalf("fifth proxy type = %q, want anytls", got)
	}
	if got := getMapString(fifth, "name"); got != "AnyTLS Node" {
		t.Fatalf("fifth proxy name = %q, want AnyTLS Node", got)
	}
	if got := getMapString(fifth, "password"); got != "any-secret" {
		t.Fatalf("fifth proxy password = %q, want any-secret", got)
	}
}
