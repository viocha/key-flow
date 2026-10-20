package main

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestInputEngineRestartKeepsWorkers(t *testing.T) {
	engine, err := NewInputEngine(defaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()
	var calls []string
	engine.startPlatform = func(*InputEngine) (func(), error) {
		calls = append(calls, "启动")
		return func() { calls = append(calls, "停止") }, nil
	}
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	engine.BeginInputCapture()
	if err := engine.Restart(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"启动", "停止", "启动"}) {
		t.Fatalf("监听生命周期顺序错误: %v", calls)
	}
	if engine.InputCaptureActive() || !engine.Status().Running {
		t.Fatal("重启应清除录制状态并恢复监听运行状态")
	}
	finished := make(chan struct{})
	engine.enqueueTask(func() { close(finished) })
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("重启后动作队列无法执行任务")
	}
	engine.Stop()
	engine.Stop()
	if err := engine.Restart(); err == nil {
		t.Fatal("应用关闭后不应重新启动监听")
	}
}

func TestEnableReinstallsInputHooks(t *testing.T) {
	config := defaultConfig()
	engine, err := NewInputEngine(config)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()
	starts, stops := 0, 0
	engine.startPlatform = func(*InputEngine) (func(), error) {
		starts++
		return func() { stops++ }, nil
	}
	if err := engine.Start(); err != nil {
		t.Fatal(err)
	}
	if err := engine.UpdateConfig(config); err != nil {
		t.Fatal(err)
	}
	if starts != 1 {
		t.Fatal("普通配置保存不应重建监听")
	}
	config.Enabled = false
	if err := engine.UpdateConfig(config); err != nil {
		t.Fatal(err)
	}
	config.Enabled = true
	if err := engine.UpdateConfig(config); err != nil {
		t.Fatal(err)
	}
	if starts != 2 || stops != 1 {
		t.Fatalf("重新启用应卸载并重装 Hook: 启动 %d 次，停止 %d 次", starts, stops)
	}
}

func TestInputEngineRecoversFromInstallFailure(t *testing.T) {
	config := defaultConfig()
	engine, err := NewInputEngine(config)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()
	failure := errors.New("模拟 Hook 安装失败")
	engine.startPlatform = func(*InputEngine) (func(), error) { return nil, failure }
	if err := engine.Start(); !errors.Is(err, failure) {
		t.Fatalf("应报告安装失败: %v", err)
	}
	if engine.Status().Running || engine.Status().Error == "" {
		t.Fatal("安装失败应记录状态")
	}
	engine.startPlatform = func(*InputEngine) (func(), error) { return func() {}, nil }
	if err := engine.UpdateConfig(config); err != nil {
		t.Fatal(err)
	}
	if !engine.Status().Running || engine.Status().Error != "" {
		t.Fatal("重新启用应恢复安装并清除错误")
	}
}

func TestConcurrentInputEngineRestart(t *testing.T) {
	engine, err := NewInputEngine(defaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Stop()
	active := 0
	engine.startPlatform = func(*InputEngine) (func(), error) {
		active++
		if active != 1 {
			t.Error("监听实例不应重叠")
		}
		return func() { active-- }, nil
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if err := engine.Restart(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	engine.Stop()
	if active != 0 {
		t.Fatal("关闭后不应保留监听")
	}
}
