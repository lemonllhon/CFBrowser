package backend

import "testing"

func TestProxyNeedsBrowserAuthBridge(t *testing.T) {
	tests := []struct {
		name     string
		proxyURL string
		want     bool
	}{
		{name: "http with auth", proxyURL: "http://user:pass@127.0.0.1:8080", want: true},
		{name: "https with auth", proxyURL: "https://user:pass@127.0.0.1:8443", want: true},
		{name: "socks5 with auth", proxyURL: "socks5://user:pass@127.0.0.1:1080", want: true},
		{name: "http without auth", proxyURL: "http://127.0.0.1:8080", want: false},
		{name: "https without auth", proxyURL: "https://127.0.0.1:8443", want: false},
		{name: "socks5 without auth", proxyURL: "socks5://127.0.0.1:1080", want: false},
		{name: "direct", proxyURL: "direct://", want: false},
		{name: "unsupported with auth", proxyURL: "vmess://user:pass@127.0.0.1:1080", want: false},
		{name: "invalid", proxyURL: "://bad", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proxyNeedsBrowserAuthBridge(tt.proxyURL); got != tt.want {
				t.Fatalf("proxyNeedsBrowserAuthBridge(%q) = %v, want %v", tt.proxyURL, got, tt.want)
			}
		})
	}
}
