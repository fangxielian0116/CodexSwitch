package codexswitch

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

func newDefaultHTTPClient(logger *slog.Logger) *http.Client {
	client := &http.Client{Timeout: 10 * time.Second}
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return client
	}

	transport = transport.Clone()
	proxy, enabled, err := systemProxyFunc()
	if err != nil {
		logger.Warn("读取系统代理失败，将继续使用环境变量代理", "error", err)
	} else if enabled {
		transport.Proxy = proxy
		logger.Info("已启用 Windows 静态系统代理")
	}
	client.Transport = transport
	return client
}

type systemProxySettings struct {
	ProxyServer   string
	ProxyOverride string
}

type systemProxyRoute struct {
	httpProxy  *url.URL
	httpsProxy *url.URL
	bypass     []string
}

func systemProxyFunc() (func(*http.Request) (*url.URL, error), bool, error) {
	settings, err := readSystemProxySettings()
	if err != nil {
		return nil, false, err
	}
	return systemProxyFuncForSettings(settings)
}

func systemProxyFuncForSettings(settings systemProxySettings) (func(*http.Request) (*url.URL, error), bool, error) {
	route, err := parseSystemProxySettings(settings)
	if err != nil {
		return nil, false, err
	}
	if route.httpProxy == nil && route.httpsProxy == nil {
		return nil, false, nil
	}

	return func(req *http.Request) (*url.URL, error) {
		if req == nil || req.URL == nil || shouldBypassSystemProxy(req.URL.Hostname(), route.bypass) {
			return nil, nil
		}

		switch strings.ToLower(req.URL.Scheme) {
		case "https":
			return route.httpsProxy, nil
		case "http":
			return route.httpProxy, nil
		default:
			return nil, nil
		}
	}, true, nil
}

func parseSystemProxySettings(settings systemProxySettings) (systemProxyRoute, error) {
	var route systemProxyRoute
	var defaultProxy *url.URL

	for _, rawEntry := range strings.Split(settings.ProxyServer, ";") {
		entry := strings.TrimSpace(rawEntry)
		if entry == "" {
			continue
		}

		name, value := "", entry
		if separator := strings.IndexByte(entry, '='); separator >= 0 {
			name = strings.ToLower(strings.TrimSpace(entry[:separator]))
			value = strings.TrimSpace(entry[separator+1:])
		}

		proxyURL, err := parseSystemProxyURL(value)
		if err != nil {
			return systemProxyRoute{}, err
		}
		if proxyURL == nil {
			continue
		}

		switch name {
		case "http":
			route.httpProxy = proxyURL
		case "https":
			route.httpsProxy = proxyURL
		case "", "socks":
			defaultProxy = proxyURL
		}
	}

	if route.httpProxy == nil {
		route.httpProxy = defaultProxy
	}
	if route.httpsProxy == nil {
		route.httpsProxy = defaultProxy
	}

	for _, rawPattern := range strings.Split(settings.ProxyOverride, ";") {
		if pattern := strings.TrimSpace(rawPattern); pattern != "" {
			route.bypass = append(route.bypass, pattern)
		}
	}

	return route, nil
}

func parseSystemProxyURL(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}

	proxyURL, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("解析系统代理地址失败: %w", err)
	}
	if proxyURL.Host == "" {
		return nil, fmt.Errorf("系统代理地址缺少主机名")
	}

	switch strings.ToLower(proxyURL.Scheme) {
	case "http", "https", "socks5", "socks5h":
		return proxyURL, nil
	default:
		return nil, fmt.Errorf("系统代理协议 %q 不受支持", proxyURL.Scheme)
	}
}

func shouldBypassSystemProxy(host string, patterns []string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return true
	}

	for _, rawPattern := range patterns {
		pattern := strings.TrimSpace(strings.ToLower(rawPattern))
		if pattern == "" {
			continue
		}
		if pattern == "<local>" && !strings.Contains(host, ".") {
			return true
		}
		pattern = strings.TrimPrefix(pattern, "http://")
		pattern = strings.TrimPrefix(pattern, "https://")
		if separator := strings.LastIndexByte(pattern, ':'); separator > -1 && !strings.Contains(pattern[separator+1:], "]") {
			pattern = pattern[:separator]
		}
		if matched, _ := path.Match(pattern, host); matched {
			return true
		}
	}

	return false
}
