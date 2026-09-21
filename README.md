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
go test ./...
```

构建：

```powershell
wails3 build
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
