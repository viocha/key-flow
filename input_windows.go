//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	whKeyboardLL = 13
	whMouseLL    = 14

	wmKeyDown    = 0x0100
	wmKeyUp      = 0x0101
	wmSysKeyDown = 0x0104
	wmSysKeyUp   = 0x0105

	wmMouseMove   = 0x0200
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmRButtonDown = 0x0204
	wmRButtonUp   = 0x0205
	wmMButtonDown = 0x0207
	wmMButtonUp   = 0x0208
	wmMouseWheel  = 0x020A
	wmMouseHWheel = 0x020E
	wmXButtonDown = 0x020B
	wmXButtonUp   = 0x020C

	vkBack     = 0x08
	vkTab      = 0x09
	vkReturn   = 0x0D
	vkShift    = 0x10
	vkControl  = 0x11
	vkMenu     = 0x12
	vkPause    = 0x13
	vkCapital  = 0x14
	vkEscape   = 0x1B
	vkSpace    = 0x20
	vkPrior    = 0x21
	vkNext     = 0x22
	vkEnd      = 0x23
	vkHome     = 0x24
	vkLeft     = 0x25
	vkUp       = 0x26
	vkRight    = 0x27
	vkDown     = 0x28
	vkPrint    = 0x2C
	vkInsert   = 0x2D
	vkDelete   = 0x2E
	vkLWin     = 0x5B
	vkRWin     = 0x5C
	vkApps     = 0x5D
	vkNumLock  = 0x90
	vkScroll   = 0x91
	vkLShift   = 0xA0
	vkRShift   = 0xA1
	vkLControl = 0xA2
	vkRControl = 0xA3
	vkLMenu    = 0xA4
	vkRMenu    = 0xA5

	vkVolumeMute = 0xAD
	vkVolumeDown = 0xAE
	vkVolumeUp   = 0xAF
	vkMediaNext  = 0xB0
	vkMediaPrev  = 0xB1
	vkMediaPlay  = 0xB3

	inputMouse    = 0
	inputKeyboard = 1

	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002
	keyeventfUnicode     = 0x0004

	mouseeventfLeftDown   = 0x0002
	mouseeventfLeftUp     = 0x0004
	mouseeventfRightDown  = 0x0008
	mouseeventfRightUp    = 0x0010
	mouseeventfMiddleDown = 0x0020
	mouseeventfMiddleUp   = 0x0040
	mouseeventfXDown      = 0x0080
	mouseeventfXUp        = 0x0100
	mouseeventfWheel      = 0x0800
	mouseeventfHWheel     = 0x1000

	keyRunnerInputMarker = uintptr(0x4B52554E)
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	shell32                      = windows.NewLazySystemDLL("shell32.dll")
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procSetWindowsHookExW        = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx           = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx      = user32.NewProc("UnhookWindowsHookEx")
	procGetMessageW              = user32.NewProc("GetMessageW")
	procPostThreadMessageW       = user32.NewProc("PostThreadMessageW")
	procSendInput                = user32.NewProc("SendInput")
	procGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procShellExecuteW            = shell32.NewProc("ShellExecuteW")
	procGetCurrentThreadID       = kernel32.NewProc("GetCurrentThreadId")
)

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type kbdLLHookStruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type msLLHookStruct struct {
	Pt          point
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type keybdInput struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type keyboardInput struct {
	Type    uint32
	_       uint32
	Ki      keybdInput
	Padding [8]byte
}

type mouseInputData struct {
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type mouseInput struct {
	Type uint32
	_    uint32
	Mi   mouseInputData
}

type activeTrigger struct {
	name             string
	layer            compiledLayer
	started          time.Time
	entered          bool
	used             bool
	timer            *time.Timer
	targetWindow     windows.Handle
	cursorChanged    bool
	mouseStart       point
	mousePassthrough bool
}

type activeGesture struct {
	trigger      string
	config       compiledGestures
	points       []GesturePoint
	active       bool
	canceled     bool
	label        string
	targetWindow windows.Handle
	timeout      time.Duration
	timeoutTimer *time.Timer
}

type pointerCapture struct {
	button  string
	window  *windowManipulation
	swallow bool
}

type pointerMoveRequest struct {
	capture *pointerCapture
	point   point
}

type windowsInputState struct {
	engine *InputEngine

	mu                   sync.Mutex
	threadID             uint32
	keyboardHook         windows.Handle
	mouseHook            windows.Handle
	closed               bool
	down                 map[uint32]bool
	swallowKeyUp         map[uint32]bool
	originalShortcutDown map[uint32]bool
	suppressedModifiers  map[uint32]bool
	trigger              *activeTrigger
	gesture              *activeGesture
	pointerCapture       *pointerCapture
	edgeSwitcherActive   bool
	wheel                wheelGate
	cursorWindow         windows.Handle
	pointerMoves         chan pointerMoveRequest
	pointerTasks         chan func()
	pointerStop          chan struct{}
	pointerWG            sync.WaitGroup
}

func startPlatformInputEngine(engine *InputEngine) (func(), error) {
	state := &windowsInputState{
		engine:              engine,
		down:                make(map[uint32]bool),
		swallowKeyUp:        make(map[uint32]bool),
		suppressedModifiers: make(map[uint32]bool),
		pointerMoves:        make(chan pointerMoveRequest, 1),
		pointerTasks:        make(chan func(), 32),
		pointerStop:         make(chan struct{}),
	}
	ready := make(chan error, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		threadID, _, _ := procGetCurrentThreadID.Call()
		cursorWindow := createCursorFeedbackWindow()

		state.mu.Lock()
		state.threadID = uint32(threadID)
		state.cursorWindow = cursorWindow
		state.mu.Unlock()

		keyboardCallback := syscall.NewCallback(func(code int, wParam uintptr, lParam uintptr) uintptr {
			if code < 0 {
				return callNextHook(code, wParam, lParam)
			}
			data := (*kbdLLHookStruct)(unsafe.Pointer(lParam))
			if data.DwExtraInfo == keyRunnerInputMarker {
				return callNextHook(code, wParam, lParam)
			}
			if state.handleKeyboard(uint32(wParam), data.VkCode) {
				return 1
			}
			return callNextHook(code, wParam, lParam)
		})

		keyboardHook, _, keyboardErr := procSetWindowsHookExW.Call(whKeyboardLL, keyboardCallback, 0, 0)
		if keyboardHook == 0 {
			if cursorWindow != 0 {
				procDestroyWindow.Call(uintptr(cursorWindow))
			}
			ready <- keyboardErr
			return
		}

		mouseCallback := syscall.NewCallback(func(code int, wParam uintptr, lParam uintptr) uintptr {
			if code < 0 {
				return callNextHook(code, wParam, lParam)
			}
			data := (*msLLHookStruct)(unsafe.Pointer(lParam))
			if data.DwExtraInfo == keyRunnerInputMarker {
				return callNextHook(code, wParam, lParam)
			}
			handled := state.handleMouse(uint32(wParam), data)
			if handled {
				return 1
			}
			return callNextHook(code, wParam, lParam)
		})

		mouseHook, _, mouseErr := procSetWindowsHookExW.Call(whMouseLL, mouseCallback, 0, 0)
		if mouseHook == 0 {
			procUnhookWindowsHookEx.Call(keyboardHook)
			if cursorWindow != 0 {
				procDestroyWindow.Call(uintptr(cursorWindow))
			}
			ready <- mouseErr
			return
		}

		stopHistory, historyErr := startWindowHistory()
		if historyErr != nil {
			procUnhookWindowsHookEx.Call(mouseHook)
			procUnhookWindowsHookEx.Call(keyboardHook)
			if cursorWindow != 0 {
				procDestroyWindow.Call(uintptr(cursorWindow))
			}
			ready <- historyErr
			return
		}
		defer stopHistory()
		state.mu.Lock()
		state.keyboardHook = windows.Handle(keyboardHook)
		state.mouseHook = windows.Handle(mouseHook)
		state.mu.Unlock()
		state.pointerWG.Add(1)
		go state.runPointerMoves()
		ready <- nil

		var message msg
		for {
			result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
			if int32(result) <= 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
		}
		hideCursorFeedback(cursorWindow)
		if cursorWindow != 0 {
			procDestroyWindow.Call(uintptr(cursorWindow))
		}
		procUnhookWindowsHookEx.Call(mouseHook)
		procUnhookWindowsHookEx.Call(keyboardHook)
	}()

	if err := <-ready; err != nil {
		return nil, fmt.Errorf("安装全局输入 Hook 失败: %w", err)
	}

	return func() {
		state.mu.Lock()
		if state.closed {
			state.mu.Unlock()
			return
		}
		state.closed = true
		close(state.pointerStop)
		if state.trigger != nil && state.trigger.timer != nil {
			state.trigger.timer.Stop()
		}
		hideCursorFeedback(state.cursorWindow)
		threadID := state.threadID
		state.mu.Unlock()
		state.pointerWG.Wait()

		if threadID != 0 {
			const wmQuit = 0x0012
			procPostThreadMessageW.Call(uintptr(threadID), wmQuit, 0, 0)
		}
	}, nil
}

func callNextHook(code int, wParam, lParam uintptr) uintptr {
	result, _, _ := procCallNextHookEx.Call(0, uintptr(code), wParam, lParam)
	return result
}

func (s *windowsInputState) handleKeyboard(message uint32, vk uint32) bool {
	isDown := message == wmKeyDown || message == wmSysKeyDown
	isUp := message == wmKeyUp || message == wmSysKeyUp
	if !isDown && !isUp {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if isDown {
		s.down[vk] = true
	}
	// 替代快捷键每次按下只执行一次，重复按下和松开都不交给目标程序。
	if s.originalShortcutDown[vk] {
		if isUp {
			delete(s.originalShortcutDown, vk)
			delete(s.down, vk)
		}
		return true
	}

	if s.edgeSwitcherActive && isDown && (vk == vkEscape || vk == vkReturn) {
		s.edgeSwitcherActive = false
	}

	if gesture := s.gesture; gesture != nil && vk == vkEscape {
		if isDown {
			s.cancelGestureLocked(gesture)
			s.swallowKeyUp[vk] = true
		}
		if isUp {
			delete(s.down, vk)
			delete(s.swallowKeyUp, vk)
		}
		return true
	}

	if s.trigger != nil && s.trigger.name == "CapsLock" && vk == vkCapital {
		if isDown {
			if message == wmKeyDown || message == wmSysKeyDown {
				return true
			}
		}
		if isUp {
			delete(s.down, vk)
			s.finishTriggerLocked()
			return true
		}
	}

	if isUp {
		delete(s.down, vk)
		if s.swallowKeyUp[vk] {
			delete(s.swallowKeyUp, vk)
			delete(s.suppressedModifiers, vk)
			return true
		}
		delete(s.suppressedModifiers, vk)
		return false
	}

	config := s.engine.snapshot()
	if !config.enabled {
		return false
	}

	if s.engine.InputCaptureActive() {
		return false
	}
	if s.trigger == nil {
		if layer, ok := config.layers["CapsLock"]; ok && layer.originalShortcut != "" &&
			layer.originalShortcut == s.currentChordLocked(keyNameFromVK(vk)) {
			if s.originalShortcutDown == nil {
				s.originalShortcutDown = make(map[uint32]bool)
			}
			s.originalShortcutDown[vk] = true
			s.engine.enqueueTask(func() {
				if err := sendOriginalTrigger("CapsLock"); err != nil {
					s.engine.setError(err)
				}
			})
			return true
		}
	}

	if isDown && vk == vkCapital {
		if layer, ok := config.layers["CapsLock"]; ok {
			s.beginTriggerLocked("CapsLock", layer)
			return true
		}
	}

	if s.trigger != nil {
		modifier := keyNameFromVK(vk)
		if isModifierKey(modifier) && layerUsesModifier(s.trigger.layer, modifier) {
			s.swallowKeyUp[vk] = true
			s.suppressedModifiers[vk] = true
			return true
		}
	}

	mainKey := keyNameFromVK(vk)
	if mainKey == "" || isModifierKey(mainKey) {
		return false
	}

	if s.trigger != nil {
		chord := s.currentChordLocked(mainKey)
		if action, ok := s.trigger.layer.bindings[chord]; ok {
			if s.trigger.timer != nil {
				s.trigger.timer.Stop()
			}
			s.trigger.entered = true
			s.trigger.used = true
			s.swallowKeyUp[vk] = true
			s.engine.enqueueActionContext(
				action,
				captureActionContext(
					s.trigger.targetWindow,
					cursorPoint(),
					s.suppressedModifierNamesLocked()...,
				),
			)
			return true
		}

		// Trigger 已经按下时，未映射的键不再继续参与普通全局快捷键匹配。
		// 它仍然交给目标程序，但会立即把本次 Trigger 视为 Layer 使用。
		if s.trigger.timer != nil {
			s.trigger.timer.Stop()
		}
		s.trigger.entered = true
		return false
	}

	chord := s.currentChordLocked(mainKey)
	if action, ok := config.hotkeys[chord]; ok {
		s.swallowKeyUp[vk] = true
		s.engine.enqueueActionContext(
			action,
			captureActionContext(foregroundTargetWindow(), cursorPoint()),
		)
		return true
	}
	return false
}

func (s *windowsInputState) handleMouseLegacy(message uint32, mouseData uint32) bool {
	name, isDown, isUp := mouseTriggerFromMessage(message, mouseData)
	if name == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trigger != nil && s.trigger.name == name {
		if isDown {
			return true
		}
		if isUp {
			s.finishTriggerLocked()
			return true
		}
	}

	config := s.engine.snapshot()
	if !config.enabled {
		return false
	}
	if s.engine.InputCaptureActive() {
		return false
	}
	if !isDown {
		return false
	}
	layer, ok := config.layers[name]
	if !ok {
		return false
	}
	s.beginTriggerLocked(name, layer)
	return true
}

func (s *windowsInputState) beginTriggerLocked(name string, layer compiledLayer) {
	if s.trigger != nil {
		return
	}
	trigger := &activeTrigger{
		name:         name,
		layer:        layer,
		started:      time.Now(),
		targetWindow: foregroundTargetWindow(),
	}
	trigger.timer = time.AfterFunc(layer.hold, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.trigger == trigger {
			trigger.entered = true
		}
	})
	s.trigger = trigger
}

func (s *windowsInputState) finishTriggerLocked() {
	trigger := s.trigger
	if trigger == nil {
		return
	}
	if trigger.timer != nil {
		trigger.timer.Stop()
	}
	if trigger.cursorChanged {
		cursorWindow := s.cursorWindow
		s.enqueuePointerTask(func() {
			hideCursorFeedback(cursorWindow)
		})
	}
	s.trigger = nil
	s.wheel.reset()

	if !trigger.entered && !trigger.used && time.Since(trigger.started) < trigger.layer.hold && trigger.layer.tapBehavior == "original" {
		name := trigger.name
		s.engine.enqueueTask(func() {
			if err := sendOriginalTrigger(name); err != nil {
				s.engine.setError(err)
			}
		})
	}
}

func (s *windowsInputState) currentChordLocked(mainKey string) string {
	keys := make([]string, 0, 5)
	if s.modifierDownLocked("Win") {
		keys = append(keys, "Win")
	}
	if s.modifierDownLocked("Ctrl") {
		keys = append(keys, "Ctrl")
	}
	if s.modifierDownLocked("Alt") {
		keys = append(keys, "Alt")
	}
	if s.modifierDownLocked("Shift") {
		keys = append(keys, "Shift")
	}
	keys = append(keys, mainKey)
	chord, _ := canonicalChord(keys)
	return chord
}

func (s *windowsInputState) modifierDownLocked(name string) bool {
	switch name {
	case "Ctrl":
		return s.down[vkControl] || s.down[vkLControl] || s.down[vkRControl]
	case "Shift":
		return s.down[vkShift] || s.down[vkLShift] || s.down[vkRShift]
	case "Alt":
		return s.down[vkMenu] || s.down[vkLMenu] || s.down[vkRMenu]
	case "Win":
		return s.down[vkLWin] || s.down[vkRWin]
	default:
		return false
	}
}

func (s *windowsInputState) suppressedModifierNamesLocked() []string {
	seen := make(map[string]bool, 4)
	for vk := range s.suppressedModifiers {
		name := keyNameFromVK(vk)
		if isModifierKey(name) {
			seen[name] = true
		}
	}

	modifiers := make([]string, 0, len(seen))
	for _, name := range []string{"Win", "Ctrl", "Alt", "Shift"} {
		if seen[name] {
			modifiers = append(modifiers, name)
		}
	}
	return modifiers
}

func layerUsesModifier(layer compiledLayer, modifier string) bool {
	for chord := range layer.bindings {
		for _, part := range strings.Split(chord, "+") {
			if part == modifier {
				return true
			}
		}
	}
	for shortcut := range layer.mouseShortcuts {
		for _, part := range strings.Split(shortcut, "+") {
			if part == modifier {
				return true
			}
		}
	}
	return false
}

func mouseTriggerFromMessage(message uint32, mouseData uint32) (name string, isDown bool, isUp bool) {
	switch message {
	case wmRButtonDown:
		return "MouseRight", true, false
	case wmRButtonUp:
		return "MouseRight", false, true
	case wmXButtonDown, wmXButtonUp:
		button := uint16(mouseData >> 16)
		if button == 1 {
			name = "MouseBack"
		} else if button == 2 {
			name = "MouseForward"
		}
		return name, message == wmXButtonDown, message == wmXButtonUp
	default:
		return "", false, false
	}
}

func isModifierKey(name string) bool {
	_, ok := modifierOrder[name]
	return ok
}

func keyNameFromVK(vk uint32) string {
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	if vk >= 0x70 && vk <= 0x87 {
		return "F" + strconv.Itoa(int(vk-0x70+1))
	}
	switch vk {
	case vkControl, vkLControl, vkRControl:
		return "Ctrl"
	case vkShift, vkLShift, vkRShift:
		return "Shift"
	case vkMenu, vkLMenu, vkRMenu:
		return "Alt"
	case vkLWin, vkRWin:
		return "Win"
	case vkCapital:
		return "CapsLock"
	case vkEscape:
		return "Escape"
	case vkReturn:
		return "Enter"
	case vkSpace:
		return "Space"
	case vkBack:
		return "Backspace"
	case vkDelete:
		return "Delete"
	case vkInsert:
		return "Insert"
	case vkHome:
		return "Home"
	case vkEnd:
		return "End"
	case vkPrior:
		return "PageUp"
	case vkNext:
		return "PageDown"
	case vkLeft:
		return "ArrowLeft"
	case vkRight:
		return "ArrowRight"
	case vkUp:
		return "ArrowUp"
	case vkDown:
		return "ArrowDown"
	case vkTab:
		return "Tab"
	case vkPrint:
		return "PrintScreen"
	case vkScroll:
		return "ScrollLock"
	case vkPause:
		return "Pause"
	case vkNumLock:
		return "NumLock"
	case vkApps:
		return "ContextMenu"
	case vkVolumeUp:
		return "VolumeUp"
	case vkVolumeDown:
		return "VolumeDown"
	case vkVolumeMute:
		return "VolumeMute"
	case vkMediaPlay:
		return "MediaPlayPause"
	case vkMediaNext:
		return "MediaNext"
	case vkMediaPrev:
		return "MediaPrev"
	default:
		return ""
	}
}

func vkFromKeyName(name string) (uint16, bool) {
	name = normalizeKeyName(name)
	if len(name) == 1 {
		ch := name[0]
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			return uint16(ch), true
		}
	}
	if strings.HasPrefix(name, "F") {
		if n, err := strconv.Atoi(strings.TrimPrefix(name, "F")); err == nil && n >= 1 && n <= 24 {
			return uint16(0x70 + n - 1), true
		}
	}
	switch name {
	case "Ctrl":
		return vkControl, true
	case "Shift":
		return vkShift, true
	case "Alt":
		return vkMenu, true
	case "Win":
		return vkLWin, true
	case "CapsLock":
		return vkCapital, true
	case "Escape":
		return vkEscape, true
	case "Enter":
		return vkReturn, true
	case "Space":
		return vkSpace, true
	case "Backspace":
		return vkBack, true
	case "Delete":
		return vkDelete, true
	case "Insert":
		return vkInsert, true
	case "Home":
		return vkHome, true
	case "End":
		return vkEnd, true
	case "PageUp":
		return vkPrior, true
	case "PageDown":
		return vkNext, true
	case "ArrowLeft":
		return vkLeft, true
	case "ArrowRight":
		return vkRight, true
	case "ArrowUp":
		return vkUp, true
	case "ArrowDown":
		return vkDown, true
	case "Tab":
		return vkTab, true
	case "PrintScreen":
		return vkPrint, true
	case "ScrollLock":
		return vkScroll, true
	case "Pause":
		return vkPause, true
	case "NumLock":
		return vkNumLock, true
	case "ContextMenu":
		return vkApps, true
	case "VolumeUp":
		return vkVolumeUp, true
	case "VolumeDown":
		return vkVolumeDown, true
	case "VolumeMute":
		return vkVolumeMute, true
	case "MediaPlayPause":
		return vkMediaPlay, true
	case "MediaNext":
		return vkMediaNext, true
	case "MediaPrev":
		return vkMediaPrev, true
	default:
		return 0, false
	}
}

func executePlatformActionContext(action Action, context ActionContext, keyInterval time.Duration) error {
	switch action.Type {
	case ActionKeyStroke:
		return sendKeyStrokeWithSuppressedModifiers(action.Keys, context.SuppressedModifiers)
	case ActionText:
		return sendText(action.Text)
	case ActionLaunch:
		return shellOpen(action.Target)
	case ActionCommand:
		program, args, workingDirectory, err := buildCommandAction(action, context)
		if err != nil {
			return err
		}
		command := exec.Command(program, args...)
		command.Dir = workingDirectory
		if err := command.Start(); err != nil {
			return fmt.Errorf("启动命令失败: %w", err)
		}
		return nil
	case ActionEdit:
		return executeEditAction(action, keyInterval, context.SuppressedModifiers)
	case ActionWindow:
		if context.Window != 0 {
			return executeWindowActionForWindow(action, windows.Handle(context.Window))
		}
		return executeWindowAction(action)
	case ActionMouse:
		return executeMouseAction(action)
	case ActionVolume:
		key := "VolumeUp"
		count := action.Delta
		if action.Delta < 0 {
			key = "VolumeDown"
			count = -count
		}
		for range count {
			if err := sendKeyStrokeWithSuppressedModifiers([]string{key}, context.SuppressedModifiers); err != nil {
				return err
			}
		}
		return nil
	case ActionBrightness:
		percent, pt, err := adjustDisplayBrightness(action.Delta)
		if err == nil {
			showBrightnessFeedback(percent, pt)
		}
		return err
	default:
		return fmt.Errorf("Windows 暂不支持动作类型: %s", action.Type)
	}
}

func sendOriginalTrigger(name string) error {
	switch name {
	case "CapsLock":
		return sendKeyStroke([]string{"CapsLock"})
	case "MouseRight":
		return sendMouseButton(mouseeventfRightDown, mouseeventfRightUp, 0)
	case "MouseMiddle":
		return sendMouseButton(mouseeventfMiddleDown, mouseeventfMiddleUp, 0)
	case "MouseBack":
		return sendMouseButton(mouseeventfXDown, mouseeventfXUp, 1)
	case "MouseForward":
		return sendMouseButton(mouseeventfXDown, mouseeventfXUp, 2)
	default:
		return fmt.Errorf("未知触发键: %s", name)
	}
}

func sendKeyStroke(keys []string) error {
	return sendKeyStrokeWithSuppressedModifiers(keys, nil)
}

func sendKeyStrokeWithSuppressedModifiers(keys []string, suppressedModifiers []string) error {
	chord, err := canonicalChord(keys)
	if err != nil {
		return err
	}
	inputs, err := keyStrokeInputs(chord, suppressedModifiers, func(vk uint16) bool {
		return keyPressed(uint32(vk))
	})
	if err != nil {
		return err
	}
	sent, _, callErr := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if sent != uintptr(len(inputs)) {
		return fmt.Errorf("SendInput 发送组合键失败: %v", callErr)
	}
	return nil
}

func keyStrokeInputs(chord string, suppressedModifiers []string, pressed func(uint16) bool) ([]keyboardInput, error) {
	parts := strings.Split(chord, "+")

	type pressedKey struct {
		vk       uint16
		extended bool
	}

	desiredModifiers := make(map[string]bool, 4)
	for _, part := range parts[:len(parts)-1] {
		desiredModifiers[part] = true
	}
	suppressed := make(map[string]bool, len(suppressedModifiers))
	for _, name := range suppressedModifiers {
		suppressed[normalizeKeyName(name)] = true
	}

	modifiers := []struct {
		name     string
		vk       uint16
		extended bool
	}{
		{name: "Ctrl", vk: vkControl},
		{name: "Shift", vk: vkShift},
		{name: "Alt", vk: vkMenu},
		{name: "Win", vk: vkLWin, extended: true},
	}

	inputs := make([]keyboardInput, 0, 18)
	appendKey := func(key pressedKey, up bool) {
		flags := uint32(0)
		if up {
			flags |= keyeventfKeyUp
		}
		if key.extended {
			flags |= keyeventfExtendedKey
		}
		inputs = append(inputs, keyboardInput{
			Type: inputKeyboard,
			Ki:   keybdInput{Vk: key.vk, Flags: flags, ExtraInfo: keyRunnerInputMarker},
		})
	}
	modifierPressed := func(name string, vk uint16) bool {
		if name == "Win" {
			return pressed(vkLWin) || pressed(vkRWin)
		}
		return pressed(vk)
	}

	// 先释放与目标组合键无关的实体修饰键，发送后恢复。
	releasedModifiers := make([]pressedKey, 0, 4)
	for _, modifier := range modifiers {
		if suppressed[modifier.name] || desiredModifiers[modifier.name] || !modifierPressed(modifier.name, modifier.vk) {
			continue
		}
		key := pressedKey{vk: modifier.vk, extended: modifier.extended}
		if modifier.name == "Win" && !pressed(vkLWin) {
			key.vk = vkRWin
		}
		appendKey(key, true)
		releasedModifiers = append(releasedModifiers, key)
	}

	injectedModifiers := make([]pressedKey, 0, 4)
	var main pressedKey

	for i, part := range parts {
		vk, ok := vkFromKeyName(part)
		if !ok {
			return nil, fmt.Errorf("无法模拟按键: %s", part)
		}
		key := pressedKey{vk: vk, extended: isExtendedVK(vk)}
		if i == len(parts)-1 {
			main = key
			continue
		}
		if suppressed[part] || !modifierPressed(part, vk) {
			appendKey(key, false)
			injectedModifiers = append(injectedModifiers, key)
		}
	}

	appendKey(main, false)
	appendKey(main, true)
	for i := len(injectedModifiers) - 1; i >= 0; i-- {
		appendKey(injectedModifiers[i], true)
	}
	for i := len(releasedModifiers) - 1; i >= 0; i-- {
		appendKey(releasedModifiers[i], false)
	}
	return inputs, nil
}

func sendText(text string) error {
	for _, unit := range utf16.Encode([]rune(text)) {
		down := keyboardInput{
			Type: inputKeyboard,
			Ki: keybdInput{
				Scan:      unit,
				Flags:     keyeventfUnicode,
				ExtraInfo: keyRunnerInputMarker,
			},
		}
		up := down
		up.Ki.Flags |= keyeventfKeyUp
		inputs := []keyboardInput{down, up}
		sent, _, callErr := procSendInput.Call(
			uintptr(len(inputs)),
			uintptr(unsafe.Pointer(&inputs[0])),
			unsafe.Sizeof(inputs[0]),
		)
		if sent != uintptr(len(inputs)) {
			return fmt.Errorf("SendInput 输入文本失败: %v", callErr)
		}
	}
	return nil
}

func sendMouseButtonEvent(flag uint32, mouseData uint32) error {
	input := mouseInput{
		Type: inputMouse,
		Mi: mouseInputData{
			MouseData: mouseData,
			Flags:     flag,
			ExtraInfo: keyRunnerInputMarker,
		},
	}
	sent, _, callErr := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&input)),
		unsafe.Sizeof(input),
	)
	if sent != 1 {
		return fmt.Errorf("SendInput 发送鼠标事件失败: %v", callErr)
	}
	return nil
}

func sendMouseButton(downFlag, upFlag uint32, mouseData uint32) error {
	inputs := []mouseInput{
		{
			Type: inputMouse,
			Mi: mouseInputData{
				MouseData: mouseData,
				Flags:     downFlag,
				ExtraInfo: keyRunnerInputMarker,
			},
		},
		{
			Type: inputMouse,
			Mi: mouseInputData{
				MouseData: mouseData,
				Flags:     upFlag,
				ExtraInfo: keyRunnerInputMarker,
			},
		},
	}
	sent, _, callErr := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if sent != uintptr(len(inputs)) {
		return fmt.Errorf("SendInput 发送鼠标事件失败: %v", callErr)
	}
	return nil
}

func executeMouseAction(action Action) error {
	switch action.MouseCommand {
	case "leftClick", "doubleClick":
		if err := sendMouseButton(mouseeventfLeftDown, mouseeventfLeftUp, 0); err != nil {
			return err
		}
		if action.MouseCommand == "doubleClick" {
			return sendMouseButton(mouseeventfLeftDown, mouseeventfLeftUp, 0)
		}
		return nil
	case "rightClick":
		return sendMouseButton(mouseeventfRightDown, mouseeventfRightUp, 0)
	case "middleClick":
		return sendMouseButton(mouseeventfMiddleDown, mouseeventfMiddleUp, 0)
	case "backClick":
		return sendMouseButton(mouseeventfXDown, mouseeventfXUp, 1)
	case "forwardClick":
		return sendMouseButton(mouseeventfXDown, mouseeventfXUp, 2)
	case "wheelUp", "wheelDown", "wheelLeft", "wheelRight":
		delta := int32(action.Delta * 120)
		flag := uint32(mouseeventfWheel)
		if action.MouseCommand == "wheelDown" || action.MouseCommand == "wheelLeft" {
			delta = -delta
		}
		if action.MouseCommand == "wheelLeft" || action.MouseCommand == "wheelRight" {
			flag = mouseeventfHWheel
		}
		return sendMouseButtonEvent(flag, uint32(delta))
	default:
		return fmt.Errorf("未知鼠标命令: %s", action.MouseCommand)
	}
}

func shellOpen(target string) error {
	operation, _ := windows.UTF16PtrFromString("open")
	targetPtr, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	result, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(operation)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		1,
	)
	if result <= 32 {
		return fmt.Errorf("打开目标失败: %v", callErr)
	}
	return nil
}

func keyPressed(vk uint32) bool {
	value, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return uint16(value)&0x8000 != 0
}

func captureActionContext(hwnd windows.Handle, pt point, suppressedModifiers ...string) ActionContext {
	return ActionContext{
		Window:              uintptr(hwnd),
		CursorX:             pt.X,
		CursorY:             pt.Y,
		TriggeredAt:         time.Now(),
		SuppressedModifiers: append([]string(nil), suppressedModifiers...),
	}
}

func isExtendedVK(vk uint16) bool {
	switch vk {
	case vkInsert, vkDelete, vkHome, vkEnd, vkPrior, vkNext, vkLeft, vkRight, vkUp, vkDown, vkLWin, vkRWin, vkApps, vkPrint, vkNumLock,
		vkVolumeMute, vkVolumeDown, vkVolumeUp, vkMediaNext, vkMediaPrev, vkMediaPlay:
		return true
	default:
		return false
	}
}
