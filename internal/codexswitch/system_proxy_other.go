//go:build !windows

package codexswitch

func readSystemProxySettings() (systemProxySettings, error) {
	return systemProxySettings{}, nil
}
