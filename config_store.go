package main

import (
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
	if err := json.Unmarshal(data, &config); err != nil {
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
			DefaultHoldMS:    200,
			GestureTimeoutMS: 1000,
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
	if config.Settings.DefaultHoldMS == 0 {
		config.Settings.DefaultHoldMS = 200
	}
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
		for hi := range profile.Hotkeys {
			profile.Hotkeys[hi].Name = strings.TrimSpace(profile.Hotkeys[hi].Name)
		}
		for li := range profile.Layers {
			layer := &profile.Layers[li]
			layer.Name = strings.TrimSpace(layer.Name)
			if layer.HoldMS == 0 {
				layer.HoldMS = config.Settings.DefaultHoldMS
			}
			if layer.TapBehavior == "" {
				layer.TapBehavior = "original"
			}
			if layer.MouseShortcuts == nil {
				layer.MouseShortcuts = []MouseShortcutBinding{}
			}
			if layer.WindowOperations.ScaleStepPercent == 0 {
				layer.WindowOperations.ScaleStepPercent = 5
			}
		}
		for gi := range profile.Gestures {
			group := &profile.Gestures[gi]
			if group.StartDistancePX == 0 {
				group.StartDistancePX = 18
			}
			if group.SegmentDistancePX == 0 {
				group.SegmentDistancePX = 28
			}
			if group.Bindings == nil {
				group.Bindings = []GestureBinding{}
			}
			for bi := range group.Bindings {
				group.Bindings[bi].Name = strings.TrimSpace(group.Bindings[bi].Name)
			}
		}
		for ei := range profile.EdgeWheels {
			rule := &profile.EdgeWheels[ei]
			if rule.Adjustment == "" {
				rule.Adjustment = EdgeAdjustmentVolume
			}
			if rule.Step == 0 {
				if rule.Adjustment == EdgeAdjustmentBrightness {
					rule.Step = 5
				} else {
					rule.Step = 1
				}
			}
		}
	}
	return config
}

func validateConfig(config AppConfig) error {
	if config.Settings.DefaultHoldMS < 50 || config.Settings.DefaultHoldMS > 1500 {
		return errors.New("默认触发延迟须为 50–1500 毫秒")
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

		hotkeyChords := map[string]string{}
		for _, rule := range profile.Hotkeys {
			if rule.ID == "" || rule.Name == "" {
				return errors.New("快捷键规则的 ID 和名称不能为空")
			}
			chord, err := canonicalChord(rule.Keys)
			if err != nil {
				return fmt.Errorf("快捷键 %q 无效: %w", rule.Name, err)
			}
			if rule.Enabled {
				if previous, exists := hotkeyChords[chord]; exists {
					return fmt.Errorf("快捷键冲突: %q 与 %q 都使用 %s", previous, rule.Name, chord)
				}
				hotkeyChords[chord] = rule.Name
			}
			if err := validateAction(rule.Action, 0); err != nil {
				return fmt.Errorf("快捷键 %q 的动作无效: %w", rule.Name, err)
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
		edgeZones := map[string]struct{}{}
		for _, rule := range profile.EdgeWheels {
			if rule.ID == "" {
				return errors.New("屏幕边缘规则的 ID 不能为空")
			}
			if !isSupportedEdgeZone(rule.Zone) {
				return fmt.Errorf("不支持的屏幕边缘区域: %s", rule.Zone)
			}
			if rule.Enabled {
				if _, exists := edgeZones[rule.Zone]; exists {
					return fmt.Errorf("屏幕边缘区域重复: %s", rule.Zone)
				}
				edgeZones[rule.Zone] = struct{}{}
			}
			switch rule.Adjustment {
			case EdgeAdjustmentVolume:
				if rule.Step < 1 || rule.Step > 20 {
					return fmt.Errorf("屏幕边缘 %s 的音量步长须为 1–20", rule.Zone)
				}
			case EdgeAdjustmentBrightness:
				if rule.Step < 1 || rule.Step > 100 {
					return fmt.Errorf("屏幕边缘 %s 的亮度步长须为 1–100", rule.Zone)
				}
			case EdgeAdjustmentRecentApps:
			case EdgeAdjustmentBidirectionalKeys:
				if _, err := canonicalChord(rule.UpKeys); err != nil {
					return fmt.Errorf("屏幕边缘 %s 的滚轮向上快捷键无效: %w", rule.Zone, err)
				}
				if _, err := canonicalChord(rule.DownKeys); err != nil {
					return fmt.Errorf("屏幕边缘 %s 的滚轮向下快捷键无效: %w", rule.Zone, err)
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

	for _, binding := range shortcuts {
		if !isSupportedMouseShortcutInput(binding.Input) {
			return fmt.Errorf("不支持的鼠标输入: %s", binding.Input)
		}
		if binding.Input == triggerInput {
			return fmt.Errorf("%s 已作为当前触发键", mouseInputLabel(binding.Input))
		}
		if previous, exists := used[binding.Input]; exists {
			return fmt.Errorf("鼠标输入重复: %s 与 %s", previous, mouseInputLabel(binding.Input))
		}
		used[binding.Input] = mouseInputLabel(binding.Input)
		if err := validateAction(binding.Action, 0); err != nil {
			return fmt.Errorf("%s动作无效: %w", mouseInputLabel(binding.Input), err)
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
		if previous, exists := used[input]; exists {
			return fmt.Errorf("%s与%s冲突", name, previous)
		}
		used[input] = name
	}

	if window.MoveButton != "" && window.MoveButton == window.ResizeButton {
		return errors.New("拖动窗口与调整窗口尺寸不能使用同一个鼠标键")
	}
	if window.ScaleWheel {
		if window.ScaleStepPercent < 1 || window.ScaleStepPercent > 50 {
			return errors.New("窗口中心缩放步长须为 1–50%")
		}
		if _, exists := used["wheelUp"]; exists {
			return errors.New("窗口中心缩放与滚轮向上快捷键冲突")
		}
		if _, exists := used["wheelDown"]; exists {
			return errors.New("窗口中心缩放与滚轮向下快捷键冲突")
		}
	}
	return nil
}

func isSupportedMouseShortcutInput(input string) bool {
	switch input {
	case "left", "right", "middle", "back", "forward", "wheelUp", "wheelDown":
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
	case ActionWindow:
		switch action.WindowCommand {
		case "close", "terminateProcess", "minimize", "maximize", "restore", "toggleMaximize", "previousWindow", "nextWindow", "restoreMinimized":
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
	case ActionVolume:
		if action.Delta == 0 || action.Delta < -20 || action.Delta > 20 {
			return errors.New("音量变化步数须为 -20–20 且不能为 0")
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
