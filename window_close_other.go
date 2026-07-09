//go:build !windows

package main

func installCloseToTrayHandler(*App) {
}

func shouldHandleCloseToTrayInBeforeClose() bool {
	return true
}

func showMainWindowNative() bool {
	return false
}
