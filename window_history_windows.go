//go:build windows

package main

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
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

// 窗口枚举及前台窗口事件使用独立线程，避免阻塞键鼠 Hook 的消息循环。
func startWindowHistory() (func(), error) {
	ready := make(chan error, 1)
	finished := make(chan struct{})
	var threadID uintptr
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(finished)
		threadID, _, _ = procGetCurrentThreadID.Call()
		var message msg
		procPeekMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, pmNoRemove)
		stop, err := installWindowHistory()
		ready <- err
		if err != nil {
			return
		}
		defer stop()
		for {
			result, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
			if int32(result) <= 0 {
				return
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
		}
	}()
	if err := <-ready; err != nil {
		<-finished
		return nil, err
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			procPostThreadMessageW.Call(threadID, 0x0012, 0, 0)
			<-finished
		})
	}, nil
}

func installWindowHistory() (func(), error) {
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
