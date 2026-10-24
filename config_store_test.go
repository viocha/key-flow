package main

import (
	"strings"
	"testing"
	"time"
)

func testHotkey(id string, enabled bool, keys ...string) HotkeyRule {
	return HotkeyRule{
		ID:      id,
		Enabled: enabled,
		Keys:    keys,
		Action:  Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "C"}},
	}
}

func testLayer(id, name, trigger string, enabled bool) TriggerLayer {
	return TriggerLayer{
		ID:          id,
		Name:        name,
		Enabled:     enabled,
		Trigger:     trigger,
		HoldMS:      200,
		TapBehavior: "original",
		Bindings: []LayerBinding{
			{
				ID:      id + "-h",
				Enabled: true,
				Keys:    []string{"H"},
				Action:  Action{Type: ActionKeyStroke, Keys: []string{"ArrowLeft"}},
			},
		},
	}
}

func TestCapsLockOriginalShortcut(t *testing.T) {
	config := defaultConfig()
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.TapBehavior = "shortcut"
	layer.OriginalShortcut = []string{"CapsLock", "shift"}
	config.Profiles[0].Layers = []TriggerLayer{layer}
	if err := validateConfig(config); err != nil {
		t.Fatal(err)
	}
	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	engine := &InputEngine{compiled: compiled}
	if got := engine.snapshot().layers["CapsLock"].originalShortcut; got != "Shift+CapsLock" {
		t.Fatalf("原始操作快捷键 = %q", got)
	}
}

func TestValidateOriginalShortcut(t *testing.T) {
	for _, tc := range []struct {
		name     string
		trigger  string
		keys     []string
		conflict bool
	}{
		{"空快捷键", "CapsLock", nil, false},
		{"仅修饰键", "CapsLock", []string{"Shift"}, false},
		{"单独 CapsLock", "CapsLock", []string{"CapsLock"}, false},
		{"多个主键", "CapsLock", []string{"A", "B"}, false},
		{"鼠标触发键", "MouseRight", []string{"Shift", "CapsLock"}, false},
		{"通用快捷键冲突", "CapsLock", []string{"Shift", "CapsLock"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := defaultConfig()
			layer := testLayer("caps", "触发键", tc.trigger, true)
			layer.TapBehavior = "shortcut"
			layer.OriginalShortcut = tc.keys
			config.Profiles[0].Layers = []TriggerLayer{layer}
			if tc.conflict {
				config.Profiles[0].Hotkeys = []HotkeyRule{testHotkey("conflict", true, "Shift", "CapsLock")}
			}
			if err := validateConfig(config); err == nil {
				t.Fatal("应拒绝无效原始操作快捷键")
			}
		})
	}
}

func TestDefaultConfigValidatesAndCompiles(t *testing.T) {
	config := defaultConfig()
	if err := validateConfig(config); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if !compiled.enabled {
		t.Fatal("default runtime config should be enabled")
	}
	if len(compiled.hotkeys) != 0 {
		t.Fatalf("default hotkeys = %d, want 0", len(compiled.hotkeys))
	}
	if len(compiled.layers) != 0 {
		t.Fatalf("default layers = %d, want 0", len(compiled.layers))
	}
	if compiled.keyInterval != 0 {
		t.Fatalf("default key interval = %v, want 0", compiled.keyInterval)
	}
}

func TestGlobalEdgeSettingsApplyToAllRules(t *testing.T) {
	for _, adjustment := range []EdgeAdjustmentType{
		EdgeAdjustmentVolume, EdgeAdjustmentBrightness, EdgeAdjustmentRecentApps,
		EdgeAdjustmentSwitchApps, EdgeAdjustmentSwitchTabs, EdgeAdjustmentPage, EdgeAdjustmentCustom,
	} {
		t.Run(string(adjustment), func(t *testing.T) {
			config := defaultConfig()
			config.Profiles[0].EdgeWheelEndMarginPX = 100
			config.Profiles[0].EdgeWheelCornerExtendPX = 80
			action := Action{Type: ActionKeyStroke, Keys: []string{"PageUp"}}
			for _, zone := range []string{"top", "bottom", "left", "right", "topLeft", "topRight", "bottomLeft", "bottomRight"} {
				config.Profiles[0].EdgeWheels = append(config.Profiles[0].EdgeWheels, EdgeWheelRule{
					ID: zone, Enabled: true, Zone: zone,
					Input: "wheelVertical", Adjustment: adjustment, Step: 5, WheelSteps: 1,
					Action: &action, OppositeAction: &action,
				})
			}
			config = cloneConfig(config)
			compiled, err := compileConfig(config)
			if err != nil {
				t.Fatal(err)
			}
			engine := &InputEngine{compiled: compiled}
			for key, rule := range engine.snapshot().edgeWheels {
				if rule.endMarginPX != 100 || rule.cornerExtendPX != 80 {
					t.Errorf("%s 未应用全局设置: 两端间距 %d，角落延伸 %d", key, rule.endMarginPX, rule.cornerExtendPX)
				}
			}
		})
	}
}

func TestValidateEdgeEndMargin(t *testing.T) {
	for _, margin := range []int{-1, 0, 100, 16384, 16385} {
		config := defaultConfig()
		config.Profiles[0].EdgeWheelEndMarginPX = margin
		err := validateConfig(config)
		if valid := margin >= 0 && margin <= 16384; (err == nil) != valid {
			t.Errorf("两端间距 %d，校验错误: %v", margin, err)
		}
	}
}

func TestValidateCornerExtension(t *testing.T) {
	for _, distance := range []int{-1, 0, 100, 16384, 16385} {
		config := defaultConfig()
		config.Profiles[0].EdgeWheelCornerExtendPX = distance
		err := validateConfig(config)
		if valid := distance >= 0 && distance <= 16384; (err == nil) != valid {
			t.Errorf("角落延伸距离 %d，校验错误: %v", distance, err)
		}
	}
}

func TestCompileConfigUsesDefaultKeyInterval(t *testing.T) {
	config := defaultConfig()
	config.Settings.DefaultKeyIntervalMS = 25

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.keyInterval != 25*time.Millisecond {
		t.Fatalf("key interval = %v, want 25ms", compiled.keyInterval)
	}
}

func TestValidateConfigRejectsInvalidDefaultKeyInterval(t *testing.T) {
	config := defaultConfig()
	config.Settings.DefaultKeyIntervalMS = 501

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "默认按键间隔") {
		t.Fatalf("expected default key interval error, got %v", err)
	}
}

func TestCompileConfigUsesActiveProfileOnly(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		testHotkey("active-rule", true, "Ctrl", "Alt", "T"),
	}
	config.Profiles = append(config.Profiles, Profile{
		ID:                "other",
		Name:              "其他",
		Hotkeys:           []HotkeyRule{testHotkey("other-rule", true, "Ctrl", "K")},
		EdgeWheelMarginPX: 4,
	})

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled.hotkeys) != 1 {
		t.Fatalf("compiled hotkeys = %d, want 1", len(compiled.hotkeys))
	}
	if _, ok := compiled.hotkeys["Ctrl+Alt+T"]; !ok {
		t.Fatal("active profile hotkey was not compiled")
	}
	if _, ok := compiled.hotkeys["Ctrl+K"]; ok {
		t.Fatal("inactive profile hotkey must not be compiled")
	}
}

func TestCompileConfigAcceptsExistingModifierOrder(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		testHotkey("old-key-order", true, "Ctrl", "Shift", "Win", "K"),
		{
			ID:         "old-mouse-order",
			Enabled:    true,
			MouseInput: "left",
			Modifiers:  []string{"Ctrl", "Alt", "Win"},
			Action:     Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "C"}},
		},
	}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := compiled.hotkeys["Win+Ctrl+Shift+K"]; !ok {
		t.Fatal("旧顺序的键盘快捷键未按新顺序编译")
	}
	if _, ok := compiled.mouseHotkeys["Win+Ctrl+Alt+left"]; !ok {
		t.Fatal("旧顺序的鼠标快捷键未按新顺序编译")
	}
}

func TestCompileConfigSupportsGlobalMouseHotkey(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{{
		ID:         "alt-wheel-up",
		Enabled:    true,
		MouseInput: "wheelUp",
		WheelSteps: 1,
		Modifiers:  []string{"Alt"},
		Action:     Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
	}}

	if err := validateConfig(config); err != nil {
		t.Fatalf("global mouse hotkey should be valid: %v", err)
	}
	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := compiled.mouseHotkeys["Alt+wheelUp"]; !ok {
		t.Fatal("global mouse hotkey was not compiled")
	}
	if got := compiled.mouseHotkeys["Alt+wheelUp"].wheelSteps; got != 1 {
		t.Fatalf("global wheel steps = %d, want 1", got)
	}
}

func TestCompileConfigExpandsBidirectionalMouseShortcuts(t *testing.T) {
	config := defaultConfig()
	globalRight := Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}}
	layerDown := Action{Type: ActionVolume, Delta: -2}
	config.Profiles[0].Hotkeys = []HotkeyRule{{
		ID: "horizontal", Enabled: true, MouseInput: "wheelHorizontal", WheelSteps: 2,
		Modifiers:      []string{"Ctrl"},
		Action:         Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Shift", "Tab"}},
		OppositeAction: &globalRight,
	}}
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.MouseShortcuts = []MouseShortcutBinding{{
		Input: "wheelVertical", Enabled: true, WheelSteps: 3,
		Action: Action{Type: ActionVolume, Delta: 2}, OppositeAction: &layerDown,
	}}
	config.Profiles[0].Layers = []TriggerLayer{layer}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.mouseHotkeys["Ctrl+wheelLeft"]; got.wheelSteps != 2 || got.action.Keys[2] != "Tab" {
		t.Fatalf("左滚轮动作 = %#v", got)
	}
	if got := compiled.mouseHotkeys["Ctrl+wheelRight"]; got.wheelSteps != 2 || len(got.action.Keys) != 2 {
		t.Fatalf("右滚轮动作 = %#v", got)
	}
	if got := compiled.layers["CapsLock"].mouseShortcuts["wheelUp"]; got.wheelSteps != 3 || got.action.Delta != 2 {
		t.Fatalf("上滚轮动作 = %#v", got)
	}
	if got := compiled.layers["CapsLock"].mouseShortcuts["wheelDown"]; got.wheelSteps != 3 || got.action.Delta != -2 {
		t.Fatalf("下滚轮动作 = %#v", got)
	}
}

func TestValidateConfigRejectsBidirectionalMouseConflict(t *testing.T) {
	config := defaultConfig()
	other := Action{Type: ActionVolume, Delta: -1}
	config.Profiles[0].Hotkeys = []HotkeyRule{
		{ID: "both", Enabled: true, MouseInput: "wheelVertical", WheelSteps: 1,
			Modifiers: []string{"Ctrl"}, Action: Action{Type: ActionVolume, Delta: 1}, OppositeAction: &other},
		{ID: "down", Enabled: true, MouseInput: "wheelDown", WheelSteps: 1,
			Modifiers: []string{"Ctrl"}, Action: Action{Type: ActionVolume, Delta: -1}},
	}
	if err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "快捷键冲突") {
		t.Fatalf("双向与单向冲突未检出: %v", err)
	}
}

func TestValidateConfigAcceptsBidirectionalLayerScaleShortcut(t *testing.T) {
	config := defaultConfig()
	opposite := Action{Type: ActionWindow, WindowCommand: "scale", Delta: -5}
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.MouseShortcuts = []MouseShortcutBinding{{
		Input: "wheelVertical", Enabled: true, WheelSteps: 1,
		Action: Action{Type: ActionWindow, WindowCommand: "scale", Delta: 5}, OppositeAction: &opposite,
	}}
	config.Profiles[0].Layers = []TriggerLayer{layer}
	if err := validateConfig(config); err != nil {
		t.Fatalf("窗口中心缩放鼠标快捷键应有效: %v", err)
	}
}

func TestValidateConfigRejectsGlobalMouseHotkeyWithoutModifier(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{{
		ID:         "wheel-up",
		Enabled:    true,
		MouseInput: "wheelUp",
		WheelSteps: 1,
		Action:     Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
	}}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "至少需要一个修饰键") {
		t.Fatalf("expected missing modifier error, got %v", err)
	}
}

func TestValidateConfigRejectsDuplicateGlobalMouseHotkey(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		{
			ID:         "first",
			Enabled:    true,
			MouseInput: "left",
			Modifiers:  []string{"Alt", "Shift"},
			Action:     Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
		},
		{
			ID:         "second",
			Enabled:    true,
			MouseInput: "left",
			Modifiers:  []string{"Shift", "Alt"},
			Action:     Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "W"}},
		},
	}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "快捷键冲突") {
		t.Fatalf("expected duplicate global mouse hotkey error, got %v", err)
	}
}

func TestValidateConfigRejectsEnabledHotkeyConflict(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		testHotkey("first", true, "Ctrl", "Alt", "T"),
		testHotkey("second", true, "Alt", "Ctrl", "T"),
	}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "快捷键冲突") {
		t.Fatalf("expected hotkey conflict, got %v", err)
	}
}

func TestValidateConfigAllowsDisabledHotkeyConflict(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		testHotkey("first", true, "Ctrl", "Alt", "T"),
		testHotkey("second", false, "Alt", "Ctrl", "T"),
	}

	if err := validateConfig(config); err != nil {
		t.Fatalf("disabled conflicts should be allowed: %v", err)
	}
}

func TestValidateConfigRejectsDuplicateLayerBinding(t *testing.T) {
	config := defaultConfig()
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.Bindings = append(layer.Bindings, LayerBinding{
		ID:      "duplicate-h",
		Enabled: true,
		Keys:    []string{"H"},
		Action:  Action{Type: ActionKeyStroke, Keys: []string{"Home"}},
	})
	config.Profiles[0].Layers = []TriggerLayer{layer}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "重复映射") {
		t.Fatalf("expected layer binding conflict, got %v", err)
	}
}

func TestValidateConfigRejectsDuplicateEnabledTrigger(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Layers = []TriggerLayer{
		testLayer("first-caps", "CapsLock", "CapsLock", true),
		testLayer("second-caps", "第二个 CapsLock", "CapsLock", true),
	}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "触发键冲突") {
		t.Fatalf("expected trigger conflict, got %v", err)
	}
}

func TestCompileConfigSeparatesMouseShortcutsAndWindowOperations(t *testing.T) {
	config := defaultConfig()
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.MouseShortcuts = []MouseShortcutBinding{
		{
			Input:   "middle",
			Enabled: true,
			Action: Action{
				Type:          ActionWindow,
				WindowCommand: "toggleMaximize",
			},
		},
		{
			Input:     "middle",
			Enabled:   true,
			Modifiers: []string{"Shift"},
			Action: Action{
				Type: ActionKeyStroke,
				Keys: []string{"Ctrl", "W"},
			},
		},
		{
			Input:      "wheelUp",
			Enabled:    true,
			WheelSteps: 1,
			Action: Action{
				Type: ActionKeyStroke,
				Keys: []string{"Ctrl", "Tab"},
			},
		},
	}
	layer.WindowOperations = WindowPointerOperations{
		Enabled:      true,
		MoveButton:   "left",
		ResizeButton: "right",
	}
	config.Profiles[0].Layers = []TriggerLayer{layer}

	if err := validateConfig(config); err != nil {
		t.Fatalf("mouse config should be valid: %v", err)
	}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := compiled.layers["CapsLock"]
	if !ok {
		t.Fatal("CapsLock layer was not compiled")
	}
	if action := got.mouseShortcuts["middle"].action; action.Type != ActionWindow || action.WindowCommand != "toggleMaximize" {
		t.Fatalf("middle shortcut = %#v", action)
	}
	if action := got.mouseShortcuts["Shift+middle"].action; action.Type != ActionKeyStroke {
		t.Fatalf("shift+middle shortcut = %#v", action)
	}
	if action := got.mouseShortcuts["wheelUp"].action; action.Type != ActionKeyStroke {
		t.Fatalf("wheel-up shortcut = %#v", action)
	}
	if got.mouseShortcuts["wheelUp"].wheelSteps != 1 {
		t.Fatal("layer wheel sensitivity was not compiled")
	}
	if got.windowOperations.MoveButton != "left" || got.windowOperations.ResizeButton != "right" {
		t.Fatalf("window operations = %#v", got.windowOperations)
	}
}

func TestValidateConfigRejectsDuplicateMouseShortcutCombination(t *testing.T) {
	config := defaultConfig()
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.MouseShortcuts = []MouseShortcutBinding{
		{
			Input:     "left",
			Enabled:   true,
			Modifiers: []string{"Alt", "Shift"},
			Action:    Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
		},
		{
			Input:     "left",
			Enabled:   true,
			Modifiers: []string{"Shift", "Alt"},
			Action:    Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "W"}},
		},
	}
	config.Profiles[0].Layers = []TriggerLayer{layer}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "鼠标快捷键重复") {
		t.Fatalf("expected duplicate mouse shortcut error, got %v", err)
	}
}

func TestValidateActionAcceptsWindowNavigationCommands(t *testing.T) {
	for _, command := range []string{"previousWindow", "nextWindow", "restoreMinimized"} {
		if err := validateAction(Action{Type: ActionWindow, WindowCommand: command}, 0); err != nil {
			t.Fatalf("window command %q should be valid: %v", command, err)
		}
	}
}

func TestValidateConfigRejectsMouseShortcutWindowOperationConflict(t *testing.T) {
	config := defaultConfig()
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.MouseShortcuts = []MouseShortcutBinding{
		{
			Input:   "left",
			Enabled: true,
			Action:  Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
		},
	}
	layer.WindowOperations = WindowPointerOperations{
		Enabled:    true,
		MoveButton: "left",
	}
	config.Profiles[0].Layers = []TriggerLayer{layer}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("expected mouse/window conflict, got %v", err)
	}
}

func TestDisabledLayerBindingsDoNotCompileOrReserveWindowInput(t *testing.T) {
	config := defaultConfig()
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.Bindings[0].Enabled = false
	layer.MouseShortcuts = []MouseShortcutBinding{{
		Input:  "left",
		Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
	}}
	layer.WindowOperations = WindowPointerOperations{
		Enabled:    true,
		MoveButton: "left",
	}
	config.Profiles[0].Layers = []TriggerLayer{layer}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	got := compiled.layers["CapsLock"]
	if len(got.bindings) != 0 || len(got.mouseShortcuts) != 0 {
		t.Fatalf("disabled bindings compiled: %#v", got)
	}
	if got.windowOperations.MoveButton != "left" {
		t.Fatal("disabled mouse shortcut should not reserve window input")
	}
}
