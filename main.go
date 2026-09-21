package main

import (
	"embed"
	"log"
	"os"

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

	engine, err := NewInputEngine(store.Snapshot())
	if err != nil {
		log.Fatal(err)
	}
	service := NewAppService(store, engine)

	var manager *application.WebviewWindow
	var overlay *nativeGestureOverlay
	app := application.New(application.Options{
		Name:        "key-flow",
		Description: "Key Flow",
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "dev.keyflow.desktop",
			ExitCode: 0,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if hasArgument(data.Args, "--hidden") {
					return
				}
				if manager != nil {
					manager.Show().Focus()
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
		},
	})
	service.app = app
	app.SetIcon(appIcon)

	manager = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                   "manager",
		Title:                  "Key Flow",
		Width:                  1040,
		Height:                 700,
		MinWidth:               840,
		MinHeight:              560,
		BackgroundColour:       application.NewRGB(255, 255, 255),
		Hidden:                 startupMode,
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

	engine.SetGestureOverlayHandler(func(event GestureOverlayEvent) {
		if overlay != nil {
			overlay.Update(event)
		}
	})

	manager.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		manager.Hide()
		event.Cancel()
	})

	tray := app.SystemTray.New()
	tray.SetIcon(appIcon)
	tray.SetTooltip("Key Flow")
	service.onConfigChanged = func(config AppConfig) {
		tray.SetMenu(buildTrayMenu(app, manager, service, config))
	}
	tray.SetMenu(buildTrayMenu(app, manager, service, store.Snapshot()))
	tray.OnClick(func() {
		manager.Show().Focus()
	})

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		overlay = createNativeGestureOverlay()
		if err := engine.Start(); err != nil {
			log.Printf("启动全局输入引擎失败: %v", err)
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
