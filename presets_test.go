package main

import "testing"

func TestActionPresetsValidAndUnique(t *testing.T) {
	ids := map[string]struct{}{}
	for _, preset := range actionPresets() {
		if preset.ID == "" {
			t.Fatal("预置动作 ID 不能为空")
		}
		if _, exists := ids[preset.ID]; exists {
			t.Fatalf("预置动作 ID 重复: %s", preset.ID)
		}
		ids[preset.ID] = struct{}{}
		if err := validateAction(preset.Action, 0); err != nil {
			t.Fatalf("预置动作 %q 无效: %v", preset.Name, err)
		}
	}
}
