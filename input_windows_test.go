//go:build windows

package main

import (
	"testing"
	"time"
)

func TestPlatformInputEngineRestart(t *testing.T) {
	config := defaultConfig()
	config.Enabled = false
	engine, err := NewInputEngine(config)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	var previous *windowsInputState
	for range 3 {
		var current *windowsInputState
		count := 0
		inputHookStates.Range(func(_, value any) bool {
			state := value.(*windowsInputState)
			if state.engine == engine {
				current = state
				count++
			}
			return true
		})
		if count != 1 || current == previous {
			t.Fatalf("应只有一份有效监听状态: 数量 %d", count)
		}
		previous = current
		if err := engine.Restart(); err != nil {
			t.Fatal(err)
		}
	}
	engine.Stop()
	inputHookStates.Range(func(_, value any) bool {
		if value.(*windowsInputState).engine == engine {
			t.Error("关闭后监听线程状态未清理")
		}
		return true
	})
}

func capsLockTestState() *windowsInputState {
	return &windowsInputState{
		engine: &InputEngine{
			compiled: compiledConfig{
				enabled: true,
				layers: map[string]compiledLayer{
					"CapsLock": {
						hold:             time.Second,
						tapBehavior:      "shortcut",
						originalShortcut: "Shift+CapsLock",
						bindings:         map[string]Action{"H": {Type: ActionKeyStroke, Keys: []string{"ArrowLeft"}}},
					},
				},
			},
			tasks: make(chan func(), 16),
		},
		down:                make(map[uint32]bool),
		swallowKeyUp:        make(map[uint32]bool),
		suppressedModifiers: make(map[uint32]bool),
	}
}

func TestCapsLockOriginalShortcutInput(t *testing.T) {
	for _, shift := range []uint32{vkLShift, vkRShift} {
		state := capsLockTestState()
		state.handleKeyboard(wmKeyDown, shift)
		if !state.handleKeyboard(wmKeyDown, vkCapital) || state.trigger != nil {
			t.Fatal("替代快捷键应被拦截且不进入 Layer")
		}
		state.handleKeyboard(wmKeyUp, shift)
		if !state.handleKeyboard(wmKeyDown, vkCapital) || !state.handleKeyboard(wmKeyUp, vkCapital) {
			t.Fatal("修饰键松开后的重复按下和主键松开也应被拦截")
		}
		if len(state.engine.tasks) != 1 || len(state.down) != 0 {
			t.Fatalf("动作数 = %d，按键状态 = %v", len(state.engine.tasks), state.down)
		}
		state.handleKeyboard(wmKeyDown, vkCapital)
		state.handleKeyboard(wmKeyUp, vkCapital)
		if len(state.engine.tasks) != 1 {
			t.Fatal("单击 CapsLock 不应执行原始操作")
		}
	}
}

func TestCapsLockShortcutModeLayer(t *testing.T) {
	state := capsLockTestState()
	state.handleKeyboard(wmKeyDown, vkCapital)
	if state.trigger == nil {
		t.Fatal("CapsLock 应开始 Layer 触发")
	}
	state.trigger.started = time.Now().Add(-2 * time.Second)
	state.handleKeyboard(wmKeyUp, vkCapital)
	if len(state.engine.tasks) != 0 {
		t.Fatal("长按松开不应执行原始操作")
	}
	state.handleKeyboard(wmKeyDown, vkCapital)
	if !state.handleKeyboard(wmKeyDown, 'H') || !state.handleKeyboard(wmKeyUp, 'H') {
		t.Fatal("Layer 映射应被拦截")
	}
	state.handleKeyboard(wmKeyUp, vkCapital)
	if len(state.engine.tasks) != 1 {
		t.Fatal("应只执行 Layer 映射动作")
	}
}

func TestCapsLockOriginalShortcutCaptureAndPause(t *testing.T) {
	for _, capture := range []bool{true, false} {
		state := capsLockTestState()
		if capture {
			state.engine.BeginInputCapture()
		} else {
			state.engine.compiled.enabled = false
		}
		state.handleKeyboard(wmKeyDown, vkLShift)
		if state.handleKeyboard(wmKeyDown, vkCapital) || state.handleKeyboard(wmKeyUp, vkCapital) {
			t.Fatal("录制或暂停时快捷键应交给系统")
		}
		if len(state.engine.tasks) != 0 || state.trigger != nil {
			t.Fatal("录制或暂停时不应触发动作")
		}
	}
}

func TestLongHoldDoesNotDependOnTimerScheduling(t *testing.T) {
	state := capsLockTestState()
	state.trigger = &activeTrigger{
		name:    "CapsLock",
		started: time.Now().Add(-time.Second),
		layer:   compiledLayer{hold: time.Millisecond * 200, tapBehavior: "original"},
	}
	state.finishTriggerLocked()
	if len(state.engine.tasks) != 0 {
		t.Fatal("长按判断应使用实际时长，避免计时器调度延迟导致误触")
	}
}

func TestHorizontalWheelDirection(t *testing.T) {
	if got := inputForWheel(wmMouseHWheel, 1); got != "wheelRight" {
		t.Fatalf("正向横向滚动 = %s", got)
	}
	if got := inputForWheel(wmMouseHWheel, -1); got != "wheelLeft" {
		t.Fatalf("反向横向滚动 = %s", got)
	}
	if got := inputForWheel(wmMouseWheel, 1); got != "wheelUp" {
		t.Fatalf("垂直滚动 = %s", got)
	}
}

func TestKeyStrokeInputsWinShortcut(t *testing.T) {
	inputs, err := keyStrokeInputs("Win+E", nil, func(uint16) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		vk    uint16
		flags uint32
	}{
		{vkLWin, keyeventfExtendedKey},
		{'E', 0},
		{'E', keyeventfKeyUp},
		{vkLWin, keyeventfExtendedKey | keyeventfKeyUp},
	}
	if len(inputs) != len(want) {
		t.Fatalf("输入事件数 = %d, 期望 %d", len(inputs), len(want))
	}
	for i, input := range inputs {
		if input.Ki.Vk != want[i].vk || input.Ki.Flags != want[i].flags {
			t.Errorf("事件 %d = (%X, %X), 期望 (%X, %X)", i, input.Ki.Vk, input.Ki.Flags, want[i].vk, want[i].flags)
		}
	}
}

func TestKeyStrokeInputsReleasesUnrelatedRightWin(t *testing.T) {
	inputs, err := keyStrokeInputs("Escape", nil, func(vk uint16) bool { return vk == vkRWin })
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 4 ||
		inputs[0].Ki.Vk != vkRWin || inputs[0].Ki.Flags != keyeventfExtendedKey|keyeventfKeyUp ||
		inputs[1].Ki.Vk != vkEscape || inputs[1].Ki.Flags != 0 ||
		inputs[2].Ki.Vk != vkEscape || inputs[2].Ki.Flags != keyeventfKeyUp ||
		inputs[3].Ki.Vk != vkRWin || inputs[3].Ki.Flags != keyeventfExtendedKey {
		t.Fatalf("输入事件顺序错误: %+v", inputs)
	}
}

func TestLayerUsesModifier(t *testing.T) {
	layer := compiledLayer{
		bindings: map[string]Action{
			"Alt+W": {Type: ActionKeyStroke, Keys: []string{"Ctrl", "Home"}},
		},
		mouseShortcuts: map[string]compiledMouseShortcut{
			"Shift+wheelUp": {action: Action{Type: ActionKeyStroke, Keys: []string{"PageUp"}}, wheelSteps: 1},
		},
	}

	if !layerUsesModifier(layer, "Alt") {
		t.Fatal("Alt should be reserved by the layer")
	}
	if !layerUsesModifier(layer, "Shift") {
		t.Fatal("Shift should be reserved by the layer mouse shortcut")
	}
	if layerUsesModifier(layer, "Ctrl") {
		t.Fatal("Ctrl should not be reserved by the layer")
	}
}

func TestSuppressedModifierNamesLocked(t *testing.T) {
	state := &windowsInputState{
		suppressedModifiers: map[uint32]bool{
			vkLMenu:  true,
			vkRShift: true,
		},
	}

	modifiers := state.suppressedModifierNamesLocked()
	if len(modifiers) != 2 || modifiers[0] != "Alt" || modifiers[1] != "Shift" {
		t.Fatalf("suppressed modifiers = %v, want [Alt Shift]", modifiers)
	}
}
