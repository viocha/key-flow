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

func TestNewMouseAndWindowActionsValidate(t *testing.T) {
	valid := []Action{
		{Type: ActionMouse, MouseCommand: "doubleClick"},
		{Type: ActionMouse, MouseCommand: "wheelLeft", Delta: 1},
		{Type: ActionMouse, MouseCommand: "wheelRight", Delta: 20},
		{Type: ActionWindow, WindowCommand: "toggleVerticalMaximize"},
		{Type: ActionWindow, WindowCommand: "opacity", Delta: -100},
		{Type: ActionWindow, WindowCommand: "scale", Delta: 5},
	}
	for _, action := range valid {
		if err := validateAction(action, 0); err != nil {
			t.Errorf("有效动作 %+v 被拒绝: %v", action, err)
		}
	}
	invalid := []Action{
		{Type: ActionMouse, MouseCommand: "wheelUp"},
		{Type: ActionMouse, MouseCommand: "wheelDown", Delta: 21},
		{Type: ActionWindow, WindowCommand: "opacity"},
	}
	for _, action := range invalid {
		if err := validateAction(action, 0); err == nil {
			t.Errorf("无效动作 %+v 未被拒绝", action)
		}
	}
	for _, input := range []string{"wheelLeft", "wheelRight"} {
		if !isSupportedMouseShortcutInput(input) || !isWheelInput(input) {
			t.Errorf("横向滚轮输入 %s 未被识别", input)
		}
	}
}

func TestAdjustablePresetsAreSingleParameterizedActions(t *testing.T) {
	want := map[string]int{
		"window-opacity":     5,
		"window-scale":       5,
		"media-volume":       1,
		"display-brightness": 5,
	}
	for _, preset := range actionPresets() {
		if delta, ok := want[preset.ID]; ok {
			if preset.Action.Delta != delta {
				t.Errorf("%s 默认变化 = %d，期望 %d", preset.ID, preset.Action.Delta, delta)
			}
			delete(want, preset.ID)
		}
		switch preset.ID {
		case "window-opacity-up", "window-opacity-down", "media-volume-up", "media-volume-down", "display-brightness-up", "display-brightness-down":
			t.Errorf("不应保留正反两个可参数化预置: %s", preset.ID)
		}
	}
	if len(want) != 0 {
		t.Fatalf("缺少可参数化预置: %v", want)
	}
}

func TestVolumeKeyPressCountValidation(t *testing.T) {
	for _, delta := range []int{-20, -1, 1, 20} {
		if err := validateAction(Action{Type: ActionVolume, Delta: delta}, 0); err != nil {
			t.Errorf("音量按键 %d 次被拒绝: %v", delta, err)
		}
	}
	for _, delta := range []int{-21, 0, 21} {
		if err := validateAction(Action{Type: ActionVolume, Delta: delta}, 0); err == nil {
			t.Errorf("音量按键 %d 次未被拒绝", delta)
		}
	}
}
