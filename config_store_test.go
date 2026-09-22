package main

import (
	"strings"
	"testing"
)

func testHotkey(id, name string, enabled bool, keys ...string) HotkeyRule {
	return HotkeyRule{
		ID:      id,
		Name:    name,
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
				ID:     id + "-h",
				Keys:   []string{"H"},
				Action: Action{Type: ActionKeyStroke, Keys: []string{"ArrowLeft"}},
			},
		},
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
}

func TestCompileConfigUsesActiveProfileOnly(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		testHotkey("active-rule", "当前规则", true, "Ctrl", "Alt", "T"),
	}
	config.Profiles = append(config.Profiles, Profile{
		ID:                "other",
		Name:              "其他",
		Hotkeys:           []HotkeyRule{testHotkey("other-rule", "其他规则", true, "Ctrl", "K")},
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

func TestValidateConfigRejectsEnabledHotkeyConflict(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		testHotkey("first", "第一个", true, "Ctrl", "Alt", "T"),
		testHotkey("second", "第二个", true, "Alt", "Ctrl", "T"),
	}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "快捷键冲突") {
		t.Fatalf("expected hotkey conflict, got %v", err)
	}
}

func TestValidateConfigAllowsDisabledHotkeyConflict(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].Hotkeys = []HotkeyRule{
		testHotkey("first", "第一个", true, "Ctrl", "Alt", "T"),
		testHotkey("second", "备用快捷键", false, "Alt", "Ctrl", "T"),
	}

	if err := validateConfig(config); err != nil {
		t.Fatalf("disabled conflicts should be allowed: %v", err)
	}
}

func TestValidateConfigRejectsDuplicateLayerBinding(t *testing.T) {
	config := defaultConfig()
	layer := testLayer("caps", "CapsLock", "CapsLock", true)
	layer.Bindings = append(layer.Bindings, LayerBinding{
		ID:     "duplicate-h",
		Keys:   []string{"H"},
		Action: Action{Type: ActionKeyStroke, Keys: []string{"Home"}},
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
			Input: "middle",
			Action: Action{
				Type:          ActionWindow,
				WindowCommand: "toggleMaximize",
			},
		},
		{
			Input:     "middle",
			Modifiers: []string{"Shift"},
			Action: Action{
				Type: ActionKeyStroke,
				Keys: []string{"Ctrl", "W"},
			},
		},
		{
			Input: "wheelUp",
			Action: Action{
				Type: ActionKeyStroke,
				Keys: []string{"Ctrl", "Tab"},
			},
		},
	}
	layer.WindowOperations = WindowPointerOperations{
		Enabled:          true,
		MoveButton:       "left",
		ResizeButton:     "right",
		ScaleStepPercent: 5,
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
	if action := got.mouseShortcuts["middle"]; action.Type != ActionWindow || action.WindowCommand != "toggleMaximize" {
		t.Fatalf("middle shortcut = %#v", action)
	}
	if action := got.mouseShortcuts["Shift+middle"]; action.Type != ActionKeyStroke {
		t.Fatalf("shift+middle shortcut = %#v", action)
	}
	if action := got.mouseShortcuts["wheelUp"]; action.Type != ActionKeyStroke {
		t.Fatalf("wheel-up shortcut = %#v", action)
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
			Modifiers: []string{"Alt", "Shift"},
			Action:    Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
		},
		{
			Input:     "left",
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
			Input:  "left",
			Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
		},
	}
	layer.WindowOperations = WindowPointerOperations{
		Enabled:          true,
		MoveButton:       "left",
		ScaleStepPercent: 5,
	}
	config.Profiles[0].Layers = []TriggerLayer{layer}

	err := validateConfig(config)
	if err == nil || !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("expected mouse/window conflict, got %v", err)
	}
}
