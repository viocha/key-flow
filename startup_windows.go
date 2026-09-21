//go:build windows

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

const startupTaskName = "Key Flow Startup"

func startupTaskEnabled() (bool, error) {
	executable, err := currentExecutablePath()
	if err != nil {
		return false, err
	}

	script := "$task = Get-ScheduledTask -TaskName $env:KEY_FLOW_TASK -ErrorAction SilentlyContinue\n" +
		"if ($null -eq $task) { Write-Output \"false\"; exit 0 }\n" +
		"$action = $task.Actions | Select-Object -First 1\n" +
		"$arguments = if ($null -eq $action.Arguments) { \"\" } else { $action.Arguments }\n" +
		"$matches = $task.State -ne \"Disabled\" -and " +
		"[string]::Equals($action.Execute, $env:KEY_FLOW_EXE, [System.StringComparison]::OrdinalIgnoreCase) -and " +
		"[string]::Equals($arguments.Trim(), \"--hidden\", [System.StringComparison]::Ordinal)\n" +
		"if ($matches) { Write-Output \"true\" } else { Write-Output \"false\" }\n"

	output, err := runStartupPowerShell(script, executable)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(output), "true"), nil
}

func setStartupTaskEnabled(enabled bool) error {
	executable, err := currentExecutablePath()
	if err != nil {
		return err
	}

	if !enabled {
		script := "Unregister-ScheduledTask -TaskName $env:KEY_FLOW_TASK -Confirm:$false -ErrorAction SilentlyContinue\n"
		_, err := runStartupPowerShell(script, executable)
		return err
	}

	script := "$identity = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name\n" +
		"$action = New-ScheduledTaskAction -Execute $env:KEY_FLOW_EXE -Argument \"--hidden\"\n" +
		"$trigger = New-ScheduledTaskTrigger -AtLogOn -User $identity\n" +
		"$principal = New-ScheduledTaskPrincipal -UserId $identity -LogonType Interactive -RunLevel Highest\n" +
		"$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -MultipleInstances IgnoreNew -ExecutionTimeLimit ([TimeSpan]::Zero)\n" +
		"Register-ScheduledTask -TaskName $env:KEY_FLOW_TASK -Action $action -Trigger $trigger -Principal $principal -Settings $settings " +
		"-Description \"Key Flow 开机自启（管理员权限，仅托盘）\" -Force | Out-Null\n"

	_, err = runStartupPowerShell(script, executable)
	return err
}

func currentExecutablePath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("读取程序路径失败: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("规范化程序路径失败: %w", err)
	}
	return executable, nil
}

func runStartupPowerShell(script, executable string) (string, error) {
	encoded := encodePowerShell(script)
	command := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-EncodedCommand", encoded,
	)
	command.Env = append(
		os.Environ(),
		"KEY_FLOW_TASK="+startupTaskName,
		"KEY_FLOW_EXE="+executable,
	)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("配置开机自启失败: %s", message)
	}
	return stdout.String(), nil
}

func encodePowerShell(script string) string {
	encoded := utf16.Encode([]rune(script))
	raw := make([]byte, len(encoded)*2)
	for i, value := range encoded {
		raw[i*2] = byte(value)
		raw[i*2+1] = byte(value >> 8)
	}
	return base64.StdEncoding.EncodeToString(raw)
}
