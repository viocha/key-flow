//go:build windows

package main

import (
	"os"
	"strconv"
	"testing"

	"golang.org/x/sys/windows"
)

// 指定现有窗口，验证实际 Win32/UWP 窗口的最小化与恢复。
func TestRestoreMinimizedExternalWindow(t *testing.T) {
	value := os.Getenv("KEY_FLOW_TEST_WINDOW_HWND")
	if value == "" {
		t.Skip("未指定窗口恢复测试目标")
	}
	raw, err := strconv.ParseUint(value, 0, 64)
	if err != nil {
		t.Fatal(err)
	}
	hwnd := windows.Handle(raw)
	if valid, _, _ := procIsWindow.Call(uintptr(hwnd)); valid == 0 {
		t.Fatal("测试窗口无效")
	}
	minimizedWindowsMu.Lock()
	previous := append([]windows.Handle(nil), minimizedWindows...)
	minimizedWindows = nil
	minimizedWindowsMu.Unlock()
	t.Cleanup(func() {
		minimizedWindowsMu.Lock()
		minimizedWindows = previous
		minimizedWindowsMu.Unlock()
	})
	t.Cleanup(func() { procShowWindow.Call(uintptr(hwnd), swRestore) })
	if err := executeWindowActionForWindow(Action{Type: ActionWindow, WindowCommand: "minimize"}, hwnd); err != nil {
		t.Fatal(err)
	}
	iconic, _, _ := procIsIconic.Call(uintptr(hwnd))
	if iconic == 0 {
		t.Fatal("目标窗口未进入最小化状态")
	}
	if err := executeWindowActionForWindow(Action{Type: ActionWindow, WindowCommand: "restoreMinimized"}, 0); err != nil {
		t.Fatal(err)
	}
	if iconic, _, _ := procIsIconic.Call(uintptr(hwnd)); iconic != 0 {
		t.Fatal("窗口恢复后不应保持最小化")
	}
	active, _, _ := procGetForegroundWindow.Call()
	if active != uintptr(hwnd) {
		t.Fatal("恢复的窗口应处于前台")
	}
	minimizedWindowsMu.Lock()
	remaining := len(minimizedWindows)
	minimizedWindowsMu.Unlock()
	if remaining != 0 {
		t.Fatal("恢复成功后应移出最小化记录")
	}
}
