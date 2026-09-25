//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestNextSwitchWindowCyclesBothDirections(t *testing.T) {
	order := []windows.Handle{1, 2, 3}
	for _, tc := range []struct {
		current   windows.Handle
		direction int
		want      windows.Handle
	}{
		{1, 1, 2},
		{2, 1, 3},
		{3, 1, 1},
		{1, -1, 3},
		{3, -1, 2},
		{0, 1, 1},
		{0, -1, 3},
		{4, 1, 1},
	} {
		if got := nextSwitchWindow(order, tc.current, tc.direction); got != tc.want {
			t.Errorf("current=%d direction=%d: got %d, want %d", tc.current, tc.direction, got, tc.want)
		}
	}
	if got := nextSwitchWindow([]windows.Handle{1}, 1, 1); got != 0 {
		t.Errorf("只有当前窗口时应保持不变，得到 %d", got)
	}
	if got := nextSwitchWindow([]windows.Handle{1}, 0, 1); got != 1 {
		t.Errorf("当前窗口不可切换时应激活唯一窗口，得到 %d", got)
	}
}

func TestSameWindowsIgnoresActivationOrder(t *testing.T) {
	a := []windows.Handle{1, 2, 3}
	if !sameWindows(a, []windows.Handle{3, 1, 2}) {
		t.Fatal("激活窗口后顺序改变不应重置切换序列")
	}
	if sameWindows(a, []windows.Handle{1, 2, 4}) {
		t.Fatal("窗口集合改变后应重置切换序列")
	}
}
