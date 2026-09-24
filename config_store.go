package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ConfigStore struct {
	mu     sync.RWMutex
	path   string
	config AppConfig
}

func NewConfigStore() (*ConfigStore, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("读取用户配置目录失败: %w", err)
	}

	store := &ConfigStore{
		path: filepath.Join(root, "key-flow", "config.json"),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ConfigStore) Path() string {
	return s.path
}

func (s *ConfigStore) Snapshot() AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.config)
}

func (s *ConfigStore) Save(config AppConfig) (AppConfig, error) {
	config = normalizeConfig(config)
	if err := validateConfig(config); err != nil {
		return AppConfig{}, err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return AppConfig{}, fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return AppConfig{}, fmt.Errorf("创建配置目录失败: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, byte(10)), 0o600); err != nil {
		return AppConfig{}, fmt.Errorf("写入临时配置失败: %w", err)
	}
	if err := replaceFile(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return AppConfig{}, fmt.Errorf("保存配置失败: %w", err)
	}

	s.mu.Lock()
	s.config = cloneConfig(config)
	s.mu.Unlock()
	return cloneConfig(config), nil
}

func (s *ConfigStore) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.config = defaultConfig()
		_, err = s.Save(s.config)
		return err
	}
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	var config AppConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}
	config = normalizeConfig(config)
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("配置无效: %w", err)
	}

	s.config = config
	return nil
}

func defaultConfig() AppConfig {
	return AppConfig{
		Enabled:         true,
		ActiveProfileID: "profile-default",
		Settings: Settings{
			DefaultHoldMS:        200,
			DefaultKeyIntervalMS: 0,
			GestureTimeoutMS:     1000,
		},
		Profiles: []Profile{
			{
				ID:                "profile-default",
				Name:              "默认",
				HotkeysEnabled:    true,
				Hotkeys:           []HotkeyRule{},
				Layers:            []TriggerLayer{},
				Gestures:          []GestureGroup{},
				EdgeWheelsEnabled: true,
				EdgeWheelMarginPX: 4,
				EdgeWheels:        []EdgeWheelRule{},
			},
		},
	}
}

func normalizeConfig(config AppConfig) AppConfig {
	for pi := range config.Profiles {
		profile := &config.Profiles[pi]
		profile.Name = strings.TrimSpace(profile.Name)
		if profile.Hotkeys == nil {
			profile.Hotkeys = []HotkeyRule{}
		}
		if profile.Layers == nil {
			profile.Layers = []TriggerLayer{}
		}
		if profile.Gestures == nil {
			profile.Gestures = []GestureGroup{}
		}
		if profile.EdgeWheels == nil {
			profile.EdgeWheels = []EdgeWheelRule{}
		}
		for li := range profile.Layers {
			layer := &profile.Layers[li]
			layer.Name = strings.TrimSpace(layer.Name)
			if layer.MouseShortcuts == nil {
				layer.MouseShortcuts = []MouseShortcutBinding{}
			}
		}
		for gi := range profile.Gestures {
			group := &profile.Gestures[gi]
			if group.Bindings == nil {
				group.Bindings = []GestureBinding{}
			}
			for bi := range group.Bindings {
				group.Bindings[bi].Name = strings.TrimSpace(group.Bindings[bi].Name)
			}
		}
	}
	return config
}

func validateConfig(config AppConfig) error {
	if config.Settings.DefaultHoldMS < 50 || config.Settings.DefaultHoldMS > 1500 {
		return errors.New("默认触发延迟须为 50–1500 毫秒")
	}
	if config.Settings.DefaultKeyIntervalMS < 0 || config.Settings.DefaultKeyIntervalMS > 500 {
		return errors.New("默认按键间隔须为 0–500 毫秒")
	}
	if config.Settings.GestureTimeoutMS < 500 || config.Settings.GestureTimeoutMS > 10000 {
		return errors.New("手势无操作超时须为 500–10000 毫秒")
	}
	if len(config.Profiles) == 0 {
		return errors.New("至少需要一个 Profile")
	}

	activeFound := false
	profileIDs := map[string]struct{}{}
	for _, profile := range config.Profiles {
		if strings.TrimSpace(profile.ID) == "" || strings.TrimSpace(profile.Name) == "" {
			return errors.New("Profile 的 ID 和名称不能为空")
		}
		if _, exists := profileIDs[profile.ID]; exists {
			return fmt.Errorf("Profile ID 重复: %s", profile.ID)
		}
		profileIDs[profile.ID] = struct{}{}
		if profile.ID == config.ActiveProfileID {
			activeFound = true
		}

		hotkeyChords := map[string]struct{}{}
		for _, rule := range profile.Hotkeys {
			if rule.ID == "" {
				return errors.New("快捷键规则的 ID 不能为空")
			}
			var shortcutKeys []string
			if rule.MouseInput != "" {
				if len(rule.Keys) != 0 {
					return errors.New("通用快捷键不能同时配置键盘按键和鼠标输入")
				}
				if !isSupportedMouseShortcutInput(rule.MouseInput) {
					return fmt.Errorf("通用快捷键使用了不支持的鼠标输入: %s", rule.MouseInput)
				}
				if isWheelInput(rule.MouseInput) {
					if err := validateWheelSteps(rule.WheelSteps); err != nil {
						return fmt.Errorf("通用快捷键的滚轮灵敏度无效: %w", err)
					}
				}
				modifiers, err := canonicalModifiers(rule.Modifiers)
				if err != nil {
					return fmt.Errorf("通用快捷键的修饰键无效: %w", err)
				}
				if len(modifiers) == 0 {
					return errors.New("通用鼠标快捷键至少需要一个修饰键")
				}
				for _, input := range mouseShortcutInputs(rule.MouseInput) {
					key, err := mouseShortcutKey(modifiers, input)
					if err != nil {
						return fmt.Errorf("通用快捷键无效: %w", err)
					}
					shortcutKeys = append(shortcutKeys, "mouse:"+key)
				}
			} else {
				if len(rule.Modifiers) != 0 {
					return errors.New("通用快捷键未配置鼠标输入，不能单独设置鼠标修饰键")
				}
				chord, err := canonicalChord(rule.Keys)
				if err != nil {
					return fmt.Errorf("通用快捷键无效: %w", err)
				}
				shortcutKeys = []string{"key:" + chord}
			}
			if rule.Enabled {
				for _, shortcutKey := range shortcutKeys {
					if _, exists := hotkeyChords[shortcutKey]; exists {
						return fmt.Errorf("快捷键冲突: 重复使用 %s", strings.TrimPrefix(strings.TrimPrefix(shortcutKey, "key:"), "mouse:"))
					}
					hotkeyChords[shortcutKey] = struct{}{}
				}
			}
			if err := validateAction(rule.Action, 0); err != nil {
				return fmt.Errorf("通用快捷键动作无效: %w", err)
			}
			if err := validateOppositeMouseAction(rule.MouseInput, rule.OppositeAction); err != nil {
				return fmt.Errorf("通用快捷键反向动作无效: %w", err)
			}
		}

		activeTriggers := map[string]string{}
		for _, layer := range profile.Layers {
			if layer.ID == "" || layer.Name == "" {
				return errors.New("触发键 Layer 的 ID 和名称不能为空")
			}
			if !isSupportedTrigger(layer.Trigger) {
				return fmt.Errorf("不支持的触发键: %s", layer.Trigger)
			}
			if layer.HoldMS < 50 || layer.HoldMS > 1500 {
				return fmt.Errorf("触发键 %q 的延迟须为 50–1500 毫秒", layer.Name)
			}
			if layer.TapBehavior != "original" && layer.TapBehavior != "none" {
				return fmt.Errorf("触发键 %q 的短按行为无效", layer.Name)
			}
			if layer.Enabled {
				if previous, exists := activeTriggers[layer.Trigger]; exists {
					return fmt.Errorf("触发键冲突: %q 与 %q 都使用 %s", previous, layer.Name, layer.Trigger)
				}
				activeTriggers[layer.Trigger] = layer.Name
			}

			if err := validateMouseBindings(layer.Trigger, layer.MouseShortcuts, layer.WindowOperations); err != nil {
				return fmt.Errorf("触发键 %q 的鼠标配置无效: %w", layer.Name, err)
			}

			bindingChords := map[string]struct{}{}
			for _, binding := range layer.Bindings {
				if binding.ID == "" {
					return fmt.Errorf("触发键 %q 存在缺少 ID 的绑定", layer.Name)
				}
				chord, err := canonicalChord(binding.Keys)
				if err != nil {
					return fmt.Errorf("触发键 %q 的绑定无效: %w", layer.Name, err)
				}
				if _, exists := bindingChords[chord]; exists {
					return fmt.Errorf("触发键 %q 存在重复映射: %s", layer.Name, chord)
				}
				bindingChords[chord] = struct{}{}
				if err := validateAction(binding.Action, 0); err != nil {
					return fmt.Errorf("触发键 %q 的动作无效: %w", layer.Name, err)
				}
			}

		}

		gestureTriggers := map[string]struct{}{}
		for _, group := range profile.Gestures {
			if group.ID == "" {
				return errors.New("手势分组的 ID 不能为空")
			}
			if !isSupportedGestureTrigger(group.Trigger) {
				return fmt.Errorf("不支持的手势启动键: %s", group.Trigger)
			}
			if group.StartDistancePX < 4 || group.StartDistancePX > 100 {
				return fmt.Errorf("手势 %s 的启动距离须为 4–100 像素", group.Trigger)
			}
			if group.SegmentDistancePX < 4 || group.SegmentDistancePX > 200 {
				return fmt.Errorf("手势 %s 的分段距离须为 4–200 像素", group.Trigger)
			}
			if group.Enabled {
				if _, exists := gestureTriggers[group.Trigger]; exists {
					return fmt.Errorf("手势启动键重复: %s", group.Trigger)
				}
				gestureTriggers[group.Trigger] = struct{}{}
				if _, exists := activeTriggers[group.Trigger]; exists {
					return fmt.Errorf("手势启动键与触发键冲突: %s", group.Trigger)
				}
			}
			if err := validateGestureBindings(group.Bindings); err != nil {
				return fmt.Errorf("手势 %s 无效: %w", group.Trigger, err)
			}
		}

		if profile.EdgeWheelMarginPX < 1 || profile.EdgeWheelMarginPX > 64 {
			return errors.New("屏幕边缘触发范围须为 1–64 像素")
		}
		edgeInputs := map[string]struct{}{}
		for _, rule := range profile.EdgeWheels {
			if rule.ID == "" {
				return errors.New("屏幕边缘规则的 ID 不能为空")
			}
			if !isSupportedEdgeZone(rule.Zone) {
				return fmt.Errorf("不支持的屏幕边缘区域: %s", rule.Zone)
			}
			if !isSupportedEdgeWheelInput(rule.Input) {
				return fmt.Errorf("不支持的屏幕边缘滚轮方向: %s", rule.Input)
			}
			if err := validateWheelSteps(rule.WheelSteps); err != nil {
				return fmt.Errorf("屏幕边缘 %s 的滚轮灵敏度无效: %w", rule.Zone, err)
			}
			if rule.Enabled {
				key := rule.Zone + ":" + rule.Input
				if _, exists := edgeInputs[key]; exists {
					return fmt.Errorf("屏幕边缘区域与滚轮方向重复: %s", rule.Zone)
				}
				edgeInputs[key] = struct{}{}
			}
			switch rule.Adjustment {
			case EdgeAdjustmentVolume:
				if rule.Step < 1 || rule.Step > 20 {
					return fmt.Errorf("屏幕边缘 %s 的音量按键次数须为 1–20", rule.Zone)
				}
			case EdgeAdjustmentBrightness:
				if rule.Step < 1 || rule.Step > 100 {
					return fmt.Errorf("屏幕边缘 %s 的亮度步长须为 1–100", rule.Zone)
				}
			case EdgeAdjustmentRecentApps, EdgeAdjustmentSwitchApps, EdgeAdjustmentSwitchTabs, EdgeAdjustmentPage:
			case EdgeAdjustmentCustom:
				if rule.Action == nil || rule.OppositeAction == nil {
					return fmt.Errorf("屏幕边缘 %s 的自定义动作不完整", rule.Zone)
				}
				if err := validateAction(*rule.Action, 0); err != nil {
					return fmt.Errorf("屏幕边缘 %s 的第一方向动作无效: %w", rule.Zone, err)
				}
				if err := validateAction(*rule.OppositeAction, 0); err != nil {
					return fmt.Errorf("屏幕边缘 %s 的第二方向动作无效: %w", rule.Zone, err)
				}
			default:
				return fmt.Errorf("屏幕边缘 %s 的调节类型无效: %s", rule.Zone, rule.Adjustment)
			}
		}
	}
	if !activeFound {
		return errors.New("当前 Profile 不存在")
	}
	return nil
}

func validateMouseBindings(trigger string, shortcuts []MouseShortcutBinding, window WindowPointerOperations) error {
	triggerInput := mouseInputForTrigger(trigger)
	used := map[string]string{}
	usedInputs := map[string]string{}

	for _, binding := range shortcuts {
		if !isSupportedMouseShortcutInput(binding.Input) {
			return fmt.Errorf("不支持的鼠标输入: %s", binding.Input)
		}
		if isWheelInput(binding.Input) {
			if err := validateWheelSteps(binding.WheelSteps); err != nil {
				return fmt.Errorf("%s的滚轮灵敏度无效: %w", mouseInputLabel(binding.Input), err)
			}
		}
		if binding.Input == triggerInput {
			return fmt.Errorf("%s 已作为当前触发键", mouseInputLabel(binding.Input))
		}
		label := mouseShortcutBindingLabel(binding.Modifiers, binding.Input)
		for _, input := range mouseShortcutInputs(binding.Input) {
			key, err := mouseShortcutKey(binding.Modifiers, input)
			if err != nil {
				return fmt.Errorf("%s修饰键无效: %w", mouseInputLabel(binding.Input), err)
			}
			if previous, exists := used[key]; exists {
				return fmt.Errorf("鼠标快捷键重复: %s 与 %s", previous, label)
			}
			used[key] = label
			if binding.Enabled {
				usedInputs[input] = label
			}
		}
		if err := validateAction(binding.Action, 0); err != nil {
			return fmt.Errorf("%s动作无效: %w", label, err)
		}
		if err := validateOppositeMouseAction(binding.Input, binding.OppositeAction); err != nil {
			return fmt.Errorf("%s反向动作无效: %w", label, err)
		}
	}

	if !window.Enabled {
		return nil
	}

	for name, input := range map[string]string{
		"拖动窗口":   window.MoveButton,
		"调整窗口尺寸": window.ResizeButton,
	} {
		if input == "" {
			continue
		}
		if !isSupportedMouseButtonInput(input) {
			return fmt.Errorf("%s不支持鼠标输入 %q", name, input)
		}
		if input == triggerInput {
			return fmt.Errorf("%s不能使用当前触发键 %s", name, mouseInputLabel(input))
		}
		if previous, exists := usedInputs[input]; exists {
			return fmt.Errorf("%s与%s冲突", name, previous)
		}
		usedInputs[input] = name
	}

	if window.MoveButton != "" && window.MoveButton == window.ResizeButton {
		return errors.New("拖动窗口与调整窗口尺寸不能使用同一个鼠标键")
	}
	return nil
}

func isWheelInput(input string) bool {
	switch input {
	case "wheelUp", "wheelDown", "wheelLeft", "wheelRight", "wheelVertical", "wheelHorizontal":
		return true
	default:
		return false
	}
}

func isSupportedEdgeWheelInput(input string) bool {
	return input == "wheelVertical" || input == "wheelHorizontal"
}

func mouseShortcutInputs(input string) []string {
	switch input {
	case "wheelVertical":
		return []string{"wheelUp", "wheelDown"}
	case "wheelHorizontal":
		return []string{"wheelLeft", "wheelRight"}
	default:
		return []string{input}
	}
}

func validateOppositeMouseAction(input string, action *Action) error {
	if len(mouseShortcutInputs(input)) == 1 {
		if action != nil {
			return errors.New("单向输入不能配置反向动作")
		}
		return nil
	}
	if action == nil {
		return errors.New("双向输入需要配置两个动作")
	}
	return validateAction(*action, 0)
}

func validateWheelSteps(steps int) error {
	if steps < 1 || steps > 20 {
		return errors.New("每次触发所需滚轮格数须为 1–20")
	}
	return nil
}

func mouseShortcutBindingLabel(modifiers []string, input string) string {
	canonical, _ := canonicalModifiers(modifiers)
	parts := make([]string, 0, len(canonical)+1)
	parts = append(parts, canonical...)
	parts = append(parts, mouseInputLabel(input))
	return strings.Join(parts, " + ")
}

func isSupportedMouseShortcutInput(input string) bool {
	switch input {
	case "left", "right", "middle", "back", "forward", "wheelUp", "wheelDown", "wheelLeft", "wheelRight", "wheelVertical", "wheelHorizontal":
		return true
	default:
		return false
	}
}

func isSupportedMouseButtonInput(input string) bool {
	switch input {
	case "left", "right", "middle", "back", "forward":
		return true
	default:
		return false
	}
}

func mouseInputForTrigger(trigger string) string {
	switch trigger {
	case "MouseRight":
		return "right"
	case "MouseMiddle":
		return "middle"
	case "MouseBack":
		return "back"
	case "MouseForward":
		return "forward"
	default:
		return ""
	}
}

func mouseInputLabel(input string) string {
	switch input {
	case "left":
		return "左键"
	case "right":
		return "右键"
	case "middle":
		return "中键"
	case "back":
		return "后退键"
	case "forward":
		return "前进键"
	case "wheelUp":
		return "滚轮向上"
	case "wheelDown":
		return "滚轮向下"
	case "wheelLeft":
		return "滚轮向左"
	case "wheelRight":
		return "滚轮向右"
	case "wheelVertical":
		return "垂直滚动"
	case "wheelHorizontal":
		return "水平滚动"
	default:
		return input
	}
}

func validateGestureBindings(bindings []GestureBinding) error {
	seen := map[string]string{}
	for _, binding := range bindings {
		if binding.ID == "" {
			return errors.New("手势的 ID 不能为空")
		}
		label := gestureBindingLabel(binding)
		if len(binding.Preview) < 2 {
			return fmt.Errorf("手势 %q 缺少录制路径", label)
		}
		key, err := gestureKey(binding.Directions)
		if err != nil {
			return fmt.Errorf("手势 %q: %w", label, err)
		}
		if binding.Enabled {
			if previous, exists := seen[key]; exists {
				return fmt.Errorf("手势 %q 与 %q 的路径重复", previous, label)
			}
			seen[key] = label
		}
		if err := validateAction(binding.Action, 0); err != nil {
			return fmt.Errorf("手势 %q 的动作无效: %w", label, err)
		}
	}
	return nil
}

func validateAction(action Action, depth int) error {
	if depth > 8 {
		return errors.New("动作序列嵌套过深")
	}
	if action.DelayMS < 0 || action.DelayMS > 60_000 {
		return errors.New("动作延迟须为 0–60000 毫秒")
	}

	switch action.Type {
	case ActionKeyStroke:
		_, err := canonicalChord(action.Keys)
		return err
	case ActionText:
		return nil
	case ActionLaunch:
		if strings.TrimSpace(action.Target) == "" {
			return errors.New("打开目标不能为空")
		}
	case ActionCommand:
		if strings.TrimSpace(action.Program) == "" {
			return errors.New("命令程序不能为空")
		}
		switch action.ArgsMode {
		case "commandLine":
		case "list":
		default:
			return errors.New("命令参数输入方式无效")
		}
	case ActionSequence:
		if len(action.Sequence) == 0 {
			return errors.New("动作序列不能为空")
		}
		for _, child := range action.Sequence {
			if err := validateAction(child, depth+1); err != nil {
				return err
			}
		}
	case ActionEdit:
		switch action.EditCommand {
		case "deleteLine", "insertLineBelow", "insertLineAbove", "selectLine", "selectLineWithBreak":
		default:
			return fmt.Errorf("未知编辑命令: %s", action.EditCommand)
		}
	case ActionWindow:
		switch action.WindowCommand {
		case "close", "terminateProcess", "minimize", "maximize", "restore", "toggleMaximize", "toggleVerticalMaximize", "previousWindow", "nextWindow", "restoreMinimized":
		case "opacity", "scale":
			if action.Delta == 0 || action.Delta < -100 || action.Delta > 100 {
				name := "透明度"
				if action.WindowCommand == "scale" {
					name = "缩放"
				}
				return fmt.Errorf("窗口%s变化须为 -100–100 且不能为 0", name)
			}
		case "alwaysOnTop":
			if _, err := parseWindowBorderColor(action.WindowBorderColor); err != nil {
				return err
			}
			if action.WindowBorderSize < 1 || action.WindowBorderSize > 12 {
				return errors.New("置顶边框尺寸须为 1–12")
			}
		default:
			return fmt.Errorf("未知窗口命令: %s", action.WindowCommand)
		}
	case ActionMouse:
		switch action.MouseCommand {
		case "leftClick", "rightClick", "middleClick", "backClick", "forwardClick", "doubleClick":
		case "wheelUp", "wheelDown", "wheelLeft", "wheelRight":
			if action.Delta < 1 || action.Delta > 20 {
				return errors.New("鼠标滚动格数须为 1–20")
			}
		default:
			return fmt.Errorf("未知鼠标命令: %s", action.MouseCommand)
		}
	case ActionVolume:
		if action.Delta == 0 || action.Delta < -20 || action.Delta > 20 {
			return errors.New("音量调节值须为 -20–20 且不能为 0")
		}
	case ActionBrightness:
		if action.Delta == 0 || action.Delta < -100 || action.Delta > 100 {
			return errors.New("亮度变化须为 -100–100 且不能为 0")
		}
	default:
		return fmt.Errorf("未知动作类型: %s", action.Type)
	}
	return nil
}

func cloneConfig(config AppConfig) AppConfig {
	data, err := json.Marshal(config)
	if err != nil {
		return config
	}
	var copy AppConfig
	if err := json.Unmarshal(data, &copy); err != nil {
		return config
	}
	return copy
}
