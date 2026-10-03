//go:build windows

package main

import (
	"fmt"
	"math"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	gaRoot                  = 2
	swMinimize              = 6
	swMaximize              = 3
	swRestore               = 9
	swShowNoActivate        = 4
	wmClose                 = 0x0010
	gwlExStyle              = -20
	wsExTopmost             = 0x00000008
	wsExToolWindow          = 0x00000080
	wsExAppWindow           = 0x00040000
	wsExLayered             = 0x00080000
	wsExNoActivate          = 0x08000000
	lwaAlpha                = 0x00000002
	swpNoSize               = 0x0001
	swpNoMove               = 0x0002
	swpNoZOrder             = 0x0004
	swpNoActivate           = 0x0010
	monitorDefaultToNearest = 2
	gwOwner                 = 4
	dwmaCloaked             = 14
	pmNoRemove              = 0x0000
)

var (
	procWindowFromPoint            = user32.NewProc("WindowFromPoint")
	procGetAncestor                = user32.NewProc("GetAncestor")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procSetWindowPos               = user32.NewProc("SetWindowPos")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procIsZoomed                   = user32.NewProc("IsZoomed")
	procPostMessageW               = user32.NewProc("PostMessageW")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procMonitorFromPoint           = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW            = user32.NewProc("GetMonitorInfoW")
	procGetWindowLongPtrW          = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW          = user32.NewProc("SetWindowLongPtrW")
	procGetLayeredWindowAttributes = user32.NewProc("GetLayeredWindowAttributes")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procGetSystemMetrics           = user32.NewProc("GetSystemMetrics")
	procIsWindow                   = user32.NewProc("IsWindow")
	procIsIconic                   = user32.NewProc("IsIconic")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procGetWindow                  = user32.NewProc("GetWindow")
	procGetShellWindow             = user32.NewProc("GetShellWindow")
	procPeekMessageW               = user32.NewProc("PeekMessageW")
	procAttachThreadInput          = user32.NewProc("AttachThreadInput")
	procBringWindowToTop           = user32.NewProc("BringWindowToTop")
	procSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	procDwmGetWindowAttribute      = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmGetWindowAttribute")
	dxva2                          = windows.NewLazySystemDLL("dxva2.dll")
	procGetNumberPhysicalMonitors  = dxva2.NewProc("GetNumberOfPhysicalMonitorsFromHMONITOR")
	procGetPhysicalMonitors        = dxva2.NewProc("GetPhysicalMonitorsFromHMONITOR")
	procDestroyPhysicalMonitors    = dxva2.NewProc("DestroyPhysicalMonitors")
	procGetMonitorBrightness       = dxva2.NewProc("GetMonitorBrightness")
	procSetMonitorBrightness       = dxva2.NewProc("SetMonitorBrightness")
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
	minimizedWindowsMu  sync.Mutex
	minimizedWindows    []windows.Handle
	windowSwitchMu      sync.Mutex
	windowNavigationMu  sync.Mutex
	windowNavigating    bool
	windowHistory       activationHistory
	verticalMaximizedMu sync.Mutex
	verticalMaximized   = map[windows.Handle]struct{ original, applied rect }{}
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

func windowAtPoint(pt point) windows.Handle {
	hwnd, _, _ := procWindowFromPoint.Call(pointArgument(pt))
	if hwnd == 0 {
		return 0
	}
	root, _, _ := procGetAncestor.Call(hwnd, gaRoot)
	if root != 0 {
		hwnd = root
	}
	return windows.Handle(hwnd)
}

func targetWindowAtPoint(pt point) windows.Handle {
	hwnd := windowAtPoint(pt)
	if hwnd == 0 {
		return 0
	}
	var processID uint32
	procGetWindowThreadProcessID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
	if processID == uint32(windows.GetCurrentProcessId()) {
		return 0
	}
	return hwnd
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
		// 窗口拖拽允许直接操作 Key Flow 自身窗口。普通动作仍使用
		// targetWindowAtPoint 的进程过滤和各自动作的前台窗口回退逻辑。
		hwnd = windowAtPoint(pt)
	}
	if hwnd == 0 {
		return nil, fmt.Errorf("鼠标位置没有可操作窗口")
	}

	zoomed, _, _ := procIsZoomed.Call(uintptr(hwnd))
	if zoomed != 0 {
		// 还原最大化窗口时保留当前前台窗口。
		procShowWindow.Call(uintptr(hwnd), swShowNoActivate)
	}
	value, err := getWindowRect(hwnd)
	if err != nil {
		return nil, err
	}
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
		_ = activateSwitchWindow(hwnd)
		return
	}
}

type windowEnumeration struct {
	shell  uintptr
	result []windows.Handle
}

// Windows 回调槽位不会释放；复用回调，通过令牌隔离每次同步枚举的数据。
var windowEnumerations sync.Map
var windowEnumerationID atomic.Uintptr
var switchableWindowCallback = syscall.NewCallback(func(raw, token uintptr) uintptr {
	value, ok := windowEnumerations.Load(token)
	if !ok {
		return 0
	}
	enumeration := value.(*windowEnumeration)
	if raw == enumeration.shell {
		return 1
	}
	visible, _, _ := procIsWindowVisible.Call(raw)
	if visible == 0 {
		return 1
	}
	style, _, _ := procGetWindowLongPtrW.Call(raw, ^uintptr(19))
	if style&wsExNoActivate != 0 {
		return 1
	}
	if style&wsExToolWindow != 0 && style&wsExAppWindow == 0 {
		return 1
	}
	owner, _, _ := procGetWindow.Call(raw, gwOwner)
	if owner != 0 && style&wsExAppWindow == 0 {
		return 1
	}
	length, _, _ := procGetWindowTextLengthW.Call(raw)
	if length == 0 {
		return 1
	}
	iconic, _, _ := procIsIconic.Call(raw)
	if iconic == 0 {
		var cloaked uint32
		status, _, _ := procDwmGetWindowAttribute.Call(raw, dwmaCloaked, uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
		if status == 0 && cloaked != 0 {
			return 1
		}
	}
	enumeration.result = append(enumeration.result, windows.Handle(raw))
	return 1
})

func switchableWindows() []windows.Handle {
	shell, _, _ := procGetShellWindow.Call()
	enumeration := &windowEnumeration{shell: shell, result: make([]windows.Handle, 0, 16)}
	token := windowEnumerationID.Add(1)
	windowEnumerations.Store(token, enumeration)
	defer windowEnumerations.Delete(token)
	procEnumWindows.Call(switchableWindowCallback, token)
	return enumeration.result
}

func activateSwitchWindow(target windows.Handle) error {
	if target == 0 {
		return nil
	}

	// AttachThreadInput 绑定的是线程输入队列，因此整个激活过程必须固定在同一条 OS 线程上。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 显式创建当前 worker 线程的消息队列。AttachThreadInput 在任一线程没有消息队列时会失败。
	var message msg
	procPeekMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, pmNoRemove)

	currentThread, _, _ := procGetCurrentThreadID.Call()
	foreground, _, _ := procGetForegroundWindow.Call()
	foregroundThread, _, _ := procGetWindowThreadProcessID.Call(foreground, 0)
	targetThread, _, _ := procGetWindowThreadProcessID.Call(uintptr(target), 0)
	if targetThread == 0 {
		return fmt.Errorf("无法获取目标窗口线程")
	}

	attachedForeground := false
	if foregroundThread != 0 && foregroundThread != currentThread {
		ok, _, callErr := procAttachThreadInput.Call(currentThread, foregroundThread, 1)
		if ok == 0 {
			return fmt.Errorf("关联前台输入线程失败: %v", callErr)
		}
		attachedForeground = true
	}
	defer func() {
		if attachedForeground {
			procAttachThreadInput.Call(currentThread, foregroundThread, 0)
		}
	}()

	attachedTarget := false
	if targetThread != currentThread && targetThread != foregroundThread {
		ok, _, callErr := procAttachThreadInput.Call(currentThread, targetThread, 1)
		if ok == 0 {
			return fmt.Errorf("关联目标输入线程失败: %v", callErr)
		}
		attachedTarget = true
	}
	defer func() {
		if attachedTarget {
			procAttachThreadInput.Call(currentThread, targetThread, 0)
		}
	}()

	// 此时当前线程与前台/目标窗口共享输入队列；先取得前台，再恢复最小化状态。
	// 不调用 SwitchToThisWindow，避免 Shell 把切换当成 Alt-Tab 风格的注意提示。
	ok, _, callErr := procSetForegroundWindow.Call(uintptr(target))
	active, _, _ := procGetForegroundWindow.Call()
	if ok == 0 || active != uintptr(target) {
		return fmt.Errorf("切换前台窗口失败: %v", callErr)
	}

	iconic, _, _ := procIsIconic.Call(uintptr(target))
	if iconic != 0 {
		procShowWindow.Call(uintptr(target), swRestore)
	}
	procBringWindowToTop.Call(uintptr(target))

	active, _, _ = procGetForegroundWindow.Call()
	if active != uintptr(target) {
		return fmt.Errorf("窗口已恢复但未保持前台")
	}
	return nil
}

func switchWindow(direction int) error {
	windowNavigationMu.Lock()
	defer windowNavigationMu.Unlock()
	windowSwitchMu.Lock()

	available := switchableWindows()
	foreground, _, _ := procGetForegroundWindow.Call()
	current := windows.Handle(foreground)
	windowHistory.prune(available)
	windowHistory.observe(current, available)
	target, index := windowHistory.target(direction)
	if target == 0 {
		windowSwitchMu.Unlock()
		return nil
	}
	windowNavigating = true
	windowSwitchMu.Unlock()
	// 激活期间不持有历史锁，避免阻塞负责系统事件和输入 Hook 的线程。
	err := activateSwitchWindow(target)
	windowSwitchMu.Lock()
	defer windowSwitchMu.Unlock()
	windowNavigating = false
	if err == nil {
		windowHistory.index = index
	} else {
		foreground, _, _ = procGetForegroundWindow.Call()
		windowHistory.observe(windows.Handle(foreground), switchableWindows())
	}
	return err
}

func executeWindowActionForWindow(action Action, hwnd windows.Handle) error {
	switch action.WindowCommand {
	case "previousWindow":
		return switchWindow(-1)
	case "nextWindow":
		return switchWindow(1)
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
	case "toggleVerticalMaximize":
		return toggleVerticalMaximize(hwnd)
	case "opacity":
		return changeWindowOpacity(hwnd, action.Delta)
	case "scale":
		sign := 1
		if action.Delta < 0 {
			sign = -1
		}
		return scaleWindow(hwnd, sign, int(math.Abs(float64(action.Delta))))
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

func toggleVerticalMaximize(hwnd windows.Handle) error {
	zoomed, _, _ := procIsZoomed.Call(uintptr(hwnd))
	if zoomed != 0 {
		procShowWindow.Call(uintptr(hwnd), swRestore)
	}
	current, err := getWindowRect(hwnd)
	if err != nil {
		return err
	}
	verticalMaximizedMu.Lock()
	defer verticalMaximizedMu.Unlock()
	if previous, ok := verticalMaximized[hwnd]; ok && current.Top == previous.applied.Top && current.Bottom == previous.applied.Bottom {
		restored := current
		restored.Top = previous.original.Top
		restored.Bottom = previous.original.Bottom
		if err := setWindowRect(hwnd, restored); err != nil {
			return err
		}
		delete(verticalMaximized, hwnd)
		return nil
	}
	center := point{X: current.Left + (current.Right-current.Left)/2, Y: current.Top + (current.Bottom-current.Top)/2}
	monitor := monitorAtPoint(center)
	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	ok, _, callErr := procGetMonitorInfoW.Call(uintptr(monitor), uintptr(unsafe.Pointer(&info)))
	if monitor == 0 || ok == 0 {
		return fmt.Errorf("读取显示器工作区域失败: %v", callErr)
	}
	applied := current
	applied.Top = info.Work.Top
	applied.Bottom = info.Work.Bottom
	if err := setWindowRect(hwnd, applied); err != nil {
		return err
	}
	verticalMaximized[hwnd] = struct{ original, applied rect }{current, applied}
	return nil
}

func changeWindowOpacity(hwnd windows.Handle, delta int) error {
	style, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), ^uintptr(19))
	alpha := uint8(255)
	var colorKey, flags uint32
	if style&wsExLayered != 0 {
		ok, _, _ := procGetLayeredWindowAttributes.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&colorKey)), uintptr(unsafe.Pointer(&alpha)), uintptr(unsafe.Pointer(&flags)))
		if ok == 0 || flags&lwaAlpha == 0 {
			alpha = 255
		}
	} else {
		procSetWindowLongPtrW.Call(uintptr(hwnd), ^uintptr(19), style|wsExLayered)
	}
	percent := int(math.Round(float64(alpha) * 100 / 255))
	percent = max(20, min(100, percent-delta))
	target := uint8(math.Round(float64(percent) * 255 / 100))
	ok, _, callErr := procSetLayeredWindowAttributes.Call(uintptr(hwnd), uintptr(colorKey), uintptr(target), uintptr(flags|lwaAlpha))
	if ok == 0 {
		return fmt.Errorf("调整窗口透明度失败: %v", callErr)
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

func adjustDisplayBrightness(delta int) (int, point, error) {
	pt := cursorPoint()
	monitor := monitorAtPoint(pt)
	if monitor == 0 {
		return 0, pt, fmt.Errorf("无法获取当前显示器")
	}
	if percent, err := adjustPhysicalMonitorBrightness(monitor, delta); err == nil {
		return percent, pt, nil
	}
	percent, err := adjustInternalBrightnessFallback(delta)
	return percent, pt, err
}

func adjustPhysicalMonitorBrightness(monitor windows.Handle, delta int) (int, error) {
	var count uint32
	ok, _, err := procGetNumberPhysicalMonitors.Call(uintptr(monitor), uintptr(unsafe.Pointer(&count)))
	if ok == 0 || count == 0 {
		return 0, fmt.Errorf("显示器不支持 DDC/CI: %v", err)
	}
	monitors := make([]physicalMonitor, count)
	ok, _, err = procGetPhysicalMonitors.Call(uintptr(monitor), uintptr(count), uintptr(unsafe.Pointer(&monitors[0])))
	if ok == 0 {
		return 0, fmt.Errorf("读取物理显示器失败: %v", err)
	}
	defer procDestroyPhysicalMonitors.Call(uintptr(count), uintptr(unsafe.Pointer(&monitors[0])))

	changed := false
	var displayedPercent int
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
		if ok != 0 {
			changed = true
			displayedPercent = targetPercent
			var actual uint32
			if read, _, _ := procGetMonitorBrightness.Call(
				uintptr(monitors[i].Handle),
				uintptr(unsafe.Pointer(&minValue)),
				uintptr(unsafe.Pointer(&actual)),
				uintptr(unsafe.Pointer(&maxValue)),
			); read != 0 && maxValue > minValue {
				displayedPercent = int(math.Round(float64(actual-minValue) * 100 / float64(maxValue-minValue)))
			}
		}
	}
	if !changed {
		return 0, fmt.Errorf("显示器拒绝亮度调整")
	}
	return displayedPercent, nil
}

func adjustInternalBrightnessFallback(delta int) (int, error) {
	script := fmt.Sprintf("$b=Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness -ErrorAction Stop | Select-Object -First 1; $m=Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightnessMethods -ErrorAction Stop | Select-Object -First 1; $v=[Math]::Max(0,[Math]::Min(100,[int]$b.CurrentBrightness+(%d))); $r=$m.WmiSetBrightness(0,$v); if ($r.ReturnValue -ne 0) { exit 1 }; $after=Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness -ErrorAction SilentlyContinue | Select-Object -First 1; if ($after) { [Console]::Out.Write([int]$after.CurrentBrightness) } else { [Console]::Out.Write($v) }", delta)
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	output, err := command.Output()
	if err != nil {
		return 0, fmt.Errorf("调整内置显示器亮度失败: %w", err)
	}
	percent, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("读取调整后的亮度失败: %w", err)
	}
	return percent, nil
}
