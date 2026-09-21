package main

import "github.com/wailsapp/wails/v3/pkg/application"

func buildTrayMenu(
	app *application.App,
	openManager func(),
	service *AppService,
	config AppConfig,
) *application.Menu {
	menu := app.NewMenu()
	menu.Add("打开窗口").OnClick(func(*application.Context) {
		openManager()
	})
	menu.AddSeparator()

	groups := menu.AddSubmenu("分组")
	if profile := activeProfileForTray(&config); profile != nil {
		profileID := profile.ID
		addTrayConfigToggle(groups, "通用快捷键", profile.HotkeysEnabled, service, func(next *AppConfig, enabled bool) bool {
			current := profileByIDForTray(next, profileID)
			if current == nil {
				return false
			}
			current.HotkeysEnabled = enabled
			return true
		})

		for _, layer := range profile.Layers {
			layerID := layer.ID
			addTrayConfigToggle(groups, "触发键 · "+layer.Name, layer.Enabled, service, func(next *AppConfig, enabled bool) bool {
				current := profileByIDForTray(next, profileID)
				if current == nil {
					return false
				}
				for i := range current.Layers {
					if current.Layers[i].ID == layerID {
						current.Layers[i].Enabled = enabled
						return true
					}
				}
				return false
			})
		}

		for _, gesture := range profile.Gestures {
			gestureID := gesture.ID
			label := "手势 · " + triggerLabelForTray(gesture.Trigger)
			addTrayConfigToggle(groups, label, gesture.Enabled, service, func(next *AppConfig, enabled bool) bool {
				current := profileByIDForTray(next, profileID)
				if current == nil {
					return false
				}
				for i := range current.Gestures {
					if current.Gestures[i].ID == gestureID {
						current.Gestures[i].Enabled = enabled
						return true
					}
				}
				return false
			})
		}

		addTrayConfigToggle(groups, "屏幕边缘", profile.EdgeWheelsEnabled, service, func(next *AppConfig, enabled bool) bool {
			current := profileByIDForTray(next, profileID)
			if current == nil {
				return false
			}
			current.EdgeWheelsEnabled = enabled
			return true
		})
	}

	addTrayConfigToggle(menu, "启用 Key Flow", config.Enabled, service, func(next *AppConfig, enabled bool) bool {
		next.Enabled = enabled
		return true
	})

	menu.AddSeparator()
	menu.Add("退出").OnClick(func(*application.Context) {
		app.Quit()
	})
	return menu
}

func addTrayConfigToggle(
	menu *application.Menu,
	label string,
	checked bool,
	service *AppService,
	mutate func(*AppConfig, bool) bool,
) {
	var item *application.MenuItem
	item = menu.AddCheckbox(label, checked).OnClick(func(ctx *application.Context) {
		next := service.GetConfig()
		if !mutate(&next, ctx.IsChecked()) {
			item.SetChecked(!ctx.IsChecked())
			return
		}
		if _, err := service.SaveConfig(next); err != nil {
			item.SetChecked(!ctx.IsChecked())
		}
	})
}

func activeProfileForTray(config *AppConfig) *Profile {
	return profileByIDForTray(config, config.ActiveProfileID)
}

func profileByIDForTray(config *AppConfig, profileID string) *Profile {
	for i := range config.Profiles {
		if config.Profiles[i].ID == profileID {
			return &config.Profiles[i]
		}
	}
	return nil
}

func triggerLabelForTray(trigger string) string {
	switch trigger {
	case "MouseRight":
		return "鼠标右键"
	case "MouseMiddle":
		return "鼠标中键"
	case "MouseBack":
		return "鼠标后退键"
	case "MouseForward":
		return "鼠标前进键"
	default:
		return trigger
	}
}
