package proxy

import "testing"

func TestBuildSingBoxOutboundHysteria2AliasURI(t *testing.T) {
	out, err := BuildSingBoxOutbound("hy2://secret@example.com:8443?sni=edge.example.com&insecure=1&obfs-password=obfs-secret&upmbps=50&downmbps=100#HY2")
	if err != nil {
		t.Fatalf("BuildSingBoxOutbound returned error: %v", err)
	}
	if got := getMapString(out, "type"); got != "hysteria2" {
		t.Fatalf("type = %q, want hysteria2", got)
	}
	if got := getMapString(out, "server"); got != "example.com" {
		t.Fatalf("server = %q, want example.com", got)
	}
	if got := getMapInt(out, "server_port"); got != 8443 {
		t.Fatalf("server_port = %d, want 8443", got)
	}
	if got := getMapString(out, "password"); got != "secret" {
		t.Fatalf("password = %q, want secret", got)
	}
	if got := getMapInt(out, "up_mbps"); got != 50 {
		t.Fatalf("up_mbps = %d, want 50", got)
	}
	if got := getMapInt(out, "down_mbps"); got != 100 {
		t.Fatalf("down_mbps = %d, want 100", got)
	}
	tls := toStringMap(out["tls"])
	if tls == nil {
		t.Fatal("tls is not a map")
	}
	if got := getMapString(tls, "server_name"); got != "edge.example.com" {
		t.Fatalf("tls.server_name = %q, want edge.example.com", got)
	}
	if !getMapBool(tls, "insecure") {
		t.Fatal("tls.insecure = false, want true")
	}
	obfs := toStringMap(out["obfs"])
	if obfs == nil || getMapString(obfs, "password") != "obfs-secret" {
		t.Fatalf("obfs not mapped correctly: %#v", obfs)
	}
}

func TestBuildSingBoxOutboundAnyTLSURI(t *testing.T) {
	out, err := BuildSingBoxOutbound("anytls://secret@example.com:443?sni=edge.example.com&insecure=1#AnyTLS")
	if err != nil {
		t.Fatalf("BuildSingBoxOutbound returned error: %v", err)
	}
	if got := getMapString(out, "type"); got != "anytls" {
		t.Fatalf("type = %q, want anytls", got)
	}
	if got := getMapString(out, "server"); got != "example.com" {
		t.Fatalf("server = %q, want example.com", got)
	}
	if got := getMapInt(out, "server_port"); got != 443 {
		t.Fatalf("server_port = %d, want 443", got)
	}
	if got := getMapString(out, "password"); got != "secret" {
		t.Fatalf("password = %q, want secret", got)
	}
	tls := toStringMap(out["tls"])
	if tls == nil {
		t.Fatal("tls is not a map")
	}
	if got := getMapString(tls, "server_name"); got != "edge.example.com" {
		t.Fatalf("tls.server_name = %q, want edge.example.com", got)
	}
	if !getMapBool(tls, "insecure") {
		t.Fatal("tls.insecure = false, want true")
	}
}

func TestBuildSingBoxOutboundAnyTLSClashYAML(t *testing.T) {
	out, err := BuildSingBoxOutbound(`proxies:
  - name: anytls-node
    type: anytls
    server: example.net
    port: 8443
    password: secret
    sni: edge.example.net
    skip-cert-verify: true
`)
	if err != nil {
		t.Fatalf("BuildSingBoxOutbound returned error: %v", err)
	}
	if got := getMapString(out, "type"); got != "anytls" {
		t.Fatalf("type = %q, want anytls", got)
	}
	if got := getMapString(out, "server"); got != "example.net" {
		t.Fatalf("server = %q, want example.net", got)
	}
	if got := getMapInt(out, "server_port"); got != 8443 {
		t.Fatalf("server_port = %d, want 8443", got)
	}
	tls := toStringMap(out["tls"])
	if tls == nil || getMapString(tls, "server_name") != "edge.example.net" || !getMapBool(tls, "insecure") {
		t.Fatalf("tls fields not mapped correctly: %#v", tls)
	}
}
