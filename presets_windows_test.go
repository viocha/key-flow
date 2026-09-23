//go:build windows

package main

import "testing"

func TestKeystrokePresetsHaveWindowsVirtualKeys(t *testing.T) {
	for _, preset := range actionPresets() {
		if preset.Action.Type != ActionKeyStroke {
			continue
		}
		for _, key := range preset.Action.Keys {
			if _, ok := vkFromKeyName(key); !ok {
				t.Errorf("预置动作 %q 的按键 %q 无法发送", preset.Name, key)
			}
		}
	}
}
