//go:build windows

package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
)

const windowHistoryLimit = 128

// 激活历史按实际访问顺序记录；导航只移动游标。
type activationHistory struct {
	items []windows.Handle
	index int
}

func (h *activationHistory) observe(hwnd windows.Handle, available []windows.Handle) {
	found := false
	for _, item := range available {
		if item == hwnd {
			found = true
			break
		}
	}
	if !found || (len(h.items) > 0 && h.items[h.index] == hwnd) {
		return
	}
	if len(h.items) > 0 {
		h.items = h.items[:h.index+1]
	}
	h.items = append(h.items, hwnd)
	if len(h.items) > windowHistoryLimit {
		h.items = h.items[len(h.items)-windowHistoryLimit:]
	}
	h.index = len(h.items) - 1
}

func (h *activationHistory) prune(available []windows.Handle) {
	valid := make(map[windows.Handle]bool, len(available))
	for _, hwnd := range available {
		valid[hwnd] = true
	}
	items := make([]windows.Handle, 0, len(h.items))
	index := -1
	for i, hwnd := range h.items {
		if !valid[hwnd] {
			continue
		}
		// 删除窗口后合并相邻的同一窗口。
		if len(items) == 0 || items[len(items)-1] != hwnd {
			items = append(items, hwnd)
		}
		if i <= h.index {
			index = len(items) - 1
		}
	}
	if index < 0 {
		index = 0
	}
	h.items, h.index = items, index
}

func (h *activationHistory) target(direction int) (windows.Handle, int) {
	index := h.index + direction
	if index < 0 || index >= len(h.items) {
		return 0, h.index
	}
	return h.items[index], index
}

var (
	procHistorySetWinEventHook = user32.NewProc("SetWinEventHook")
	procHistoryUnhookWinEvent  = user32.NewProc("UnhookWinEvent")
	historyWinEventCallback    = syscall.NewCallback(func(hook, event, hwnd uintptr, objectID, childID int32, threadID, eventTime uint32) uintptr {
		windowSwitchMu.Lock()
		defer windowSwitchMu.Unlock()
		current, _, _ := procGetForegroundWindow.Call()
		// 忽略过期通知；导航的重复通知由当前游标去重。
		if !windowNavigating && hwnd != 0 && current == hwnd {
			windowHistory.observe(windows.Handle(hwnd), switchableWindows())
		}
		return 0
	})
)

// 在拥有消息循环的线程上安装和卸载事件 Hook。
func startWindowHistory() (func(), error) {
	const eventSystemForeground = 0x0003
	hook, _, err := procHistorySetWinEventHook.Call(eventSystemForeground, eventSystemForeground, 0, historyWinEventCallback, 0, 0, 0)
	if hook == 0 {
		return nil, fmt.Errorf("安装前台窗口事件 Hook 失败: %w", err)
	}
	windowSwitchMu.Lock()
	windowHistory = activationHistory{}
	current, _, _ := procGetForegroundWindow.Call()
	windowHistory.observe(windows.Handle(current), switchableWindows())
	windowSwitchMu.Unlock()
	return func() { procHistoryUnhookWinEvent.Call(hook) }, nil
}
