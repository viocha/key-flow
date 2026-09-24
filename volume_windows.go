package main

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

func adjustSystemVolume(delta int) error {
	if delta == 0 {
		return nil
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		var comError *ole.OleError
		if !errors.As(err, &comError) || comError.Code() != 1 {
			return fmt.Errorf("初始化音频设备失败: %w", err)
		}
	}
	defer ole.CoUninitialize()

	unknown, err := ole.CreateInstance(wca.CLSID_MMDeviceEnumerator, wca.IID_IMMDeviceEnumerator)
	if err != nil {
		return fmt.Errorf("获取音频设备枚举器失败: %w", err)
	}
	enumerator := (*wca.IMMDeviceEnumerator)(unsafe.Pointer(unknown))
	defer enumerator.Release()

	var device *wca.IMMDevice
	if err := enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &device); err != nil {
		return fmt.Errorf("获取默认播放设备失败: %w", err)
	}
	defer device.Release()

	var endpoint *wca.IAudioEndpointVolume
	hr, _, _ := syscall.SyscallN(
		device.VTable().Activate,
		uintptr(unsafe.Pointer(device)),
		uintptr(unsafe.Pointer(wca.IID_IAudioEndpointVolume)),
		uintptr(wca.CLSCTX_ALL),
		0,
		uintptr(unsafe.Pointer(&endpoint)),
	)
	if int32(hr) < 0 {
		return fmt.Errorf("打开音量控制失败: %w", ole.NewError(hr))
	}
	defer endpoint.Release()

	var current float32
	if err := endpoint.GetMasterVolumeLevelScalar(&current); err != nil {
		return fmt.Errorf("读取当前音量失败: %w", err)
	}
	percent := int(math.Round(float64(current) * 100))
	target := max(0, min(100, percent+delta))
	if target == percent {
		return nil
	}
	if err := endpoint.SetMasterVolumeLevelScalar(float32(target)/100, nil); err != nil {
		return fmt.Errorf("调整音量失败: %w", err)
	}
	return nil
}
