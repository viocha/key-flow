//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	idcArrow    = 32512
	idcSizeNWSE = 32642
	idcSizeNESW = 32643
	idcSizeAll  = 32646

	cursorWSPopup         = 0x80000000
	cursorWSExTopmost     = 0x00000008
	cursorWSExTransparent = 0x00000020
	cursorWSExToolWindow  = 0x00000080

	cursorSWHide   = 0
	cursorSWShowNA = 8

	cursorSWPNoSize         = 0x0001
	cursorSWPNoMove         = 0x0002
	cursorSWPNoRedraw       = 0x0008
	cursorSWPNoActivate     = 0x0010
	cursorSWPNoSendChanging = 0x0400
	cursorSWPDeferErase     = 0x2000
)

var (
	procRegisterClassW       = user32.NewProc("RegisterClassW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procSetClassLongPtrW     = user32.NewProc("SetClassLongPtrW")
	procShowWindowCursor     = user32.NewProc("ShowWindow")
	procSetWindowPosCursor   = user32.NewProc("SetWindowPos")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	cursorFeedbackWndProcPtr = syscall.NewCallback(cursorFeedbackWndProc)
)

type wndClassW struct {
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
}

func cursorFeedbackWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func createCursorFeedbackWindow() windows.Handle {
	className, err := windows.UTF16PtrFromString("KeyRunnerCursorFeedbackWindow")
	if err != nil {
		return 0
	}
	instance, _, _ := procGetModuleHandleW.Call(0)
	wndClass := wndClassW{
		WndProc:   cursorFeedbackWndProcPtr,
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	procRegisterClassW.Call(uintptr(unsafe.Pointer(&wndClass)))

	hwnd, _, _ := procCreateWindowExW.Call(
		cursorWSExToolWindow|cursorWSExTopmost|cursorWSExTransparent,
		uintptr(unsafe.Pointer(className)),
		0,
		cursorWSPopup,
		0,
		0,
		0,
		0,
		0,
		0,
		instance,
		0,
	)
	return windows.Handle(hwnd)
}

func setCursorFeedback(hwnd windows.Handle, cursorID uintptr, pt point) {
	if hwnd == 0 {
		return
	}
	cursor, _, _ := procLoadCursorW.Call(0, cursorID)
	if cursor == 0 {
		return
	}
	procSetClassLongPtrW.Call(uintptr(hwnd), ^uintptr(11), cursor)
	procShowWindowCursor.Call(uintptr(hwnd), cursorSWShowNA)
	procSetWindowPosCursor.Call(
		uintptr(hwnd),
		^uintptr(0),
		uintptr(pt.X-128),
		uintptr(pt.Y-128),
		256,
		256,
		cursorSWPNoActivate|cursorSWPNoRedraw|cursorSWPDeferErase|cursorSWPNoSendChanging,
	)
}

func moveCursorFeedback(hwnd windows.Handle, pt point) {
	if hwnd == 0 {
		return
	}
	procSetWindowPosCursor.Call(
		uintptr(hwnd),
		^uintptr(0),
		uintptr(pt.X-128),
		uintptr(pt.Y-128),
		0,
		0,
		cursorSWPNoSize|cursorSWPNoActivate|cursorSWPNoRedraw|cursorSWPDeferErase|cursorSWPNoSendChanging,
	)
}

func hideCursorFeedback(hwnd windows.Handle) {
	if hwnd == 0 {
		return
	}
	procSetWindowPosCursor.Call(
		uintptr(hwnd),
		0,
		0,
		0,
		0,
		0,
		cursorSWPNoMove|cursorSWPNoActivate|cursorSWPNoRedraw|cursorSWPDeferErase,
	)
	procShowWindowCursor.Call(uintptr(hwnd), cursorSWHide)
}
