//go:build windows

package main

import (
	"golang.org/x/sys/windows"
	"testing"
)

func TestWindowHistoryBackAndForward(t *testing.T) {
	available := []windows.Handle{1, 2, 3, 4, 5, 6}
	h := activationHistory{}
	h.observe(1, available)
	h.observe(6, available)
	target, index := h.target(-1)
	if target != 1 {
		t.Fatalf("A 切到 F 后，上一个应为 A，得到 %d", target)
	}
	h.index = index
	h.observe(1, available)
	target, index = h.target(1)
	if target != 6 || len(h.items) != 2 {
		t.Fatalf("下一个应为 F，历史 %v，目标 %d", h.items, target)
	}
	h.index = index
	if target, _ := h.target(1); target != 0 {
		t.Fatal("前进到边界后应停止")
	}
	h.index = 0
	if target, _ := h.target(-1); target != 0 {
		t.Fatal("后退到边界后应停止")
	}
	h.observe(3, available)
	if target, _ := h.target(1); target != 0 {
		t.Fatal("手动选择 C 后应清除前进历史")
	}
	if target, _ := h.target(-1); target != 1 {
		t.Fatal("C 的上一个应为 A")
	}
}

func TestWindowHistoryPrunesUnavailableWindows(t *testing.T) {
	h := activationHistory{items: []windows.Handle{1, 2, 1, 3, 4}, index: 3}
	h.prune([]windows.Handle{1, 3, 4})
	if len(h.items) != 3 || h.index != 1 {
		t.Fatalf("清理后历史 %v，游标 %d", h.items, h.index)
	}
	if target, _ := h.target(-1); target != 1 {
		t.Fatal("应跳过关闭的窗口")
	}
	if target, _ := h.target(1); target != 4 {
		t.Fatal("应保留前进历史")
	}
	h.observe(9, []windows.Handle{1, 3, 4})
	if len(h.items) != 3 {
		t.Fatal("不可切换窗口不应进入历史")
	}
	h.prune(nil)
	if target, _ := h.target(-1); target != 0 {
		t.Fatal("空历史应保持不变")
	}
}

func TestWindowHistoryBounded(t *testing.T) {
	h := activationHistory{}
	for i := 0; i < windowHistoryLimit+10; i++ {
		h.observe(windows.Handle(i%2+1), []windows.Handle{1, 2})
	}
	if len(h.items) != windowHistoryLimit || h.index != windowHistoryLimit-1 {
		t.Fatalf("历史长度 %d，游标 %d", len(h.items), h.index)
	}
}

func TestWindowHistoryRepeatedVisits(t *testing.T) {
	h := activationHistory{}
	available := []windows.Handle{1, 2, 3}
	for _, hwnd := range []windows.Handle{1, 2, 1, 3} {
		h.observe(hwnd, available)
	}
	for _, want := range []windows.Handle{1, 2, 1} {
		target, index := h.target(-1)
		if target != want {
			t.Fatalf("后退目标 %d，期望 %d", target, want)
		}
		h.index = index
		h.observe(target, available)
	}
	for _, want := range []windows.Handle{2, 1, 3} {
		target, index := h.target(1)
		if target != want {
			t.Fatalf("前进目标 %d，期望 %d", target, want)
		}
		h.index = index
		h.observe(target, available)
	}
}
