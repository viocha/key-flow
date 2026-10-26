//go:build windows

package main

import (
	"testing"
	"time"
	"unsafe"
)

func waitForGestureOverlay(t *testing.T, overlay *nativeGestureOverlay, ready func() bool) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	// 原生窗口通过消息队列异步绘制，等待明确的帧状态，不假定绘制耗时。
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		if overlay.mu.TryLock() {
			done := ready()
			overlay.mu.Unlock()
			if done {
				return
			}
		}
		select {
		case <-deadline.C:
			t.Fatal("手势浮层未完成绘制，窗口线程可能阻塞")
		case <-tick.C:
		}
	}
}

func TestNativeGestureOverlayRenderingAndLifecycle(t *testing.T) {
	overlay := createNativeGestureOverlay()
	if overlay == nil {
		t.Fatal("无法创建原生手势浮层")
	}
	t.Cleanup(func() {
		finished := make(chan struct{})
		go func() {
			overlay.Destroy()
			close(finished)
		}()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Error("关闭浮层不能等待绘制锁或失去响应的窗口线程")
		}
	})
	var processID uint32
	threadID, _, _ := procGetWindowThreadProcessID.Call(uintptr(overlay.hwnd), uintptr(unsafe.Pointer(&processID)))
	if threadID != overlay.threadID {
		t.Fatal("浮层窗口应属于固定的消息线程")
	}

	overlay.Update(GestureOverlayEvent{
		Visible: true,
		Reset:   true,
		Points:  []GesturePoint{{X: 200, Y: 200}, {X: 500, Y: 200}},
		Label:   "测试动作",
	})
	waitForGestureOverlay(t, overlay, func() bool {
		return overlay.visible && overlay.label == "测试动作" && !overlayRectEmpty(overlay.pathRect) && !overlayRectEmpty(overlay.labelRect)
	})
	getPixel := gestureOverlayGDI32.NewProc("GetPixel")
	overlay.mu.Lock()
	pathPixel, _, _ := getPixel.Call(overlay.backDC, 350, 200)
	labelPixel, _, _ := getPixel.Call(overlay.backDC, uintptr(overlay.labelRect.Left+20), uintptr(overlay.labelRect.Top+20))
	overlay.mu.Unlock()
	for _, pixel := range []uintptr{pathPixel, labelPixel} {
		if pixel == gestureOverlayColorKey || pixel == 0xFFFFFFFF {
			t.Fatal("路径和动作提示必须实际绘制到位图")
		}
	}

	for i := range 80 {
		overlay.Update(GestureOverlayEvent{
			Visible: true,
			Points:  []GesturePoint{{X: float64(501 + i), Y: 200}},
			Label:   "连续移动",
		})
	}
	waitForGestureOverlay(t, overlay, func() bool {
		return overlay.label == "连续移动" && len(overlay.points) > 0 && overlay.points[len(overlay.points)-1].X == 580
	})

	overlay.Update(GestureOverlayEvent{Visible: false})
	waitForGestureOverlay(t, overlay, func() bool {
		return !overlay.visible && overlay.label == "" && len(overlay.points) == 0
	})
	overlay.mu.Lock()
	cleared, _, _ := getPixel.Call(overlay.backDC, 350, 200)
	overlay.mu.Unlock()
	if cleared != gestureOverlayColorKey {
		t.Fatal("手势结束后应清除路径")
	}
	// 合并尚未绘制的旧手势、隐藏事件和新手势时，不能混入旧路径。
	overlay.Update(GestureOverlayEvent{Visible: true, Reset: true, Points: []GesturePoint{{X: 200, Y: 300}, {X: 300, Y: 300}}, Label: "旧手势"})
	overlay.Update(GestureOverlayEvent{Visible: false})
	overlay.Update(GestureOverlayEvent{Visible: true, Reset: true, Points: []GesturePoint{{X: 400, Y: 400}, {X: 500, Y: 400}}, Label: "新手势"})
	waitForGestureOverlay(t, overlay, func() bool {
		return overlay.label == "新手势" && len(overlay.points) == 2 && overlay.points[0].X == 400
	})
}
