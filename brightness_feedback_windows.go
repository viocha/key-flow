//go:build windows

package main

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	brightnessFeedbackUpdateMessage = 0x8001
	brightnessFeedbackWMTimer       = 0x0113
	brightnessFeedbackTimerID       = 1
)

var (
	procBrightnessInvalidateRect = user32.NewProc("InvalidateRect")
	procBrightnessUpdateWindow   = user32.NewProc("UpdateWindow")
	procBrightnessSetTimer       = user32.NewProc("SetTimer")
	procBrightnessKillTimer      = user32.NewProc("KillTimer")
	brightnessFeedbackWndProcPtr = syscall.NewCallback(brightnessFeedbackWndProc)
	activeBrightnessFeedback     *nativeBrightnessFeedback
)

type nativeBrightnessFeedback struct {
	hwnd  windows.Handle
	mu    sync.Mutex
	box   rect
	scale float64
	text  string
}

func createNativeBrightnessFeedback() *nativeBrightnessFeedback {
	className, err := windows.UTF16PtrFromString("KeyFlowBrightnessFeedbackWindow")
	if err != nil {
		return nil
	}
	instance, _, _ := procGetModuleHandleW.Call(0)
	wndClass := wndClassW{
		WndProc:   brightnessFeedbackWndProcPtr,
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	procRegisterClassW.Call(uintptr(unsafe.Pointer(&wndClass)))
	hwnd, _, _ := procCreateWindowExW.Call(
		gestureOverlayWSExTopmost|
			gestureOverlayWSExTransparent|
			gestureOverlayWSExToolWindow|
			gestureOverlayWSExLayered|
			gestureOverlayWSExNoActivate,
		uintptr(unsafe.Pointer(className)),
		0,
		gestureOverlayWSPopup,
		0, 0, 0, 0,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return nil
	}
	feedback := &nativeBrightnessFeedback{hwnd: windows.Handle(hwnd)}
	activeBrightnessFeedback = feedback
	procGestureOverlaySetLayeredWindowAttributes.Call(
		hwnd, gestureOverlayColorKey, 255, gestureOverlayLWAColorKey,
	)
	return feedback
}

func (f *nativeBrightnessFeedback) Destroy() {
	if f == nil || f.hwnd == 0 {
		return
	}
	activeBrightnessFeedback = nil
	procBrightnessKillTimer.Call(uintptr(f.hwnd), brightnessFeedbackTimerID)
	procDestroyWindow.Call(uintptr(f.hwnd))
	f.hwnd = 0
}

func showBrightnessFeedback(percent int, pt point) {
	feedback := activeBrightnessFeedback
	if feedback == nil || feedback.hwnd == 0 {
		return
	}
	monitor, ok := monitorRectAtPoint(pt)
	if !ok {
		return
	}
	scale := dpiScaleAtPoint(pt)
	if scale <= 0 {
		scale = 1
	}
	width := scaleGestureOverlay(192, scale)
	height := scaleGestureOverlay(52, scale)
	bottomOffset := scaleGestureOverlay(72, scale)
	box := rect{
		Left:   monitor.Left + (monitor.Right-monitor.Left-width)/2,
		Top:    monitor.Bottom - bottomOffset - height,
		Bottom: monitor.Bottom - bottomOffset,
	}
	box.Right = box.Left + width
	feedback.mu.Lock()
	feedback.box = box
	feedback.scale = scale
	feedback.text = fmt.Sprintf("亮度 %d%%", max(0, min(100, percent)))
	feedback.mu.Unlock()
	procPostMessageW.Call(uintptr(feedback.hwnd), brightnessFeedbackUpdateMessage, 0, 0)
}

func brightnessFeedbackWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case gestureOverlayWMEraseBkgnd:
		return 1
	case gestureOverlayWMNCHitTest:
		return gestureOverlayHTTransparent
	case gestureOverlayWMMouseActivate:
		return gestureOverlayMANoActivate
	case gestureOverlayWMPaint:
		paintBrightnessFeedback(hwnd)
		return 0
	case brightnessFeedbackUpdateMessage:
		if feedback := activeBrightnessFeedback; feedback != nil {
			feedback.updateWindow()
		}
		return 0
	case brightnessFeedbackWMTimer:
		if wParam == brightnessFeedbackTimerID {
			procBrightnessKillTimer.Call(hwnd, brightnessFeedbackTimerID)
			procShowWindow.Call(hwnd, cursorSWHide)
			return 0
		}
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func (f *nativeBrightnessFeedback) updateWindow() {
	f.mu.Lock()
	box := f.box
	f.mu.Unlock()
	procSetWindowPos.Call(
		uintptr(f.hwnd), ^uintptr(0),
		uintptr(box.Left), uintptr(box.Top),
		uintptr(box.Right-box.Left), uintptr(box.Bottom-box.Top),
		swpNoActivate,
	)
	procBrightnessInvalidateRect.Call(uintptr(f.hwnd), 0, 0)
	procBrightnessUpdateWindow.Call(uintptr(f.hwnd))
	procShowWindow.Call(uintptr(f.hwnd), gestureOverlaySWShowNA)
	procBrightnessKillTimer.Call(uintptr(f.hwnd), brightnessFeedbackTimerID)
	procBrightnessSetTimer.Call(uintptr(f.hwnd), brightnessFeedbackTimerID, 1000, 0)
}

func paintBrightnessFeedback(hwnd uintptr) {
	var paint gestureOverlayPaintStruct
	hdc, _, _ := procGestureOverlayBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
	if hdc == 0 {
		return
	}
	defer procGestureOverlayEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
	feedback := activeBrightnessFeedback
	if feedback == nil {
		return
	}
	feedback.mu.Lock()
	width := feedback.box.Right - feedback.box.Left
	height := feedback.box.Bottom - feedback.box.Top
	scale := feedback.scale
	label := feedback.text
	feedback.mu.Unlock()
	if width <= 0 || height <= 0 {
		return
	}
	client := rect{Right: width, Bottom: height}
	brush, _, _ := procGestureOverlayCreateSolidBrush.Call(gestureOverlayColorKey)
	procGestureOverlayFillRect.Call(hdc, uintptr(unsafe.Pointer(&client)), brush)
	procGestureOverlayDeleteObject.Call(brush)
	area := rect{Right: width, Bottom: height + scaleGestureOverlay(72, scale)}
	drawGestureOverlayLabel(hdc, area, label, scale)
}
