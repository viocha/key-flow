package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var modifierOrder = map[string]int{
	"Win":   0,
	"Ctrl":  1,
	"Alt":   2,
	"Shift": 3,
}

func canonicalChord(keys []string) (string, error) {
	if len(keys) == 0 {
		return "", errors.New("按键不能为空")
	}

	seen := map[string]struct{}{}
	modifiers := make([]string, 0, 4)
	mainKeys := make([]string, 0, 1)
	for _, raw := range keys {
		key := normalizeKeyName(raw)
		if key == "" {
			continue
		}
		if !isSupportedKey(key) {
			return "", fmt.Errorf("不支持的按键: %s", raw)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := modifierOrder[key]; ok {
			modifiers = append(modifiers, key)
		} else {
			mainKeys = append(mainKeys, key)
		}
	}
	if len(mainKeys) != 1 {
		return "", errors.New("组合键必须且只能包含一个主键")
	}

	sort.Slice(modifiers, func(i, j int) bool {
		return modifierOrder[modifiers[i]] < modifierOrder[modifiers[j]]
	})
	return strings.Join(append(modifiers, mainKeys[0]), "+"), nil
}

func canonicalModifiers(keys []string) ([]string, error) {
	seen := map[string]struct{}{}
	modifiers := make([]string, 0, 4)
	for _, raw := range keys {
		key := normalizeKeyName(raw)
		if key == "" {
			continue
		}
		if _, ok := modifierOrder[key]; !ok {
			return nil, fmt.Errorf("不支持的修饰键: %s", raw)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		modifiers = append(modifiers, key)
	}
	sort.Slice(modifiers, func(i, j int) bool {
		return modifierOrder[modifiers[i]] < modifierOrder[modifiers[j]]
	})
	return modifiers, nil
}

func mouseShortcutKey(modifiers []string, input string) (string, error) {
	canonical, err := canonicalModifiers(modifiers)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(canonical)+1)
	parts = append(parts, canonical...)
	parts = append(parts, input)
	return strings.Join(parts, "+"), nil
}

func normalizeKeyName(value string) string {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	switch lower {
	case "ctrl", "control", "controlleft", "controlright":
		return "Ctrl"
	case "shift", "shiftleft", "shiftright":
		return "Shift"
	case "alt", "menu", "altleft", "altright":
		return "Alt"
	case "win", "meta", "metaleft", "metaright":
		return "Win"
	case "capslock":
		return "CapsLock"
	case "esc", "escape":
		return "Escape"
	case "return", "enter":
		return "Enter"
	case "space", "spacebar", " ":
		return "Space"
	case "backspace":
		return "Backspace"
	case "delete", "del":
		return "Delete"
	case "insert", "ins":
		return "Insert"
	case "home":
		return "Home"
	case "end":
		return "End"
	case "pageup":
		return "PageUp"
	case "pagedown":
		return "PageDown"
	case "arrowleft", "left":
		return "ArrowLeft"
	case "arrowright", "right":
		return "ArrowRight"
	case "arrowup", "up":
		return "ArrowUp"
	case "arrowdown", "down":
		return "ArrowDown"
	case "tab":
		return "Tab"
	case "printscreen", "prtsc":
		return "PrintScreen"
	case "scrolllock":
		return "ScrollLock"
	case "pause", "break":
		return "Pause"
	case "numlock":
		return "NumLock"
	case "contextmenu", "apps":
		return "ContextMenu"
	case "volumeup":
		return "VolumeUp"
	case "volumedown":
		return "VolumeDown"
	case "volumemute":
		return "VolumeMute"
	case "mediaplaypause":
		return "MediaPlayPause"
	case "medianext":
		return "MediaNext"
	case "mediaprev", "mediaprevious":
		return "MediaPrev"
	}

	if len(value) == 1 {
		ch := value[0]
		if ch >= 'a' && ch <= 'z' {
			return strings.ToUpper(value)
		}
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			return value
		}
	}

	if len(lower) >= 2 && lower[0] == 'f' {
		if n, err := strconv.Atoi(lower[1:]); err == nil && n >= 1 && n <= 24 {
			return "F" + strconv.Itoa(n)
		}
	}
	return value
}

func isSupportedKey(key string) bool {
	if _, ok := modifierOrder[key]; ok {
		return true
	}
	if len(key) == 1 {
		ch := key[0]
		return (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
	}
	switch key {
	case "CapsLock", "Escape", "Enter", "Space", "Backspace", "Delete", "Insert",
		"Home", "End", "PageUp", "PageDown", "ArrowLeft", "ArrowRight", "ArrowUp",
		"ArrowDown", "Tab", "VolumeUp", "VolumeDown", "VolumeMute",
		"MediaPlayPause", "MediaNext", "MediaPrev", "PrintScreen", "ScrollLock",
		"Pause", "NumLock", "ContextMenu":
		return true
	}
	if strings.HasPrefix(key, "F") {
		n, err := strconv.Atoi(strings.TrimPrefix(key, "F"))
		return err == nil && n >= 1 && n <= 24
	}
	return false
}

func isSupportedTrigger(trigger string) bool {
	switch trigger {
	case "CapsLock", "MouseRight", "MouseMiddle", "MouseBack", "MouseForward":
		return true
	default:
		return false
	}
}

func isSupportedGestureTrigger(trigger string) bool {
	switch trigger {
	case "MouseRight", "MouseMiddle", "MouseBack", "MouseForward":
		return true
	default:
		return false
	}
}
