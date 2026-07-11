//go:build windows

package codexswitch

import (
	"strings"
	"testing"
)

func TestRestartCodexWindowsScriptSupportsDesktopExecutableNames(t *testing.T) {
	for _, executableName := range []string{"Codex.exe", "ChatGPT.exe"} {
		if !strings.Contains(restartCodexWindowsScript, executableName) {
			t.Errorf("restart script does not support %s", executableName)
		}
	}

	if !strings.Contains(restartCodexWindowsScript, `\\app\\(?:Codex|ChatGPT)\.exe$`) {
		t.Error("restart script does not recognize both desktop executable paths")
	}
}
