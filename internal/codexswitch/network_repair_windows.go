//go:build windows

package codexswitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const networkRepairScript = `
param(
  [Parameter(Mandatory = $true)]
  [string]$ResultPath
)

$ErrorActionPreference = "Stop"

function Add-UniqueInterface {
  param(
    [object[]]$List,
    [object]$Interface
  )

  foreach ($item in $List) {
    if ([int]$item.ifIndex -eq [int]$Interface.ifIndex) {
      return $List
    }
  }

  return @($List) + $Interface
}

function Add-UniqueInt {
  param(
    [int[]]$List,
    [int]$Value
  )

  foreach ($item in @($List)) {
    if ([int]$item -eq $Value) {
      return @($List)
    }
  }

  return @($List) + $Value
}

function Test-ContainsInt {
  param(
    [int[]]$List,
    [int]$Value
  )

  foreach ($item in @($List)) {
    if ([int]$item -eq $Value) {
      return $true
    }
  }

  return $false
}

function Get-InterfaceLabel {
  param([object]$Interface)
  return "$($Interface.InterfaceAlias) (ifIndex $($Interface.ifIndex))"
}

function Save-Result {
  param([object]$Payload)
  $Payload | ConvertTo-Json -Depth 5 | Set-Content -Path $ResultPath -Encoding UTF8
}

$result = [ordered]@{
  success = $false
  message = ""
  physicalInterfaces = @()
  vpnInterfaces = @()
  apiDns = @()
  chatgptDns = @()
  apiReachable = $false
  chatgptReachable = $false
  error = ""
}

try {
  $connected = @(
    Get-NetIPInterface -AddressFamily IPv4 |
      Where-Object {
        $_.ConnectionState -eq "Connected" -and
        $_.InterfaceAlias -notlike "Loopback*"
      }
  )
  if ($connected.Count -eq 0) {
    throw "No connected IPv4 interface was found."
  }

  $adapters = @{}
  Get-NetAdapter | ForEach-Object {
    $adapters[[int]$_.ifIndex] = $_
  }

  $configs = @{}
  Get-NetIPConfiguration | ForEach-Object {
    $configs[[int]$_.InterfaceIndex] = $_
  }

  $dnsByIndex = @{}
  Get-DnsClientServerAddress -AddressFamily IPv4 | ForEach-Object {
    $dnsByIndex[[int]$_.InterfaceIndex] = @($_.ServerAddresses)
  }

  $virtualPattern = "(?i)(vpn|virtual|tap|tun|wintun|wireguard|openvpn|anyconnect|globalprotect|fortinet|forti|ssl|pulse|sangfor|array|clash|v2ray|vmware|virtualbox|hyper-v|zerotier|tailscale)"
  $physicalCandidates = @()

  foreach ($iface in $connected) {
    $index = [int]$iface.ifIndex
    $adapter = $adapters[$index]
    $config = $configs[$index]
    $description = "$($iface.InterfaceAlias) $($adapter.InterfaceDescription)"
    $hasGateway = $false
    if ($config -and $config.IPv4DefaultGateway) {
      $hasGateway = @($config.IPv4DefaultGateway).Count -gt 0
    }
    $isHardware = $adapter -and $adapter.HardwareInterface
    $looksVirtual = $description -match $virtualPattern

    if ($isHardware -and -not $looksVirtual -and $hasGateway) {
      $physicalCandidates = Add-UniqueInterface $physicalCandidates $iface
    }
  }

  if ($physicalCandidates.Count -eq 0) {
    $defaultRouteIndexes = @()
    Get-NetRoute -AddressFamily IPv4 -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue |
      Where-Object { $_.NextHop -and $_.NextHop -ne "0.0.0.0" } |
      ForEach-Object { $defaultRouteIndexes = Add-UniqueInt $defaultRouteIndexes ([int]$_.ifIndex) }

    foreach ($iface in $connected) {
      $index = [int]$iface.ifIndex
      if (-not (Test-ContainsInt $defaultRouteIndexes $index)) {
        continue
      }

      $adapter = $adapters[$index]
      $description = "$($iface.InterfaceAlias) $($adapter.InterfaceDescription)"
      if ($description -notmatch $virtualPattern) {
        $physicalCandidates = Add-UniqueInterface $physicalCandidates $iface
      }
    }
  }

  if ($physicalCandidates.Count -eq 0) {
    throw "No physical internet egress interface was found."
  }

  $physicalIndexes = @()
  foreach ($iface in $physicalCandidates) {
    $index = [int]$iface.ifIndex
    Set-NetIPInterface -InterfaceIndex $index -AddressFamily IPv4 -AutomaticMetric Disabled -InterfaceMetric 10
    $physicalIndexes = Add-UniqueInt $physicalIndexes $index
    $result.physicalInterfaces += Get-InterfaceLabel $iface
  }

  $vpnCandidates = @()
  foreach ($iface in $connected) {
    $index = [int]$iface.ifIndex
    if (Test-ContainsInt $physicalIndexes $index) {
      continue
    }

    $adapter = $adapters[$index]
    $config = $configs[$index]
    $description = "$($iface.InterfaceAlias) $($adapter.InterfaceDescription)"
    $servers = @()
    if ($dnsByIndex.ContainsKey($index)) {
      $servers = @($dnsByIndex[$index])
    }
    $hasDNS = $servers.Count -gt 0
    $hasGateway = $false
    if ($config -and $config.IPv4DefaultGateway) {
      $hasGateway = @($config.IPv4DefaultGateway).Count -gt 0
    }
    $looksVirtual = $description -match $virtualPattern
    $hasHigherPriority = [int]$iface.InterfaceMetric -lt 10

    if ($looksVirtual -or ($hasDNS -and -not $hasGateway) -or $hasHigherPriority) {
      $vpnCandidates = Add-UniqueInterface $vpnCandidates $iface
    }
  }

  foreach ($iface in $vpnCandidates) {
    $index = [int]$iface.ifIndex
    Set-NetIPInterface -InterfaceIndex $index -AddressFamily IPv4 -AutomaticMetric Disabled -InterfaceMetric 80
    $result.vpnInterfaces += Get-InterfaceLabel $iface
  }

  Clear-DnsClientCache
  Start-Sleep -Milliseconds 500

  try {
    $result.apiDns = @(
      Resolve-DnsName api.openai.com -Type A -ErrorAction Stop |
        Where-Object { $_.IPAddress } |
        Select-Object -ExpandProperty IPAddress
    )
  } catch {
    $result.apiDns = @()
  }

  try {
    $result.chatgptDns = @(
      Resolve-DnsName chatgpt.com -Type A -ErrorAction Stop |
        Where-Object { $_.IPAddress } |
        Select-Object -ExpandProperty IPAddress
    )
  } catch {
    $result.chatgptDns = @()
  }

  try {
    $result.apiReachable = [bool](Test-NetConnection api.openai.com -Port 443 -InformationLevel Quiet)
  } catch {
    $result.apiReachable = $false
  }

  try {
    $result.chatgptReachable = [bool](Test-NetConnection chatgpt.com -Port 443 -InformationLevel Quiet)
  } catch {
    $result.chatgptReachable = $false
  }

  $result.success = $result.apiReachable -and $result.chatgptReachable
  if ($result.success) {
    $result.message = "Network repair completed. Codex domains are reachable."
  } else {
    $result.message = "Interface priority was adjusted, but Codex domains are still not fully reachable. Check VPN DNS or firewall policy."
  }
} catch {
  $result.success = $false
  $result.error = $_.Exception.Message
  if ($_.InvocationInfo -and $_.InvocationInfo.ScriptLineNumber) {
    $result.error = "$($result.error) (line $($_.InvocationInfo.ScriptLineNumber))"
  }
  $result.message = "Network repair failed: $($result.error)"
}

Save-Result $result
if (-not $result.success) {
  exit 2
}
`

func repairNetwork() (NetworkRepairResult, error) {
	tempDir, err := os.MkdirTemp("", "codexswitch-network-repair-*")
	if err != nil {
		return NetworkRepairResult{}, fmt.Errorf("create network repair temp directory failed: %w", err)
	}
	defer os.RemoveAll(tempDir)

	scriptPath := filepath.Join(tempDir, "repair-network.ps1")
	resultPath := filepath.Join(tempDir, "repair-result.json")
	if err := os.WriteFile(scriptPath, []byte(networkRepairScript), 0o600); err != nil {
		return NetworkRepairResult{}, fmt.Errorf("write network repair script failed: %w", err)
	}

	command := fmt.Sprintf(
		"$p = Start-Process -FilePath %s -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-File',%s,'-ResultPath',%s) -Verb RunAs -WindowStyle Hidden -Wait -PassThru; if ($null -ne $p.ExitCode) { exit $p.ExitCode }",
		powershellQuote("powershell.exe"),
		powershellQuote(scriptPath),
		powershellQuote(resultPath),
	)
	output, runErr := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command).CombinedOutput()

	data, readErr := os.ReadFile(resultPath)
	if readErr != nil {
		if runErr != nil {
			return NetworkRepairResult{}, fmt.Errorf("network repair did not complete or administrator approval was canceled: %w%s", runErr, formatCommandOutput(output))
		}
		return NetworkRepairResult{}, fmt.Errorf("network repair did not produce a result file: %w%s", readErr, formatCommandOutput(output))
	}

	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var result NetworkRepairResult
	if err := json.Unmarshal(data, &result); err != nil {
		return NetworkRepairResult{}, fmt.Errorf("parse network repair result failed: %w", err)
	}
	if result.Message == "" {
		result.Message = "Network repair executed."
	}
	return result, nil
}

func powershellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func formatCommandOutput(output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return ""
	}
	return ": " + trimmed
}
