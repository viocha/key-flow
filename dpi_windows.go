//go:build windows

package main

import (
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetMonitorDPI = windows.NewLazySystemDLL("shcore.dll").NewProc("GetDpiForMonitor")
var procMonitorFromWindow = user32.NewProc("MonitorFromWindow")

// 配置与布局使用 px；只有调用 Windows 坐标和绘制 API 时换算成物理像素。
func dpiScaleAtPoint(pt point) float64 {
	monitor := monitorAtPoint(pt)
	return monitorDPIScale(monitor)
}

func monitorDPIScale(monitor windows.Handle) float64 {
	if monitor == 0 {
		return 1
	}
	var dpiX, dpiY uint32
	result, _, _ := procGetMonitorDPI.Call(uintptr(monitor), 0,
		uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
	if int32(result) < 0 || dpiX == 0 {
		return 1
	}
	return float64(dpiX) / 96
}

func physicalPixels(px, scale float64) int32 {
	if scale <= 0 {
		scale = 1
	}
	return int32(math.Round(px * scale))
}

func windowScale(hwnd windows.Handle) float64 {
	monitor, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), monitorDefaultToNearest)
	return monitorDPIScale(windows.Handle(monitor))
}
