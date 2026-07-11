//go:build windows

package codexswitch

import (
	"fmt"
	"os/exec"
	"strings"
)

const restartCodexWindowsScript = `
$ErrorActionPreference = 'Stop'

function Get-CodexProcess {
  Get-CimInstance Win32_Process |
    Where-Object {
      $_.Name -in @('Codex.exe', 'ChatGPT.exe') -and
      $_.ExecutablePath -match '\\app\\(?:Codex|ChatGPT)\.exe$' -and
      $_.CommandLine -notmatch '\s--type='
    } |
    Sort-Object ProcessId |
    Select-Object -First 1
}

function Get-CodexAppUserModelId([string]$exePath) {
  if ([string]::IsNullOrWhiteSpace($exePath)) {
    return $null
  }

  $normalized = $exePath -replace '/', '\'
  if ($normalized -notmatch '\\WindowsApps\\([^\\]+)\\app\\(?:Codex|ChatGPT)\.exe$') {
    return $null
  }

  $packageFullName = $Matches[1]
  if ($packageFullName -notmatch '^(?<name>.+?)_\d+(?:\.\d+){3}_[^_]+_[^_]*_(?<publisher>[^_]+)$') {
    return $null
  }

  return "$($Matches['name'])_$($Matches['publisher'])!App"
}

function Start-CodexAppUserModelId([string]$appUserModelId) {
  if ([string]::IsNullOrWhiteSpace($appUserModelId)) {
    return $false
  }

  Start-Process -FilePath 'explorer.exe' -ArgumentList "shell:AppsFolder\$appUserModelId" | Out-Null
  return $true
}

function Start-CodexApp([string]$exePath) {
  $appUserModelId = Get-CodexAppUserModelId $exePath
  if (Start-CodexAppUserModelId $appUserModelId) {
    return
  }

  if ([string]::IsNullOrWhiteSpace($exePath) -or -not (Test-Path -LiteralPath $exePath)) {
    throw '未找到 Codex 可执行文件，无法启动'
  }
  Start-Process -FilePath $exePath -WorkingDirectory (Split-Path -Parent $exePath) | Out-Null
}

function Start-KnownCodexApp {
  $candidates = @()

  try {
    Get-StartApps |
      Where-Object { $_.AppID -like 'OpenAI.Codex_*' } |
      ForEach-Object {
        if (-not [string]::IsNullOrWhiteSpace($_.AppID)) {
          $candidates += $_.AppID
        }
      }
  } catch {
  }

  $candidates += 'OpenAI.Codex_2p2nqsd0c76g0!App'

  foreach ($candidate in ($candidates | Select-Object -Unique)) {
    try {
      if (Start-CodexAppUserModelId $candidate) {
        return $true
      }
    } catch {
    }
  }

  return $false
}

function Wait-CodexProcess([int]$timeoutSeconds) {
  $deadline = (Get-Date).AddSeconds($timeoutSeconds)
  while ((Get-Date) -le $deadline) {
    $process = Get-CodexProcess
    if ($null -ne $process) {
      return $process
    }
    Start-Sleep -Milliseconds 300
  }
  return $null
}

function Clear-StaleTrayIcons {
  Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;

public static class TrayIconSweeper {
  [StructLayout(LayoutKind.Sequential)]
  public struct RECT {
    public int Left;
    public int Top;
    public int Right;
    public int Bottom;
  }

  [DllImport("user32.dll", SetLastError = true)]
  private static extern IntPtr FindWindow(string lpClassName, string lpWindowName);

  [DllImport("user32.dll", SetLastError = true)]
  private static extern IntPtr FindWindowEx(IntPtr parent, IntPtr childAfter, string className, string windowName);

  [DllImport("user32.dll", SetLastError = true)]
  private static extern bool GetClientRect(IntPtr hWnd, out RECT rect);

  [DllImport("user32.dll", SetLastError = true)]
  private static extern IntPtr SendMessage(IntPtr hWnd, int msg, IntPtr wParam, IntPtr lParam);

  private const int WM_MOUSEMOVE = 0x0200;

  private static IntPtr MakeLParam(int x, int y) {
    return (IntPtr)((y << 16) | (x & 0xffff));
  }

  private static void Sweep(IntPtr hWnd) {
    if (hWnd == IntPtr.Zero) {
      return;
    }
    RECT rect;
    if (!GetClientRect(hWnd, out rect)) {
      return;
    }
    for (int y = 0; y < rect.Bottom; y += 6) {
      for (int x = 0; x < rect.Right; x += 6) {
        SendMessage(hWnd, WM_MOUSEMOVE, IntPtr.Zero, MakeLParam(x, y));
      }
    }
  }

  public static void Refresh() {
    IntPtr shell = FindWindow("Shell_TrayWnd", null);
    IntPtr tray = FindWindowEx(shell, IntPtr.Zero, "TrayNotifyWnd", null);
    IntPtr pager = FindWindowEx(tray, IntPtr.Zero, "SysPager", null);
    Sweep(FindWindowEx(pager, IntPtr.Zero, "ToolbarWindow32", null));
    Sweep(FindWindowEx(tray, IntPtr.Zero, "ToolbarWindow32", null));

    IntPtr overflow = FindWindow("NotifyIconOverflowWindow", null);
    Sweep(FindWindowEx(overflow, IntPtr.Zero, "ToolbarWindow32", null));
  }
}
"@ -ErrorAction SilentlyContinue

  [TrayIconSweeper]::Refresh()
}

$process = Get-CodexProcess
$oldPid = $null

if ($null -ne $process) {
  $oldPid = $process.ProcessId
  $exe = $process.ExecutablePath
  $closed = $false
  $forced = $false
  $runningProcess = Get-Process -Id $oldPid -ErrorAction SilentlyContinue

  try {
    if ($null -ne $runningProcess -and $runningProcess.MainWindowHandle -ne 0) {
      $closed = $runningProcess.CloseMainWindow()
    }
  } catch {
    $closed = $false
  }

  if ($closed -and $null -ne $runningProcess) {
    [void]$runningProcess.WaitForExit(3000)
  }

  if ($null -ne (Get-Process -Id $oldPid -ErrorAction SilentlyContinue)) {
    Stop-Process -Id $oldPid -Force
    $forced = $true
  }

  $deadline = (Get-Date).AddSeconds(5)
  while ($null -ne (Get-Process -Id $oldPid -ErrorAction SilentlyContinue)) {
    if ((Get-Date) -gt $deadline) {
      throw "Codex 旧进程未退出：$oldPid"
    }
    Start-Sleep -Milliseconds 200
  }

  if ($forced) {
    Clear-StaleTrayIcons
  }

  Start-CodexApp $exe
} elseif (-not (Start-KnownCodexApp)) {
  throw '未找到正在运行的 Codex 主进程，也未找到可用的 Codex 启动入口'
}

$newProcess = Wait-CodexProcess 10
if ($null -eq $newProcess) {
  throw 'Codex 启动后未检测到运行中的主进程'
}
if ($null -ne $oldPid -and $newProcess.ProcessId -eq $oldPid) {
  throw "Codex 进程 ID 未变化：$oldPid"
}
`

func restartCodexProcess() error {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", restartCodexWindowsScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("重启 Codex 失败: %w: %s", err, message)
		}
		return fmt.Errorf("重启 Codex 失败: %w", err)
	}
	return nil
}
