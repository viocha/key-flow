# Key Flow

Key Flow 是一个 Windows 全局快捷键与触发键工具，使用 Go + Wails v3、React、shadcn/ui 和 Bun 构建。

## 当前能力

- 普通全局快捷键
- 快捷键重映射
- CapsLock、鼠标右键、Mouse4、Mouse5 触发键 Layer
- Trigger + Ctrl / Shift / Alt / Win 组合映射
- Trigger 短按保留原操作，长按进入 Layer
- 按场景分组的预置动作
- 输入文本、打开文件/目录/URL
- 执行 exe、PowerShell、Bun 等命令或脚本
- 动作序列后端执行能力
- Profile 配置模型
- JSON 自动保存
- 系统托盘与全局暂停开关

## 窗口切换

“切换上一个窗口”和“切换下一个窗口”按窗口激活历史后退、前进。例如从 A 点击到 F 后，上一个返回 A，下一个回到 F。到达历史边界时保持当前窗口；后退后手动选择其他窗口会清除前进历史。已关闭或不可切换的窗口会被跳过。

历史从应用启动时开始记录，最多保存 128 次访问，仅保存在内存中。

## 鼠标手势

按住手势分组的启动键移动鼠标，松开后执行匹配的动作。录制和实际触发使用分组的“分段距离”：直线附近的小幅抖动会被过滤，水平、垂直方向允许自然斜偏，45° 斜线可以录制为对角方向。横竖方向的净位移达到分段距离后计入方向，斜向需要达到分段距离的 1.4 倍；例如分段距离为 28 px 时，斜向约需 40 px 才确认。

距离以屏幕像素计算，录制界面按屏幕缩放换算采样距离和防抖容差，实时方向、终点箭头与松开后的识别结果使用相同单位。

## 开发

要求：

- Windows
- Go
- Wails v3 CLI
- Bun

开发模式：

```powershell
wails3 dev
```

运行测试：

```powershell
task test
```

构建：

```powershell
task build
```

产物：

```text
bin/key-flow.exe
```

## 配置

配置保存在当前用户配置目录下的：

```text
key-flow/config.json
```

配置采用版本化 JSON。应用内修改会自动保存，并同步更新正在运行的输入引擎。

## Windows 输入层

输入监听基于：

- `WH_KEYBOARD_LL`
- `WH_MOUSE_LL`
- `SendInput`

Hook 回调只负责匹配和拦截，动作在独立队列执行。Key Flow 自己注入的输入带有专用标记，不会再次触发规则。

普通权限进程向更高完整性级别的管理员进程注入输入会受 Windows UIPI 限制。
