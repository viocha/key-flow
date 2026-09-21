//go:build windows

package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

const (
	startupTaskName                  = "Key Flow Startup"
	taskCreateOrUpdate               = 6
	taskLogonNone                    = 0
	taskSchedulerFileNotFoundHResult = 0x80070002
)

func startupTaskEnabled() (bool, error) {
	return withTaskSchedulerRoot(func(root *ole.IDispatch) (bool, error) {
		taskVariant, err := oleutil.CallMethod(root, "GetTask", startupTaskName)
		if isTaskSchedulerFileNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("读取开机自启任务失败: %w", err)
		}

		task := taskVariant.ToIDispatch()
		if task == nil {
			return false, errors.New("读取开机自启任务失败: 任务对象无效")
		}
		defer task.Release()

		enabledVariant, err := oleutil.GetProperty(task, "Enabled")
		if err != nil {
			return false, fmt.Errorf("读取开机自启任务状态失败: %w", err)
		}
		defer enabledVariant.Clear()
		return enabledVariant.Val != 0, nil
	})
}

func setStartupTaskEnabled(enabled bool) error {
	_, err := withTaskSchedulerRoot(func(root *ole.IDispatch) (struct{}, error) {
		if !enabled {
			result, err := oleutil.CallMethod(root, "DeleteTask", startupTaskName, 0)
			if result != nil {
				result.Clear()
			}
			if isTaskSchedulerFileNotFound(err) {
				return struct{}{}, nil
			}
			if err != nil {
				return struct{}{}, fmt.Errorf("关闭开机自启失败: %w", err)
			}
			return struct{}{}, nil
		}

		executable, err := currentExecutablePath()
		if err != nil {
			return struct{}{}, err
		}
		currentUser, err := user.Current()
		if err != nil {
			return struct{}{}, fmt.Errorf("读取当前用户失败: %w", err)
		}
		username := strings.TrimSpace(currentUser.Username)
		if username == "" {
			return struct{}{}, errors.New("读取当前用户失败: 用户名为空")
		}

		registeredVariant, err := oleutil.CallMethod(
			root,
			"RegisterTask",
			startupTaskName,
			startupTaskXML(executable, username),
			taskCreateOrUpdate,
			nil,
			nil,
			taskLogonNone,
			nil,
		)
		if err != nil {
			return struct{}{}, fmt.Errorf("开启开机自启失败: %w", err)
		}
		if registeredVariant != nil {
			if registered := registeredVariant.ToIDispatch(); registered != nil {
				registered.Release()
			}
		}
		return struct{}{}, nil
	})
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

func startupTaskXML(executable, username string) string {
	escape := func(value string) string {
		var builder strings.Builder
		_ = xml.EscapeText(&builder, []byte(value))
		return builder.String()
	}
	executable = escape(executable)
	username = escape(username)

	return fmt.Sprintf(`<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo><Description>Key Flow 开机自启</Description></RegistrationInfo>
  <Triggers><LogonTrigger><Enabled>true</Enabled><UserId>%s</UserId></LogonTrigger></Triggers>
  <Principals><Principal id="Author"><UserId>%s</UserId><LogonType>InteractiveToken</LogonType><RunLevel>HighestAvailable</RunLevel></Principal></Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <StartWhenAvailable>true</StartWhenAvailable>
    <Enabled>true</Enabled>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
  </Settings>
  <Actions Context="Author"><Exec><Command>%s</Command><Arguments>--hidden</Arguments></Exec></Actions>
</Task>`, username, username, executable)
}

func withTaskSchedulerRoot[T any](fn func(*ole.IDispatch) (T, error)) (T, error) {
	var zero T
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return zero, fmt.Errorf("初始化计划任务服务失败: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Schedule.Service")
	if err != nil {
		return zero, fmt.Errorf("连接计划任务服务失败: %w", err)
	}
	defer unknown.Release()

	service, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return zero, fmt.Errorf("连接计划任务服务失败: %w", err)
	}
	defer service.Release()

	connected, err := oleutil.CallMethod(service, "Connect")
	if connected != nil {
		connected.Clear()
	}
	if err != nil {
		return zero, fmt.Errorf("连接计划任务服务失败: %w", err)
	}

	rootVariant, err := oleutil.CallMethod(service, "GetFolder", `\`)
	if err != nil {
		return zero, fmt.Errorf("打开计划任务根目录失败: %w", err)
	}
	root := rootVariant.ToIDispatch()
	if root == nil {
		return zero, errors.New("打开计划任务根目录失败: 目录对象无效")
	}
	defer root.Release()

	return fn(root)
}

func isTaskSchedulerFileNotFound(err error) bool {
	if err == nil {
		return false
	}
	var oleErr *ole.OleError
	if !errors.As(err, &oleErr) {
		return false
	}
	if uint32(oleErr.Code()) == taskSchedulerFileNotFoundHResult {
		return true
	}
	switch subErr := oleErr.SubError().(type) {
	case ole.EXCEPINFO:
		return subErr.SCODE() == taskSchedulerFileNotFoundHResult
	case *ole.EXCEPINFO:
		return subErr != nil && subErr.SCODE() == taskSchedulerFileNotFoundHResult
	default:
		return false
	}
}
