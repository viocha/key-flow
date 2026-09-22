//go:build windows

package main

import (
	"fmt"
	"math"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	gaRoot                  = 2
	swMinimize              = 6
	swMaximize              = 3
	swRestore               = 9
	wmClose                 = 0x0010
	gwlExStyle              = -20
	wsExTopmost             = 0x00000008
	swpNoSize               = 0x0001
	swpNoMove               = 0x0002
	swpNoZOrder             = 0x0004
	swpNoActivate           = 0x0010
	monitorDefaultToNearest = 2
)

var (
	procWindowFromPoint           = user32.NewProc("WindowFromPoint")
	procGetAncestor               = user32.NewProc("GetAncestor")
	procGetWindowRect             = user32.NewProc("GetWindowRect")
	procSetWindowPos              = user32.NewProc("SetWindowPos")
	procShowWindow                = user32.NewProc("ShowWindow")
	procIsZoomed                  = user32.NewProc("IsZoomed")
	procPostMessageW              = user32.NewProc("PostMessageW")
	procGetCursorPos              = user32.NewProc("GetCursorPos")
	procMonitorFromPoint          = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW           = user32.NewProc("GetMonitorInfoW")
	procGetWindowLongPtrW         = user32.NewProc("GetWindowLongPtrW")
	procGetSystemMetrics          = user32.NewProc("GetSystemMetrics")
	procIsWindow                  = user32.NewProc("IsWindow")
	procIsIconic                  = user32.NewProc("IsIconic")
	procSetForegroundWindow       = user32.NewProc("SetForegroundWindow")
	dxva2                         = windows.NewLazySystemDLL("dxva2.dll")
	procGetNumberPhysicalMonitors = dxva2.NewProc("GetNumberOfPhysicalMonitorsFromHMONITOR")
	procGetPhysicalMonitors       = dxva2.NewProc("GetPhysicalMonitorsFromHMONITOR")
	procDestroyPhysicalMonitors   = dxva2.NewProc("DestroyPhysicalMonitors")
	procGetMonitorBrightness      = dxva2.NewProc("GetMonitorBrightness")
	procSetMonitorBrightness      = dxva2.NewProc("SetMonitorBrightness")
)

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

type physicalMonitor struct {
	Handle      windows.Handle
	Description [128]uint16
}

var (
	minimizedWindowsMu sync.Mutex
	minimizedWindows   []windows.Handle
)

type windowManipulation struct {
	hwnd       windows.Handle
	mode       WindowManipulationMode
	start      point
	startRect  rect
	resizeLeft bool
	resizeTop  bool
	cursorID   uintptr
}

func pointArgument(pt point) uintptr {
	return uintptr(uint64(uint32(pt.X)) | uint64(uint32(pt.Y))<<32)
}

func targetWindowAtPoint(pt point) windows.Handle {
	hwnd, _, _ := procWindowFromPoint.Call(pointArgument(pt))
	if hwnd == 0 {
		return 0
	}
	root, _, _ := procGetAncestor.Call(hwnd, gaRoot)
	if root != 0 {
		hwnd = root
	}
	var processID uint32
	procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	if processID == uint32(windows.GetCurrentProcessId()) {
		return 0
	}
	return windows.Handle(hwnd)
}

func getWindowRect(hwnd windows.Handle) (rect, error) {
	var value rect
	ok, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&value)))
	if ok == 0 {
		return rect{}, fmt.Errorf("读取窗口位置失败: %v", err)
	}
	return value, nil
}

func beginWindowManipulation(mode WindowManipulationMode, pt point, hwnd windows.Handle) (*windowManipulation, error) {
	if hwnd == 0 {
		hwnd = targetWindowAtPoint(pt)
	}
	if hwnd == 0 {
		return nil, fmt.Errorf("鼠标位置没有可操作窗口")
	}

	zoomed, _, _ := procIsZoomed.Call(uintptr(hwnd))
	if zoomed != 0 {
		procShowWindow.Call(uintptr(hwnd), swRestore)
	}
	value, err := getWindowRect(hwnd)
	if err != nil {
		return nil, err
	}
	procSetWindowPos.Call(
		uintptr(hwnd),
		0,
		0,
		0,
		0,
		0,
		swpNoMove|swpNoSize|swpNoActivate,
	)

	session := &windowManipulation{
		hwnd:      hwnd,
		mode:      mode,
		start:     pt,
		startRect: value,
	}
	if mode == WindowManipulationResize {
		centerX := value.Left + (value.Right-value.Left)/2
		centerY := value.Top + (value.Bottom-value.Top)/2
		session.resizeLeft = pt.X < centerX
		session.resizeTop = pt.Y < centerY
		if session.resizeLeft == session.resizeTop {
			session.cursorID = idcSizeNWSE
		} else {
			session.cursorID = idcSizeNESW
		}
	} else {
		session.cursorID = idcArrow
	}
	return session, nil
}

func (session *windowManipulation) update(pt point) error {
	if session == nil || session.hwnd == 0 {
		return nil
	}
	dx := pt.X - session.start.X
	dy := pt.Y - session.start.Y
	value := session.startRect

	switch session.mode {
	case WindowManipulationMove:
		width := value.Right - value.Left
		height := value.Bottom - value.Top
		return setWindowRect(session.hwnd, rect{
			Left:   value.Left + dx,
			Top:    value.Top + dy,
			Right:  value.Left + dx + width,
			Bottom: value.Top + dy + height,
		})
	case WindowManipulationResize:
		if session.resizeLeft {
			value.Left += dx
		} else {
			value.Right += dx
		}
		if session.resizeTop {
			value.Top += dy
		} else {
			value.Bottom += dy
		}
		const minWidth = 160
		const minHeight = 100
		if value.Right-value.Left < minWidth {
			if session.resizeLeft {
				value.Left = value.Right - minWidth
			} else {
				value.Right = value.Left + minWidth
			}
		}
		if value.Bottom-value.Top < minHeight {
			if session.resizeTop {
				value.Top = value.Bottom - minHeight
			} else {
				value.Bottom = value.Top + minHeight
			}
		}
		return setWindowRect(session.hwnd, value)
	default:
		return nil
	}
}

func setWindowRect(hwnd windows.Handle, value rect) error {
	width := value.Right - value.Left
	height := value.Bottom - value.Top
	ok, _, err := procSetWindowPos.Call(
		uintptr(hwnd),
		0,
		uintptr(value.Left),
		uintptr(value.Top),
		uintptr(width),
		uintptr(height),
		swpNoZOrder|swpNoActivate,
	)
	if ok == 0 {
		return fmt.Errorf("调整窗口失败: %v", err)
	}
	return nil
}

func scaleWindow(hwnd windows.Handle, deltaSign int, stepPercent int) error {
	if hwnd == 0 {
		return nil
	}
	value, err := getWindowRect(hwnd)
	if err != nil {
		return err
	}
	if stepPercent < 1 {
		stepPercent = 5
	}
	factor := 1 + float64(deltaSign*stepPercent)/100
	width := int32(math.Round(float64(value.Right-value.Left) * factor))
	height := int32(math.Round(float64(value.Bottom-value.Top) * factor))
	if width < 160 {
		width = 160
	}
	if height < 100 {
		height = 100
	}
	centerX := value.Left + (value.Right-value.Left)/2
	centerY := value.Top + (value.Bottom-value.Top)/2
	return setWindowRect(hwnd, rect{
		Left:   centerX - width/2,
		Top:    centerY - height/2,
		Right:  centerX - width/2 + width,
		Bottom: centerY - height/2 + height,
	})
}

func executeWindowAction(action Action) error {
	if isGlobalWindowCommand(action.WindowCommand) {
		return executeWindowActionForWindow(action, 0)
	}
	hwndRaw, _, _ := procGetForegroundWindow.Call()
	if hwndRaw == 0 {
		return nil
	}
	return executeWindowActionForWindow(action, windows.Handle(hwndRaw))
}

func isGlobalWindowCommand(command string) bool {
	switch command {
	case "previousWindow", "nextWindow", "restoreMinimized":
		return true
	default:
		return false
	}
}

func rememberMinimizedWindow(hwnd windows.Handle) {
	if hwnd == 0 {
		return
	}
	minimizedWindowsMu.Lock()
	defer minimizedWindowsMu.Unlock()

	items := minimizedWindows[:0]
	for _, item := range minimizedWindows {
		if item != hwnd {
			items = append(items, item)
		}
	}
	minimizedWindows = append(items, hwnd)
	if len(minimizedWindows) > 32 {
		minimizedWindows = append([]windows.Handle(nil), minimizedWindows[len(minimizedWindows)-32:]...)
	}
}

func restoreLastMinimizedWindow() {
	for {
		minimizedWindowsMu.Lock()
		if len(minimizedWindows) == 0 {
			minimizedWindowsMu.Unlock()
			return
		}
		index := len(minimizedWindows) - 1
		hwnd := minimizedWindows[index]
		minimizedWindows = minimizedWindows[:index]
		minimizedWindowsMu.Unlock()

		valid, _, _ := procIsWindow.Call(uintptr(hwnd))
		if valid == 0 {
			continue
		}
		iconic, _, _ := procIsIconic.Call(uintptr(hwnd))
		if iconic == 0 {
			continue
		}
		procShowWindow.Call(uintptr(hwnd), swRestore)
		procSetForegroundWindow.Call(uintptr(hwnd))
		return
	}
}

func executeWindowActionForWindow(action Action, hwnd windows.Handle) error {
	switch action.WindowCommand {
	case "previousWindow":
		return sendKeyStroke([]string{"Alt", "Shift", "Escape"})
	case "nextWindow":
		return sendKeyStroke([]string{"Alt", "Escape"})
	case "restoreMinimized":
		restoreLastMinimizedWindow()
		return nil
	}
	if hwnd == 0 {
		return nil
	}
	switch action.WindowCommand {
	case "close":
		procPostMessageW.Call(uintptr(hwnd), wmClose, 0, 0)
	case "terminateProcess":
		var processID uint32
		procGetWindowThreadProcessID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
		if processID == 0 {
			return nil
		}
		if processID == uint32(windows.GetCurrentProcessId()) {
			return fmt.Errorf("不能终止 Key Flow 自身进程")
		}
		process, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, processID)
		if err != nil {
			return fmt.Errorf("打开目标进程失败: %w", err)
		}
		defer windows.CloseHandle(process)
		if err := windows.TerminateProcess(process, 1); err != nil {
			return fmt.Errorf("终止目标进程失败: %w", err)
		}
	case "minimize":
		rememberMinimizedWindow(hwnd)
		procShowWindow.Call(uintptr(hwnd), swMinimize)
	case "maximize":
		procShowWindow.Call(uintptr(hwnd), swMaximize)
	case "restore":
		procShowWindow.Call(uintptr(hwnd), swRestore)
	case "toggleMaximize":
		zoomed, _, _ := procIsZoomed.Call(uintptr(hwnd))
		if zoomed != 0 {
			procShowWindow.Call(uintptr(hwnd), swRestore)
		} else {
			procShowWindow.Call(uintptr(hwnd), swMaximize)
		}
	case "alwaysOnTop":
		style, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), ^uintptr(19))
		if style&wsExTopmost == 0 {
			color, err := parseWindowBorderColor(action.WindowBorderColor)
			if err != nil {
				return err
			}
			procSetWindowPos.Call(
				uintptr(hwnd),
				^uintptr(0),
				0,
				0,
				0,
				0,
				swpNoSize|swpNoMove|swpNoActivate,
			)
			return showTopmostBorder(hwnd, color, action.WindowBorderSize)
		}
		procSetWindowPos.Call(
			uintptr(hwnd),
			^uintptr(1),
			0,
			0,
			0,
			0,
			swpNoSize|swpNoMove|swpNoActivate,
		)
		hideTopmostBorder(hwnd)
	default:
		return fmt.Errorf("未知窗口命令: %s", action.WindowCommand)
	}
	return nil
}

func adjustSystemVolume(delta int) error {
	if delta == 0 {
		return nil
	}
	key := "VolumeUp"
	if delta < 0 {
		key = "VolumeDown"
		delta = -delta
	}
	for i := 0; i < delta; i++ {
		if err := sendKeyStroke([]string{key}); err != nil {
			return err
		}
	}
	return nil
}

func cursorPoint() point {
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt
}

func monitorAtPoint(pt point) windows.Handle {
	handle, _, _ := procMonitorFromPoint.Call(pointArgument(pt), monitorDefaultToNearest)
	return windows.Handle(handle)
}

func virtualScreenRect() rect {
	const (
		smXVirtualScreen  = 76
		smYVirtualScreen  = 77
		smCXVirtualScreen = 78
		smCYVirtualScreen = 79
	)
	x, _, _ := procGetSystemMetrics.Call(smXVirtualScreen)
	y, _, _ := procGetSystemMetrics.Call(smYVirtualScreen)
	width, _, _ := procGetSystemMetrics.Call(smCXVirtualScreen)
	height, _, _ := procGetSystemMetrics.Call(smCYVirtualScreen)
	left := int32(x)
	top := int32(y)
	return rect{Left: left, Top: top, Right: left + int32(width), Bottom: top + int32(height)}
}

func overlayGesturePoints(points []GesturePoint) []GesturePoint {
	bounds := virtualScreenRect()
	result := make([]GesturePoint, len(points))
	for i, value := range points {
		result[i] = GesturePoint{
			X: value.X - float64(bounds.Left),
			Y: value.Y - float64(bounds.Top),
		}
	}
	return result
}
func monitorRectAtPoint(pt point) (rect, bool) {
	monitor := monitorAtPoint(pt)
	if monitor == 0 {
		return rect{}, false
	}
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	ok, _, _ := procGetMonitorInfoW.Call(uintptr(monitor), uintptr(unsafe.Pointer(&info)))
	return info.Monitor, ok != 0
}

func pointInEdgeZone(pt point, zone string, margin int) bool {
	bounds, ok := monitorRectAtPoint(pt)
	if !ok {
		return false
	}
	if margin < 1 {
		margin = 1
	}
	left := int(pt.X-bounds.Left) <= margin
	right := int(bounds.Right-1-pt.X) <= margin
	top := int(pt.Y-bounds.Top) <= margin
	bottom := int(bounds.Bottom-1-pt.Y) <= margin

	switch zone {
	case "topLeft":
		return top && left
	case "topRight":
		return top && right
	case "bottomLeft":
		return bottom && left
	case "bottomRight":
		return bottom && right
	case "top":
		return top && !left && !right
	case "bottom":
		return bottom && !left && !right
	case "left":
		return left && !top && !bottom
	case "right":
		return right && !top && !bottom
	default:
		return false
	}
}

func adjustDisplayBrightness(delta int) error {
	pt := cursorPoint()
	monitor := monitorAtPoint(pt)
	if monitor == 0 {
		return fmt.Errorf("无法获取当前显示器")
	}
	if err := adjustPhysicalMonitorBrightness(monitor, delta); err == nil {
		return nil
	}
	return adjustInternalBrightnessFallback(delta)
}

func adjustPhysicalMonitorBrightness(monitor windows.Handle, delta int) error {
	var count uint32
	ok, _, err := procGetNumberPhysicalMonitors.Call(uintptr(monitor), uintptr(unsafe.Pointer(&count)))
	if ok == 0 || count == 0 {
		return fmt.Errorf("显示器不支持 DDC/CI: %v", err)
	}
	monitors := make([]physicalMonitor, count)
	ok, _, err = procGetPhysicalMonitors.Call(uintptr(monitor), uintptr(count), uintptr(unsafe.Pointer(&monitors[0])))
	if ok == 0 {
		return fmt.Errorf("读取物理显示器失败: %v", err)
	}
	defer procDestroyPhysicalMonitors.Call(uintptr(count), uintptr(unsafe.Pointer(&monitors[0])))

	changed := false
	for i := range monitors {
		var minValue, currentValue, maxValue uint32
		ok, _, _ := procGetMonitorBrightness.Call(
			uintptr(monitors[i].Handle),
			uintptr(unsafe.Pointer(&minValue)),
			uintptr(unsafe.Pointer(&currentValue)),
			uintptr(unsafe.Pointer(&maxValue)),
		)
		if ok == 0 || maxValue <= minValue {
			continue
		}
		percent := int(math.Round(float64(currentValue-minValue) * 100 / float64(maxValue-minValue)))
		targetPercent := max(0, min(100, percent+delta))
		target := minValue + uint32(math.Round(float64(maxValue-minValue)*float64(targetPercent)/100))
		ok, _, _ = procSetMonitorBrightness.Call(uintptr(monitors[i].Handle), uintptr(target))
		changed = changed || ok != 0
	}
	if !changed {
		return fmt.Errorf("显示器拒绝亮度调整")
	}
	return nil
}

func adjustInternalBrightnessFallback(delta int) error {
	script := fmt.Sprintf("$b=Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness -ErrorAction Stop | Select-Object -First 1; $m=Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightnessMethods -ErrorAction Stop | Select-Object -First 1; $v=[Math]::Max(0,[Math]::Min(100,[int]$b.CurrentBrightness+(%d))); $m.WmiSetBrightness(0,$v) | Out-Null", delta)
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("调整内置显示器亮度失败: %w", err)
	}
	return nil
}
