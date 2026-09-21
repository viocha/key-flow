//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const cfUnicodeText = 13

var (
	procCommandLineToArgvW         = shell32.NewProc("CommandLineToArgvW")
	procLocalFree                  = kernel32.NewProc("LocalFree")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	commandEnvVariablePattern      = regexp.MustCompile(`\$\{env\.([A-Za-z_][A-Za-z0-9_]*)\}`)
)

func buildCommandAction(action Action, context ActionContext) (string, []string, string, error) {
	resolver := &commandTemplateResolver{context: context}

	program, err := resolver.expand(action.Program)
	if err != nil {
		return "", nil, "", err
	}
	workingDirectory, err := resolver.expand(action.WorkingDirectory)
	if err != nil {
		return "", nil, "", err
	}

	var args []string
	switch action.ArgsMode {
	case "commandLine":
		args, err = parseWindowsCommandLineArgs(action.CommandLine)
		if err != nil {
			return "", nil, "", err
		}
		for i, value := range args {
			args[i], err = resolver.expand(value)
			if err != nil {
				return "", nil, "", err
			}
		}
	case "list":
		args = make([]string, len(action.Args))
		for i, value := range action.Args {
			args[i], err = resolver.expand(value)
			if err != nil {
				return "", nil, "", err
			}
		}
	default:
		return "", nil, "", fmt.Errorf("命令参数输入方式无效: %s", action.ArgsMode)
	}
	return program, args, workingDirectory, nil
}

func parseWindowsCommandLineArgs(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}
	full := "key-flow " + value
	ptr, err := windows.UTF16PtrFromString(full)
	if err != nil {
		return nil, fmt.Errorf("命令参数包含无效字符: %w", err)
	}

	var argc int32
	argvRaw, _, callErr := procCommandLineToArgvW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&argc)),
	)
	if argvRaw == 0 {
		return nil, fmt.Errorf("解析命令参数失败: %v", callErr)
	}
	defer procLocalFree.Call(argvRaw)

	argv := unsafe.Slice((**uint16)(unsafe.Pointer(argvRaw)), int(argc))
	result := make([]string, 0, max(0, int(argc)-1))
	for i := 1; i < int(argc); i++ {
		result = append(result, windows.UTF16PtrToString(argv[i]))
	}
	return result, nil
}

type commandTemplateResolver struct {
	context         ActionContext
	clipboardLoaded bool
	clipboardText   string
	selectionLoaded bool
	selectionText   string
}

func expandCommandTemplate(value string, context ActionContext) (string, error) {
	return (&commandTemplateResolver{context: context}).expand(value)
}

func (r *commandTemplateResolver) expand(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	context := r.context
	result := value

	hasWindowVariable := strings.Contains(result, "${window.") || strings.Contains(result, "${process.")
	if hasWindowVariable {
		hwnd := windows.Handle(context.Window)
		if hwnd == 0 {
			return "", fmt.Errorf("命令引用了目标窗口或进程，但当前动作没有目标窗口")
		}

		result = strings.ReplaceAll(result, "${window.hwnd}", fmt.Sprintf("0x%X", uintptr(hwnd)))

		if strings.Contains(result, "${window.title}") {
			result = strings.ReplaceAll(result, "${window.title}", commandWindowTitle(hwnd))
		}

		if strings.Contains(result, "${window.x}") ||
			strings.Contains(result, "${window.y}") ||
			strings.Contains(result, "${window.width}") ||
			strings.Contains(result, "${window.height}") {
			bounds, err := getWindowRect(hwnd)
			if err != nil {
				return "", err
			}
			result = strings.NewReplacer(
				"${window.x}", strconv.FormatInt(int64(bounds.Left), 10),
				"${window.y}", strconv.FormatInt(int64(bounds.Top), 10),
				"${window.width}", strconv.FormatInt(int64(bounds.Right-bounds.Left), 10),
				"${window.height}", strconv.FormatInt(int64(bounds.Bottom-bounds.Top), 10),
			).Replace(result)
		}

		if strings.Contains(result, "${process.") {
			processID := commandWindowProcessID(hwnd)
			if processID == 0 {
				return "", fmt.Errorf("无法获取目标窗口对应的进程")
			}
			result = strings.ReplaceAll(result, "${process.pid}", strconv.FormatUint(uint64(processID), 10))

			if strings.Contains(result, "${process.path}") ||
				strings.Contains(result, "${process.name}") ||
				strings.Contains(result, "${process.dir}") {
				processPath, err := commandProcessPath(processID)
				if err != nil {
					return "", err
				}
				result = strings.NewReplacer(
					"${process.path}", processPath,
					"${process.name}", filepath.Base(processPath),
					"${process.dir}", filepath.Dir(processPath),
				).Replace(result)
			}
		}
	}

	result = strings.NewReplacer(
		"${cursor.x}", strconv.FormatInt(int64(context.CursorX), 10),
		"${cursor.y}", strconv.FormatInt(int64(context.CursorY), 10),
	).Replace(result)

	triggeredAt := context.TriggeredAt
	if triggeredAt.IsZero() {
		triggeredAt = time.Now()
	}
	if strings.Contains(result, "${time.") {
		result = strings.NewReplacer(
			"${time.date}", triggeredAt.Format("2006-01-02"),
			"${time.time}", triggeredAt.Format("15:04:05"),
			"${time.datetime}", triggeredAt.Format("2006-01-02 15:04:05"),
			"${time.unix}", strconv.FormatInt(triggeredAt.Unix(), 10),
			"${time.unixMs}", strconv.FormatInt(triggeredAt.UnixMilli(), 10),
		).Replace(result)
	}

	if strings.Contains(result, "${selection.text}") {
		if !r.selectionLoaded {
			selectionText, err := readSelectedText(context)
			if err != nil {
				return "", err
			}
			r.selectionText = selectionText
			r.selectionLoaded = true
		}
		result = strings.ReplaceAll(result, "${selection.text}", r.selectionText)
	}

	if strings.Contains(result, "${clipboard.text}") {
		if !r.clipboardLoaded {
			clipboardText, err := readClipboardText()
			if err != nil {
				return "", err
			}
			r.clipboardText = clipboardText
			r.clipboardLoaded = true
		}
		result = strings.ReplaceAll(result, "${clipboard.text}", r.clipboardText)
	}

	result = commandEnvVariablePattern.ReplaceAllStringFunc(result, func(token string) string {
		match := commandEnvVariablePattern.FindStringSubmatch(token)
		if len(match) != 2 {
			return token
		}
		return os.Getenv(match[1])
	})

	for _, prefix := range []string{"${window.", "${process.", "${cursor.", "${time.", "${selection.", "${clipboard.", "${env."} {
		if strings.Contains(result, prefix) {
			return "", fmt.Errorf("命令中存在未知变量: %s", prefix)
		}
	}
	return result, nil
}

func commandWindowTitle(hwnd windows.Handle) string {
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 {
		return ""
	}
	buffer := make([]uint16, int(length)+1)
	read, _, _ := procGetWindowTextW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if read == 0 {
		return ""
	}
	return windows.UTF16ToString(buffer[:read])
}

func commandWindowProcessID(hwnd windows.Handle) uint32 {
	var processID uint32
	procGetWindowThreadProcessID.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processID)),
	)
	return processID
}

func commandProcessPath(processID uint32) (string, error) {
	process, err := windows.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION,
		false,
		processID,
	)
	if err != nil {
		return "", fmt.Errorf("打开目标进程失败: %w", err)
	}
	defer windows.CloseHandle(process)

	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(process, 0, &buffer[0], &size); err != nil {
		return "", fmt.Errorf("读取目标进程路径失败: %w", err)
	}
	return windows.UTF16ToString(buffer[:size]), nil
}

func readClipboardText() (string, error) {
	available, _, _ := procIsClipboardFormatAvailable.Call(cfUnicodeText)
	if available == 0 {
		return "", nil
	}
	opened, _, callErr := procOpenClipboard.Call(0)
	if opened == 0 {
		return "", fmt.Errorf("打开剪贴板失败: %v", callErr)
	}
	defer procCloseClipboard.Call()

	handle, _, callErr := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", fmt.Errorf("读取剪贴板文本失败: %v", callErr)
	}
	data, _, callErr := procGlobalLock.Call(handle)
	if data == 0 {
		return "", fmt.Errorf("锁定剪贴板文本失败: %v", callErr)
	}
	defer procGlobalUnlock.Call(handle)

	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(data))), nil
}

func foregroundTargetWindow() windows.Handle {
	hwndRaw, _, _ := procGetForegroundWindow.Call()
	if hwndRaw == 0 {
		return 0
	}
	hwnd := windows.Handle(hwndRaw)
	var processID uint32
	procGetWindowThreadProcessID.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&processID)),
	)
	if processID == uint32(windows.GetCurrentProcessId()) {
		return 0
	}
	return hwnd
}
