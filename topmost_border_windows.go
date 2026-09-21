//go:build windows

package main

import (
	"fmt"
	"math"
	"sync"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
)

const (
	topmostBorderWSPopup         = 0x80000000
	topmostBorderWSExTopmost     = 0x00000008
	topmostBorderWSExTransparent = 0x00000020
	topmostBorderWSExToolWindow  = 0x00000080
	topmostBorderWSExNoActivate  = 0x08000000

	topmostBorderWMPaint         = 0x000F
	topmostBorderWMEraseBkgnd    = 0x0014
	topmostBorderWMNCHitTest     = 0x0084
	topmostBorderWMMouseActivate = 0x0021
	topmostBorderHTTransparent   = ^uintptr(0)
	topmostBorderMANoActivate    = 3

	topmostBorderSWShowNA    = 8
	topmostBorderSWHide      = 0
	topmostBorderSWPShow     = 0x0040
	topmostBorderObjIDWindow = 0

	topmostBorderEventSystemMinimizeStart = 0x0016
	topmostBorderEventSystemMinimizeEnd   = 0x0017
	topmostBorderEventObjectDestroy       = 0x8001
	topmostBorderEventObjectShow          = 0x8002
	topmostBorderEventObjectHide          = 0x8003
	topmostBorderEventObjectLocation      = 0x800B

	topmostBorderWinEventOutOfContext   = 0x0000
	topmostBorderWinEventSkipOwnProcess = 0x0002

	topmostBorderDWMExtendedFrameBounds = 9
)

var (
	topmostBorderDWM                       = windows.NewLazySystemDLL("dwmapi.dll")
	procTopmostBorderSetWinEventHook       = user32.NewProc("SetWinEventHook")
	procTopmostBorderUnhookWinEvent        = user32.NewProc("UnhookWinEvent")
	procTopmostBorderIsWindowVisible       = user32.NewProc("IsWindowVisible")
	procTopmostBorderIsIconic              = user32.NewProc("IsIconic")
	procTopmostBorderGetDpiForWindow       = user32.NewProc("GetDpiForWindow")
	procTopmostBorderBeginPaint            = user32.NewProc("BeginPaint")
	procTopmostBorderEndPaint              = user32.NewProc("EndPaint")
	procTopmostBorderGetClientRect         = user32.NewProc("GetClientRect")
	procTopmostBorderFillRect              = user32.NewProc("FillRect")
	procTopmostBorderInvalidateRect        = user32.NewProc("InvalidateRect")
	procTopmostBorderUpdateWindow          = user32.NewProc("UpdateWindow")
	procTopmostBorderDwmGetWindowAttribute = topmostBorderDWM.NewProc("DwmGetWindowAttribute")
	topmostBorderWndProcPtr                = syscall.NewCallback(topmostBorderWndProc)
	topmostBorderWinEventProcPtr           = syscall.NewCallback(topmostBorderWinEventProc)
	topmostBorders                         *topmostBorderManager
	topmostBorderEdgeColors                sync.Map
)

type topmostBorder struct {
	target windows.Handle
	edges  [4]windows.Handle
	color  uint32
	size   int
}

type topmostBorderManager struct {
	mu         sync.Mutex
	className  *uint16
	borders    map[windows.Handle]*topmostBorder
	edgeOwners map[windows.Handle]*topmostBorder
	objectHook windows.Handle
	systemHook windows.Handle
}

func showTopmostBorder(hwnd windows.Handle, color uint32, size int) error {
	return application.InvokeSyncWithError(func() error {
		manager, err := getTopmostBorderManager()
		if err != nil {
			return err
		}
		return manager.show(hwnd, color, size)
	})
}

func hideTopmostBorder(hwnd windows.Handle) {
	application.InvokeSync(func() {
		if topmostBorders != nil {
			topmostBorders.remove(hwnd)
		}
	})
}

func destroyTopmostBorders() {
	application.InvokeSync(func() {
		if topmostBorders != nil {
			topmostBorders.destroy()
			topmostBorders = nil
		}
	})
}

func getTopmostBorderManager() (*topmostBorderManager, error) {
	if topmostBorders != nil {
		return topmostBorders, nil
	}

	className, err := windows.UTF16PtrFromString("KeyRunnerTopmostBorderWindow")
	if err != nil {
		return nil, err
	}
	instance, _, _ := procGetModuleHandleW.Call(0)
	wndClass := wndClassW{
		WndProc:   topmostBorderWndProcPtr,
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	procRegisterClassW.Call(uintptr(unsafe.Pointer(&wndClass)))

	manager := &topmostBorderManager{
		className:  className,
		borders:    map[windows.Handle]*topmostBorder{},
		edgeOwners: map[windows.Handle]*topmostBorder{},
	}
	objectHook, _, _ := procTopmostBorderSetWinEventHook.Call(
		topmostBorderEventObjectDestroy,
		topmostBorderEventObjectLocation,
		0,
		topmostBorderWinEventProcPtr,
		0,
		0,
		topmostBorderWinEventOutOfContext|topmostBorderWinEventSkipOwnProcess,
	)
	systemHook, _, _ := procTopmostBorderSetWinEventHook.Call(
		topmostBorderEventSystemMinimizeStart,
		topmostBorderEventSystemMinimizeEnd,
		0,
		topmostBorderWinEventProcPtr,
		0,
		0,
		topmostBorderWinEventOutOfContext|topmostBorderWinEventSkipOwnProcess,
	)
	manager.objectHook = windows.Handle(objectHook)
	manager.systemHook = windows.Handle(systemHook)
	topmostBorders = manager
	return manager, nil
}

func (m *topmostBorderManager) show(hwnd windows.Handle, color uint32, size int) error {
	if hwnd == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	border := m.borders[hwnd]
	if border == nil {
		border = &topmostBorder{
			target: hwnd,
			color:  color,
			size:   size,
		}
		instance, _, _ := procGetModuleHandleW.Call(0)
		for i := range border.edges {
			edge, _, createErr := procCreateWindowExW.Call(
				topmostBorderWSExTopmost|
					topmostBorderWSExTransparent|
					topmostBorderWSExToolWindow|
					topmostBorderWSExNoActivate,
				uintptr(unsafe.Pointer(m.className)),
				0,
				topmostBorderWSPopup,
				0,
				0,
				1,
				1,
				0,
				0,
				instance,
				0,
			)
			if edge == 0 {
				for _, created := range border.edges {
					if created != 0 {
						delete(m.edgeOwners, created)
						procDestroyWindow.Call(uintptr(created))
					}
				}
				return fmt.Errorf("创建置顶边框失败: %v", createErr)
			}
			border.edges[i] = windows.Handle(edge)
			m.edgeOwners[windows.Handle(edge)] = border
			topmostBorderEdgeColors.Store(windows.Handle(edge), color)
		}
		m.borders[hwnd] = border
	} else {
		border.color = color
		border.size = size
		for _, edge := range border.edges {
			if edge != 0 {
				topmostBorderEdgeColors.Store(edge, color)
			}
		}
	}

	m.updateLocked(border)
	return nil
}

func (m *topmostBorderManager) remove(hwnd windows.Handle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeLocked(hwnd)
}

func (m *topmostBorderManager) removeLocked(hwnd windows.Handle) {
	border := m.borders[hwnd]
	if border == nil {
		return
	}
	delete(m.borders, hwnd)
	for _, edge := range border.edges {
		if edge == 0 {
			continue
		}
		delete(m.edgeOwners, edge)
		topmostBorderEdgeColors.Delete(edge)
		procDestroyWindow.Call(uintptr(edge))
	}
}

func (m *topmostBorderManager) updateLocked(border *topmostBorder) {
	if border == nil || border.target == 0 {
		return
	}

	visible, _, _ := procTopmostBorderIsWindowVisible.Call(uintptr(border.target))
	iconic, _, _ := procTopmostBorderIsIconic.Call(uintptr(border.target))
	style, _, _ := procGetWindowLongPtrW.Call(uintptr(border.target), ^uintptr(19))
	if visible == 0 || iconic != 0 || style&wsExTopmost == 0 {
		for _, edge := range border.edges {
			if edge != 0 {
				procShowWindow.Call(uintptr(edge), topmostBorderSWHide)
			}
		}
		if style&wsExTopmost == 0 {
			m.removeLocked(border.target)
		}
		return
	}

	targetRect, ok := topmostBorderTargetRect(border.target)
	if !ok {
		return
	}
	width := targetRect.Right - targetRect.Left
	height := targetRect.Bottom - targetRect.Top
	if width <= 0 || height <= 0 {
		return
	}

	dpi, _, _ := procTopmostBorderGetDpiForWindow.Call(uintptr(border.target))
	if dpi == 0 {
		dpi = 96
	}
	thickness := int32(math.Round(float64(border.size) * float64(dpi) / 96.0))
	if thickness < 1 {
		thickness = 1
	}
	maxThickness := min(width, height) / 4
	if thickness > maxThickness {
		thickness = maxThickness
	}
	if thickness < 1 {
		return
	}

	half := thickness / 2
	edgeRects := [4]rect{
		{
			Left:   targetRect.Left - half,
			Top:    targetRect.Top - half,
			Right:  targetRect.Right + (thickness - half),
			Bottom: targetRect.Top - half + thickness,
		},
		{
			Left:   targetRect.Left - half,
			Top:    targetRect.Bottom - (thickness - half),
			Right:  targetRect.Right + (thickness - half),
			Bottom: targetRect.Bottom + half,
		},
		{
			Left:   targetRect.Left - half,
			Top:    targetRect.Top + half,
			Right:  targetRect.Left - half + thickness,
			Bottom: targetRect.Bottom - half,
		},
		{
			Left:   targetRect.Right - (thickness - half),
			Top:    targetRect.Top + half,
			Right:  targetRect.Right + half,
			Bottom: targetRect.Bottom - half,
		},
	}

	for i, edge := range border.edges {
		if edge == 0 {
			continue
		}
		value := edgeRects[i]
		procSetWindowPos.Call(
			uintptr(edge),
			^uintptr(0),
			uintptr(value.Left),
			uintptr(value.Top),
			uintptr(value.Right-value.Left),
			uintptr(value.Bottom-value.Top),
			swpNoActivate|topmostBorderSWPShow,
		)
		procTopmostBorderInvalidateRect.Call(uintptr(edge), 0, 0)
		procTopmostBorderUpdateWindow.Call(uintptr(edge))
		procShowWindow.Call(uintptr(edge), topmostBorderSWShowNA)
	}
}

func (m *topmostBorderManager) destroy() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for hwnd := range m.borders {
		m.removeLocked(hwnd)
	}
	if m.objectHook != 0 {
		procTopmostBorderUnhookWinEvent.Call(uintptr(m.objectHook))
		m.objectHook = 0
	}
	if m.systemHook != 0 {
		procTopmostBorderUnhookWinEvent.Call(uintptr(m.systemHook))
		m.systemHook = 0
	}
}

func topmostBorderTargetRect(hwnd windows.Handle) (rect, bool) {
	var value rect
	status, _, _ := procTopmostBorderDwmGetWindowAttribute.Call(
		uintptr(hwnd),
		topmostBorderDWMExtendedFrameBounds,
		uintptr(unsafe.Pointer(&value)),
		unsafe.Sizeof(value),
	)
	if int32(status) >= 0 && value.Right > value.Left && value.Bottom > value.Top {
		return value, true
	}

	ok, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&value)))
	return value, ok != 0 && value.Right > value.Left && value.Bottom > value.Top
}

func topmostBorderWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case topmostBorderWMEraseBkgnd:
		return 1
	case topmostBorderWMNCHitTest:
		return topmostBorderHTTransparent
	case topmostBorderWMMouseActivate:
		return topmostBorderMANoActivate
	case topmostBorderWMPaint:
		var paint gestureOverlayPaintStruct
		hdc, _, _ := procTopmostBorderBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
		if hdc == 0 {
			return 0
		}
		defer procTopmostBorderEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))

		var color uint32
		if value, ok := topmostBorderEdgeColors.Load(windows.Handle(hwnd)); ok {
			color, _ = value.(uint32)
		}

		var client rect
		procTopmostBorderGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
		brush, _, _ := procGestureOverlayCreateSolidBrush.Call(uintptr(color))
		if brush != 0 {
			procTopmostBorderFillRect.Call(
				hdc,
				uintptr(unsafe.Pointer(&client)),
				brush,
			)
			procGestureOverlayDeleteObject.Call(brush)
		}
		return 0
	}
	result, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func topmostBorderWinEventProc(
	_ uintptr,
	event uintptr,
	hwnd uintptr,
	idObject uintptr,
	_ uintptr,
	_ uintptr,
	_ uintptr,
) uintptr {
	manager := topmostBorders
	if manager == nil || hwnd == 0 {
		return 0
	}
	if event >= topmostBorderEventObjectDestroy &&
		event <= topmostBorderEventObjectLocation &&
		int32(idObject) != topmostBorderObjIDWindow {
		return 0
	}

	target := windows.Handle(hwnd)
	manager.mu.Lock()
	defer manager.mu.Unlock()
	border := manager.borders[target]
	if border == nil {
		return 0
	}

	switch event {
	case topmostBorderEventObjectDestroy:
		manager.removeLocked(target)
	case topmostBorderEventObjectHide, topmostBorderEventSystemMinimizeStart:
		for _, edge := range border.edges {
			if edge != 0 {
				procShowWindow.Call(uintptr(edge), topmostBorderSWHide)
			}
		}
	case topmostBorderEventObjectShow,
		topmostBorderEventObjectLocation,
		topmostBorderEventSystemMinimizeEnd:
		manager.updateLocked(border)
	}
	return 0
}
