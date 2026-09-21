//go:build windows

package main

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	uiaTextPatternID = 10014

	wmGetText       = 0x000D
	wmGetTextLength = 0x000E
	emGetSel        = 0x00B0
)

var (
	ole32Selection    = windows.NewLazySystemDLL("ole32.dll")
	oleaut32Selection = windows.NewLazySystemDLL("oleaut32.dll")

	procOleInitializeSelection    = ole32Selection.NewProc("OleInitialize")
	procOleUninitializeSelection  = ole32Selection.NewProc("OleUninitialize")
	procOleGetClipboardSelection  = ole32Selection.NewProc("OleGetClipboard")
	procOleSetClipboardSelection  = ole32Selection.NewProc("OleSetClipboard")
	procCoCreateInstanceSelection = ole32Selection.NewProc("CoCreateInstance")
	procSysFreeStringSelection    = oleaut32Selection.NewProc("SysFreeString")

	procGetClipboardSequenceNumberSelection = user32.NewProc("GetClipboardSequenceNumber")
	procEmptyClipboardSelection             = user32.NewProc("EmptyClipboard")
	procGetClassNameWSelection              = user32.NewProc("GetClassNameW")
	procSendMessageWSelection               = user32.NewProc("SendMessageW")
)

var (
	clsidCUIAutomation = windows.GUID{
		Data1: 0xFF48DBA4,
		Data2: 0x60EF,
		Data3: 0x4201,
		Data4: [8]byte{0xAA, 0x87, 0x54, 0x10, 0x3E, 0xEF, 0x59, 0x4E},
	}
	iidIUIAutomation = windows.GUID{
		Data1: 0x30CBE57D,
		Data2: 0xD9D0,
		Data3: 0x452A,
		Data4: [8]byte{0xAB, 0x13, 0x7A, 0xC5, 0xAC, 0x48, 0x25, 0xEE},
	}
	iidIUIAutomationTextPattern = windows.GUID{
		Data1: 0x32EBA289,
		Data2: 0x3583,
		Data3: 0x42C9,
		Data4: [8]byte{0x9C, 0x59, 0x3B, 0x6D, 0x9A, 0x1E, 0x9B, 0x6A},
	}
)

func readSelectedText(context ActionContext) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	oleResult, _, _ := procOleInitializeSelection.Call(0)
	oleReady := !hresultFailed(oleResult)
	if oleReady {
		defer procOleUninitializeSelection.Call()
	}

	if text, supported := readSelectionUIAutomation(context); supported {
		return text, nil
	}
	if text, supported := readSelectionStandardEdit(context); supported {
		return text, nil
	}
	if !oleReady {
		return "", errorsForSelectionFallback(oleResult)
	}
	return copySelectionWithClipboardRestore(context)
}

func readSelectionUIAutomation(context ActionContext) (string, bool) {
	var automation uintptr
	hr, _, _ := procCoCreateInstanceSelection.Call(
		uintptr(unsafe.Pointer(&clsidCUIAutomation)),
		0,
		windows.CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&iidIUIAutomation)),
		uintptr(unsafe.Pointer(&automation)),
	)
	if hresultFailed(hr) || automation == 0 {
		return "", false
	}
	defer comRelease(automation)

	var element uintptr
	if _, err := comCall(automation, 8, uintptr(unsafe.Pointer(&element))); err != nil || element == 0 {
		return "", false
	}
	defer comRelease(element)

	if context.Window != 0 {
		targetPID := commandWindowProcessID(windows.Handle(context.Window))
		if targetPID == 0 {
			return "", false
		}
		var elementPID int32
		if _, err := comCall(element, 20, uintptr(unsafe.Pointer(&elementPID))); err != nil || uint32(elementPID) != targetPID {
			return "", false
		}
	}

	var pattern uintptr
	if _, err := comCall(
		element,
		14,
		uiaTextPatternID,
		uintptr(unsafe.Pointer(&iidIUIAutomationTextPattern)),
		uintptr(unsafe.Pointer(&pattern)),
	); err != nil || pattern == 0 {
		return "", false
	}
	defer comRelease(pattern)

	var ranges uintptr
	if _, err := comCall(pattern, 5, uintptr(unsafe.Pointer(&ranges))); err != nil || ranges == 0 {
		return "", false
	}
	defer comRelease(ranges)

	var length int32
	if _, err := comCall(ranges, 3, uintptr(unsafe.Pointer(&length))); err != nil || length < 0 {
		return "", false
	}
	if length == 0 {
		return "", true
	}

	parts := make([]string, 0, length)
	for i := int32(0); i < length; i++ {
		var textRange uintptr
		if _, err := comCall(
			ranges,
			4,
			uintptr(i),
			uintptr(unsafe.Pointer(&textRange)),
		); err != nil || textRange == 0 {
			return "", false
		}

		text, ok := uiAutomationRangeText(textRange)
		comRelease(textRange)
		if !ok {
			return "", false
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n"), true
}

func uiAutomationRangeText(textRange uintptr) (string, bool) {
	var bstr uintptr
	if _, err := comCall(
		textRange,
		12,
		uintptr(^uint32(0)),
		uintptr(unsafe.Pointer(&bstr)),
	); err != nil {
		return "", false
	}
	if bstr == 0 {
		return "", true
	}
	defer procSysFreeStringSelection.Call(bstr)
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(bstr))), true
}

func readSelectionStandardEdit(context ActionContext) (string, bool) {
	if context.Window == 0 {
		return "", false
	}

	targetPID := commandWindowProcessID(windows.Handle(context.Window))
	if targetPID == 0 {
		return "", false
	}

	threadID, err := windows.GetWindowThreadProcessId(windows.HWND(context.Window), nil)
	if err != nil || threadID == 0 {
		return "", false
	}

	info := windows.GUIThreadInfo{Size: uint32(unsafe.Sizeof(windows.GUIThreadInfo{}))}
	if err := windows.GetGUIThreadInfo(threadID, &info); err != nil || info.Focus == 0 {
		return "", false
	}

	var focusPID uint32
	if _, err := windows.GetWindowThreadProcessId(info.Focus, &focusPID); err != nil || focusPID != targetPID {
		return "", false
	}

	className := windowClassName(windows.Handle(info.Focus))
	lowerClass := strings.ToLower(className)
	if !strings.Contains(lowerClass, "edit") {
		return "", false
	}

	textLength, _, _ := procSendMessageWSelection.Call(uintptr(info.Focus), wmGetTextLength, 0, 0)
	if textLength > 0xFFFF {
		return "", false
	}

	selection, _, _ := procSendMessageWSelection.Call(uintptr(info.Focus), emGetSel, 0, 0)
	start := int(uint16(selection & 0xFFFF))
	end := int(uint16((selection >> 16) & 0xFFFF))
	if start > end {
		start, end = end, start
	}
	if start == end {
		return "", true
	}

	buffer := make([]uint16, int(textLength)+1)
	read, _, _ := procSendMessageWSelection.Call(
		uintptr(info.Focus),
		wmGetText,
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&buffer[0])),
	)
	limit := int(read)
	if start > limit {
		start = limit
	}
	if end > limit {
		end = limit
	}
	if start >= end {
		return "", true
	}
	return windows.UTF16ToString(buffer[start:end]), true
}

func windowClassName(hwnd windows.Handle) string {
	buffer := make([]uint16, 256)
	length, _, _ := procGetClassNameWSelection.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if length == 0 {
		return ""
	}
	return windows.UTF16ToString(buffer[:length])
}

func copySelectionWithClipboardRestore(context ActionContext) (string, error) {
	if context.Window != 0 {
		targetPID := commandWindowProcessID(windows.Handle(context.Window))
		foreground := foregroundTargetWindow()
		if targetPID == 0 || foreground == 0 || commandWindowProcessID(foreground) != targetPID {
			return "", fmt.Errorf("无法通过复制读取选中文本：目标窗口已不是前台窗口")
		}
	}

	var originalData uintptr
	hr, _, _ := procOleGetClipboardSelection.Call(uintptr(unsafe.Pointer(&originalData)))
	if hresultFailed(hr) {
		return "", fmt.Errorf("无法保存原剪贴板: HRESULT 0x%08X", uint32(hr))
	}
	if originalData != 0 {
		defer comRelease(originalData)
	}

	sequenceBefore, _, _ := procGetClipboardSequenceNumberSelection.Call()
	if err := sendKeyStroke([]string{"Ctrl", "C"}); err != nil {
		return "", fmt.Errorf("复制选中文本失败: %w", err)
	}

	changed := false
	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		sequenceNow, _, _ := procGetClipboardSequenceNumberSelection.Call()
		if sequenceNow != sequenceBefore {
			changed = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !changed {
		return "", nil
	}

	text, readErr := readClipboardText()
	restoreErr := restoreClipboardDataObject(originalData)
	if readErr != nil {
		return "", readErr
	}
	if restoreErr != nil {
		return "", restoreErr
	}
	return text, nil
}

func restoreClipboardDataObject(originalData uintptr) error {
	if originalData != 0 {
		hr, _, _ := procOleSetClipboardSelection.Call(originalData)
		if hresultFailed(hr) {
			return fmt.Errorf("恢复原剪贴板失败: HRESULT 0x%08X", uint32(hr))
		}
		return nil
	}

	opened, _, callErr := procOpenClipboard.Call(0)
	if opened == 0 {
		return fmt.Errorf("恢复空剪贴板失败: %v", callErr)
	}
	defer procCloseClipboard.Call()
	ok, _, callErr := procEmptyClipboardSelection.Call()
	if ok == 0 {
		return fmt.Errorf("清空剪贴板失败: %v", callErr)
	}
	return nil
}

func comCall(object uintptr, index int, args ...uintptr) (uintptr, error) {
	method := comMethod(object, index)
	callArgs := make([]uintptr, 0, len(args)+1)
	callArgs = append(callArgs, object)
	callArgs = append(callArgs, args...)
	result, _, _ := syscall.SyscallN(method, callArgs...)
	if hresultFailed(result) {
		return result, fmt.Errorf("HRESULT 0x%08X", uint32(result))
	}
	return result, nil
}

func comRelease(object uintptr) {
	if object == 0 {
		return
	}
	syscall.SyscallN(comMethod(object, 2), object)
}

func comMethod(object uintptr, index int) uintptr {
	vtable := *(*uintptr)(unsafe.Pointer(object))
	return *(*uintptr)(unsafe.Pointer(vtable + uintptr(index)*unsafe.Sizeof(uintptr(0))))
}

func hresultFailed(value uintptr) bool {
	return int32(uint32(value)) < 0
}

func errorsForSelectionFallback(hr uintptr) error {
	return fmt.Errorf(
		"无法读取选中文本：直接读取不受支持，且无法使用剪贴板回退 (HRESULT 0x%08X)",
		uint32(hr),
	)
}
