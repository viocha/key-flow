//go:build windows

package main

import "testing"

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
	if len(modifiers) != 2 || modifiers[0] != "Shift" || modifiers[1] != "Alt" {
		t.Fatalf("suppressed modifiers = %v, want [Shift Alt]", modifiers)
	}
}
