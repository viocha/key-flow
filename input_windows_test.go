//go:build windows

package main

import "testing"

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
