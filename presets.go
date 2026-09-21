package main

func actionPresets() []ActionPreset {
	return []ActionPreset{
		{ID: "nav-left", Group: "导航", Name: "向左", Action: Action{Type: ActionKeyStroke, Keys: []string{"ArrowLeft"}}},
		{ID: "nav-down", Group: "导航", Name: "向下", Action: Action{Type: ActionKeyStroke, Keys: []string{"ArrowDown"}}},
		{ID: "nav-up", Group: "导航", Name: "向上", Action: Action{Type: ActionKeyStroke, Keys: []string{"ArrowUp"}}},
		{ID: "nav-right", Group: "导航", Name: "向右", Action: Action{Type: ActionKeyStroke, Keys: []string{"ArrowRight"}}},
		{ID: "nav-word-left", Group: "导航", Name: "按词向左", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "ArrowLeft"}}},
		{ID: "nav-word-right", Group: "导航", Name: "按词向右", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "ArrowRight"}}},

		{ID: "edit-copy", Group: "编辑", Name: "复制", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "C"}}},
		{ID: "edit-paste", Group: "编辑", Name: "粘贴", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "V"}}},
		{ID: "edit-cut", Group: "编辑", Name: "剪切", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "X"}}},
		{ID: "edit-undo", Group: "编辑", Name: "撤销", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Z"}}},
		{ID: "edit-redo", Group: "编辑", Name: "重做", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Shift", "Z"}}},

		{ID: "browser-new-tab", Group: "浏览器", Name: "新建标签页", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "T"}}},
		{ID: "browser-close-tab", Group: "浏览器", Name: "关闭标签页", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "W"}}},
		{ID: "browser-reopen-tab", Group: "浏览器", Name: "恢复标签页", Action: Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Shift", "T"}}},

		{ID: "window-close", Group: "窗口", Name: "关闭窗口", Action: Action{Type: ActionWindow, WindowCommand: "close"}},
		{ID: "window-terminate-process", Group: "窗口", Name: "终止进程", Action: Action{Type: ActionWindow, WindowCommand: "terminateProcess"}},
		{ID: "window-minimize", Group: "窗口", Name: "最小化窗口", Action: Action{Type: ActionWindow, WindowCommand: "minimize"}},
		{ID: "window-maximize", Group: "窗口", Name: "最大化窗口", Action: Action{Type: ActionWindow, WindowCommand: "maximize"}},
		{ID: "window-toggle-maximize", Group: "窗口", Name: "切换最大化", Action: Action{Type: ActionWindow, WindowCommand: "toggleMaximize"}},
		{ID: "window-topmost", Group: "窗口", Name: "切换置顶", Action: Action{Type: ActionWindow, WindowCommand: "alwaysOnTop", WindowBorderColor: "#34A9DE", WindowBorderSize: 2}},

		{ID: "media-play", Group: "媒体", Name: "播放 / 暂停", Action: Action{Type: ActionKeyStroke, Keys: []string{"MediaPlayPause"}}},
		{ID: "media-next", Group: "媒体", Name: "下一首", Action: Action{Type: ActionKeyStroke, Keys: []string{"MediaNext"}}},
		{ID: "media-prev", Group: "媒体", Name: "上一首", Action: Action{Type: ActionKeyStroke, Keys: []string{"MediaPrev"}}},
		{ID: "media-volume-up", Group: "媒体", Name: "音量增加", Action: Action{Type: ActionVolume, Delta: 1}},
		{ID: "media-volume-down", Group: "媒体", Name: "音量降低", Action: Action{Type: ActionVolume, Delta: -1}},
		{ID: "media-mute", Group: "媒体", Name: "静音", Action: Action{Type: ActionKeyStroke, Keys: []string{"VolumeMute"}}},

		{ID: "display-brightness-up", Group: "显示器", Name: "亮度增加 5%", Action: Action{Type: ActionBrightness, Delta: 5}},
		{ID: "display-brightness-down", Group: "显示器", Name: "亮度降低 5%", Action: Action{Type: ActionBrightness, Delta: -5}},

		{ID: "dev-terminal", Group: "开发", Name: "Windows Terminal", Action: Action{Type: ActionCommand, Program: "wt.exe", ArgsMode: "commandLine"}},
	}
}
