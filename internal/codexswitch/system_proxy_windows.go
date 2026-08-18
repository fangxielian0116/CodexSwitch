//go:build windows

package codexswitch

import "golang.org/x/sys/windows/registry"

const windowsInternetSettingsKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func readSystemProxySettings() (systemProxySettings, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsInternetSettingsKey, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return systemProxySettings{}, nil
		}
		return systemProxySettings{}, err
	}
	defer key.Close()

	proxyEnabled, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		if err == registry.ErrNotExist {
			return systemProxySettings{}, nil
		}
		return systemProxySettings{}, err
	}
	if proxyEnabled == 0 {
		return systemProxySettings{}, nil
	}

	proxyServer, _, err := key.GetStringValue("ProxyServer")
	if err != nil && err != registry.ErrNotExist {
		return systemProxySettings{}, err
	}
	proxyOverride, _, err := key.GetStringValue("ProxyOverride")
	if err != nil && err != registry.ErrNotExist {
		return systemProxySettings{}, err
	}

	return systemProxySettings{
		ProxyServer:   proxyServer,
		ProxyOverride: proxyOverride,
	}, nil
}
