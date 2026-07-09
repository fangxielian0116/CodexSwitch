//go:build !windows

package codexswitch

import (
	"fmt"
	"os/exec"
	"runtime"
)

func restartCodexProcess() error {
	if runtime.GOOS == "darwin" {
		return restartCodexOnDarwin()
	}
	return fmt.Errorf("自动重启 Codex 目前仅支持 Windows 和 macOS")
}

func restartCodexOnDarwin() error {
	const script = `
tell application "System Events"
  if exists process "Codex" then
    tell application "Codex" to quit
    delay 1
  end if
end tell
tell application "Codex" to activate
`
	if output, err := exec.Command("osascript", "-e", script).CombinedOutput(); err != nil {
		return fmt.Errorf("重启 Codex 失败: %w: %s", err, string(output))
	}
	return nil
}
