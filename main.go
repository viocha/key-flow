package main

import (
	"embed"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	startupMode := hasArgument(os.Args[1:], "--hidden")

	store, err := NewConfigStore()
	if err != nil {
		log.Fatal(err)
	}
	logFile, logErr := os.OpenFile(filepath.Join(filepath.Dir(store.Path()), "key-flow.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if logErr == nil {
		defer logFile.Close()
		// 无控制台启动时标准错误可能不可写，必须先写入日志文件。
		log.SetOutput(io.MultiWriter(logFile, os.Stderr))
	} else {
		log.Printf("打开运行日志失败: %v", logErr)
	}
	log.Printf("Key Flow 启动: pid=%d, hidden=%t", os.Getpid(), startupMode)

	engine, err := NewInputEngine(store.Snapshot())
	if err != nil {
		log.Fatal(err)
	}
	service := NewAppService(store, engine)

	var manager *application.WebviewWindow
	var managerMu sync.Mutex
	var openManager func()
	var overlay *nativeGestureOverlay
	var brightnessFeedback *nativeBrightnessFeedback
	app := application.New(application.Options{
		Name:        "key-flow",
		Description: "Key Flow",
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.keyflow.desktop",
			ExitCode: 0,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if hasArgument(data.Args, "--hidden") {
					return
				}
				if openManager != nil {
					openManager()
				}
			},
		},
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		OnShutdown: func() {
			engine.Stop()
			destroyTopmostBorders()
			if overlay != nil {
				overlay.Destroy()
			}
			if brightnessFeedback != nil {
				brightnessFeedback.Destroy()
			}
		},
	})
	service.app = app
	app.SetIcon(appIcon)

	openManager = func() {
		managerMu.Lock()
		current := manager
		if current == nil {
			current = app.Window.NewWithOptions(application.WebviewWindowOptions{
				Name:                   "manager",
				Title:                  "Key Flow",
				Width:                  1040,
				Height:                 700,
				MinWidth:               840,
				MinHeight:              560,
				BackgroundColour:       application.NewRGB(255, 255, 255),
				URL:                    "/",
				DevToolsEnabled:        true,
				OpenInspectorOnStartup: true,
				KeyBindings: map[string]func(window application.Window){
					"F12": func(window application.Window) {
						window.OpenDevTools()
					},
					"ctrl+shift+i": func(window application.Window) {
						window.OpenDevTools()
					},
				},
			})
			manager = current
			current.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
				engine.ResetInputCapture()
				managerMu.Lock()
				if manager == current {
					manager = nil
				}
				managerMu.Unlock()
			})
		}
		managerMu.Unlock()
		current.Show().Focus()
	}

	engine.SetGestureOverlayHandler(func(event GestureOverlayEvent) {
		if overlay != nil {
			overlay.Update(event)
		}
	})

	tray := app.SystemTray.New()
	tray.SetIcon(appIcon)
	tray.SetTooltip("Key Flow")
	service.onConfigChanged = func(config AppConfig) {
		tray.SetMenu(buildTrayMenu(app, openManager, service, config))
	}
	tray.SetMenu(buildTrayMenu(app, openManager, service, store.Snapshot()))
	tray.OnClick(func() {
		openManager()
	})

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		overlay = createNativeGestureOverlay()
		brightnessFeedback = createNativeBrightnessFeedback()
		if err := engine.Start(); err != nil {
			log.Printf("启动全局输入引擎失败: %v", err)
		}
		if !startupMode {
			openManager()
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func hasArgument(args []string, target string) bool {
	for _, argument := range args {
		if argument == target {
			return true
		}
	}
	return false
}
