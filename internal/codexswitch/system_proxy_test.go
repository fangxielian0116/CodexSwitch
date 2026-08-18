package codexswitch

import (
	"net/http"
	"testing"
)

func TestParseSystemProxySettingsSupportsPerSchemeProxy(t *testing.T) {
	route, err := parseSystemProxySettings(systemProxySettings{
		ProxyServer:   "http=127.0.0.1:7890;https=127.0.0.1:7891",
		ProxyOverride: "localhost;*.internal.example.com;<local>",
	})
	if err != nil {
		t.Fatalf("parseSystemProxySettings returned error: %v", err)
	}

	httpProxy, _ := route.httpProxy, route.httpsProxy
	if got := httpProxy.String(); got != "http://127.0.0.1:7890" {
		t.Fatalf("unexpected http proxy: %s", got)
	}
	if got := route.httpsProxy.String(); got != "http://127.0.0.1:7891" {
		t.Fatalf("unexpected https proxy: %s", got)
	}

	for _, host := range []string{"localhost", "service.internal.example.com", "workstation"} {
		if !shouldBypassSystemProxy(host, route.bypass) {
			t.Fatalf("expected %q to bypass system proxy", host)
		}
	}
	if shouldBypassSystemProxy("api.openai.com", route.bypass) {
		t.Fatal("did not expect api.openai.com to bypass system proxy")
	}
}

func TestParseSystemProxySettingsUsesDefaultProxyForBothSchemes(t *testing.T) {
	route, err := parseSystemProxySettings(systemProxySettings{ProxyServer: "127.0.0.1:7890"})
	if err != nil {
		t.Fatalf("parseSystemProxySettings returned error: %v", err)
	}
	if route.httpProxy == nil || route.httpsProxy == nil {
		t.Fatal("expected default proxy for both HTTP and HTTPS")
	}
	if route.httpProxy.String() != route.httpsProxy.String() {
		t.Fatalf("expected same default proxy, got %s and %s", route.httpProxy, route.httpsProxy)
	}
}

func TestSystemProxyFuncReturnsProxyForHTTPSRequests(t *testing.T) {
	proxy, enabled, err := systemProxyFuncForSettings(systemProxySettings{ProxyServer: "127.0.0.1:7890"})
	if err != nil {
		t.Fatalf("proxyFuncForTest returned error: %v", err)
	}
	if !enabled {
		t.Fatal("expected system proxy to be enabled")
	}

	request, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}
	proxyURL, err := proxy(request)
	if err != nil {
		t.Fatalf("proxy function returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("unexpected proxy URL: %v", proxyURL)
	}
}
