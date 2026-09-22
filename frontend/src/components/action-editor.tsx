import { useEffect, useId, useMemo, useRef, useState } from "react"
import { ActionType, type Action, type ActionPreset } from "../../bindings/key-flow"
import { PlusIcon, Trash2Icon } from "lucide-react"

import { KeyRecorder } from "@/components/key-recorder"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { cloneAction, defaultAction } from "@/lib/app-model"
import { showUndoDeleteToast } from "@/lib/undo-toast"

const actionTypeItems = {
  keyStroke: "模拟快捷键",
  text: "输入文本",
  launch: "打开目标",
  command: "运行命令 / 脚本",
  window: "窗口操作",
  volume: "音量",
  brightness: "亮度",
  sequence: "动作序列",
}

const windowCommandItems = {
  close: "关闭窗口",
  terminateProcess: "终止进程",
  previousWindow: "切换上一个窗口",
  nextWindow: "切换下一个窗口",
  minimize: "最小化窗口",
  restoreMinimized: "恢复最小化窗口",
  maximize: "最大化窗口",
  restore: "恢复窗口",
  toggleMaximize: "切换最大化",
  alwaysOnTop: "切换置顶",
}

const commandArgsModeItems = {
  commandLine: "命令行",
  list: "参数列表",
}

export function ActionEditor({
  action,
  presets,
  onChange,
  depth = 0,
}: {
  action: Action
  presets: ActionPreset[]
  onChange: (action: Action) => void
  depth?: number
}) {
  const fieldID = useId()
  const actionRef = useRef(action)
  actionRef.current = action
  const groups = useMemo(
    () =>
      presets.reduce<Record<string, ActionPreset[]>>((result, preset) => {
        ;(result[preset.group] ??= []).push(preset)
        return result
      }, {}),
    [presets],
  )
  const groupNames = useMemo(() => Object.keys(groups), [groups])
  const matchingPreset = presets.find(
    (preset) => preset.name === action.name && preset.action.type === action.type,
  )
  const [presetGroup, setPresetGroup] = useState(
    matchingPreset?.group ?? groupNames[0] ?? "",
  )
  const [editorMode, setEditorMode] = useState<"preset" | "custom">(
    matchingPreset ? "preset" : "custom",
  )
  const [commandArgsText, setCommandArgsText] = useState(() =>
    (action.args ?? []).join("\n"),
  )

  useEffect(() => {
    if (matchingPreset?.group) {
      setPresetGroup(matchingPreset.group)
    }
  }, [matchingPreset?.group])

  useEffect(() => {
    setPresetGroup((current) =>
      groups[current] ? current : (groupNames[0] ?? ""),
    )
  }, [groupNames, groups])

  useEffect(() => {
    const nextArgs =
      action.type === ActionType.ActionCommand ? (action.args ?? []) : []
    setCommandArgsText((current) =>
      sameStringArray(parseCommandArgs(current), nextArgs)
        ? current
        : nextArgs.join("\n"),
    )
  }, [action.args, action.type])

  const groupItems = Object.fromEntries(groupNames.map((group) => [group, group]))
  const groupPresets = groups[presetGroup] ?? []
  const presetItems = Object.fromEntries(
    groupPresets.map((preset) => [preset.id, preset.name]),
  )
  const selectedPreset =
    matchingPreset?.group === presetGroup ? matchingPreset.id : null

  const setType = (type: ActionType) => {
    switch (type) {
      case ActionType.ActionText:
        onChange({ type, text: "" })
        break
      case ActionType.ActionLaunch:
        onChange({ type, target: "" })
        break
      case ActionType.ActionCommand:
        onChange({
          type,
          program: "",
          argsMode: "commandLine",
          commandLine: "",
          args: [],
        })
        break
      case ActionType.ActionWindow:
        onChange({ type, windowCommand: "close" })
        break
      case ActionType.ActionVolume:
        onChange({ type, delta: 1 })
        break
      case ActionType.ActionBrightness:
        onChange({ type, delta: 5 })
        break
      case ActionType.ActionSequence:
        onChange({ type, sequence: [defaultAction()] })
        break
      default:
        onChange({ type: ActionType.ActionKeyStroke, keys: ["Ctrl", "C"] })
    }
  }

  return (
    <FieldGroup>
      <Field>
        <FieldLabel>动作</FieldLabel>
        <Tabs
          value={editorMode}
          onValueChange={(value) =>
            setEditorMode(value as "preset" | "custom")
          }
        >
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="preset">预置</TabsTrigger>
            <TabsTrigger value="custom">自定义</TabsTrigger>
          </TabsList>

          <TabsContent value="preset" className="pt-2">
            <div className="grid grid-cols-[9rem_minmax(0,1fr)] gap-2">
              <Select
                items={groupItems}
                value={presetGroup}
                onValueChange={(value) => value && setPresetGroup(value)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="分类" />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {groupNames.map((group) => (
                      <SelectItem key={group} value={group}>
                        {group}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>

              <Select
                items={presetItems}
                value={selectedPreset}
                onValueChange={(value) => {
                  const preset = groupPresets.find((item) => item.id === value)
                  if (preset) {
                    onChange({ ...cloneAction(preset.action), name: preset.name })
                  }
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择动作" />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {groupPresets.map((preset) => (
                      <SelectItem key={preset.id} value={preset.id}>
                        {preset.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          </TabsContent>

          <TabsContent value="custom" className="pt-2">
            <Select
              items={actionTypeItems}
              value={action.type}
              onValueChange={(value) => setType(value as ActionType)}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value={ActionType.ActionKeyStroke}>
                    模拟快捷键
                  </SelectItem>
                  <SelectItem value={ActionType.ActionText}>输入文本</SelectItem>
                  <SelectItem value={ActionType.ActionLaunch}>打开目标</SelectItem>
                  <SelectItem value={ActionType.ActionCommand}>
                    运行命令 / 脚本
                  </SelectItem>
                  <SelectItem value={ActionType.ActionWindow}>窗口操作</SelectItem>
                  <SelectItem value={ActionType.ActionVolume}>音量</SelectItem>
                  <SelectItem value={ActionType.ActionBrightness}>亮度</SelectItem>
                  {depth < 3 && (
                    <SelectItem value={ActionType.ActionSequence}>
                      动作序列
                    </SelectItem>
                  )}
                </SelectGroup>
              </SelectContent>
            </Select>
          </TabsContent>
        </Tabs>
      </Field>

      {editorMode === "custom" &&
        action.type === ActionType.ActionKeyStroke && (
        <Field>
          <FieldLabel>目标按键</FieldLabel>
          <KeyRecorder
            value={action.keys}
            onChange={(keys) => onChange({ ...action, keys })}
          />
        </Field>
      )}

      {editorMode === "custom" && action.type === ActionType.ActionText && (
        <Field>
          <FieldLabel htmlFor={fieldID + "-text"}>文本</FieldLabel>
          <Textarea
            id={fieldID + "-text"}
            className="min-h-24 resize-none"
            value={action.text ?? ""}
            onChange={(event) => onChange({ ...action, text: event.target.value })}
            placeholder="触发后输入这段文本"
          />
        </Field>
      )}

      {editorMode === "custom" &&
        action.type === ActionType.ActionLaunch && (
        <Field>
          <FieldLabel htmlFor={fieldID + "-target"}>目标</FieldLabel>
          <Input
            id={fieldID + "-target"}
            value={action.target ?? ""}
            onChange={(event) => onChange({ ...action, target: event.target.value })}
            placeholder="https://、文件、目录或程序路径"
          />
        </Field>
      )}

      {editorMode === "custom" &&
        action.type === ActionType.ActionCommand && (
        <>
          <Field>
            <FieldLabel htmlFor={fieldID + "-program"}>程序</FieldLabel>
            <Input
              id={fieldID + "-program"}
              value={action.program ?? ""}
              onChange={(event) => onChange({ ...action, program: event.target.value })}
              placeholder="powershell.exe、pwsh.exe、bun.exe…"
            />
          </Field>
          <Field>
            <FieldLabel>参数输入方式</FieldLabel>
            <Select
              items={commandArgsModeItems}
              value={action.argsMode ?? "commandLine"}
              onValueChange={(value) => {
                if (!value || value === action.argsMode) return
                if (value === "commandLine") {
                  const args = action.args ?? []
                  onChange({
                    ...action,
                    argsMode: value,
                    commandLine:
                      action.commandLine || formatWindowsArgs(args),
                  })
                  return
                }
                const args = parseWindowsArgs(action.commandLine ?? "")
                setCommandArgsText(args.join("\n"))
                onChange({
                  ...action,
                  argsMode: value,
                  args,
                })
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value="commandLine">命令行</SelectItem>
                  <SelectItem value="list">参数列表</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>

          {(action.argsMode ?? "commandLine") === "commandLine" ? (
            <Field>
              <FieldLabel htmlFor={fieldID + "-command-line"}>参数</FieldLabel>
              <Textarea
                id={fieldID + "-command-line"}
                className="min-h-24 resize-y overflow-auto font-mono text-xs [field-sizing:fixed]"
                value={action.commandLine ?? ""}
                onChange={(event) =>
                  onChange({ ...action, commandLine: event.target.value })
                }
                placeholder={'-File "D:\\My Scripts\\build.ps1" -Mode release'}
              />
              <FieldDescription>
                按 Windows 命令行规则解析；可用双引号包裹含空格参数
              </FieldDescription>
            </Field>
          ) : (
            <Field>
              <FieldLabel htmlFor={fieldID + "-args"}>参数</FieldLabel>
              <Textarea
                id={fieldID + "-args"}
                className="min-h-24 resize-y overflow-auto font-mono text-xs [field-sizing:fixed]"
                value={commandArgsText}
                onChange={(event) => {
                  const value = event.target.value
                  setCommandArgsText(value)
                  onChange({
                    ...action,
                    args: parseCommandArgs(value),
                  })
                }}
                placeholder={"每行一个参数\n-File\nD:\\scripts\\build.ps1"}
              />
              <FieldDescription>
                每行作为一个独立参数；引号会作为参数内容保留
              </FieldDescription>
            </Field>
          )}
          <Field>
            <FieldLabel htmlFor={fieldID + "-working-directory"}>工作目录</FieldLabel>
            <Input
              id={fieldID + "-working-directory"}
              value={action.workingDirectory ?? ""}
              onChange={(event) =>
                onChange({ ...action, workingDirectory: event.target.value })
              }
              placeholder="可选"
            />
          </Field>
          <Field>
            <FieldLabel>可用变量</FieldLabel>
            <FieldDescription className="flex flex-wrap gap-x-3 gap-y-1 font-mono text-xs">
              <span>{"${window.hwnd}"}</span>
              <span>{"${window.title}"}</span>
              <span>{"${window.x}"}</span>
              <span>{"${window.y}"}</span>
              <span>{"${window.width}"}</span>
              <span>{"${window.height}"}</span>
              <span>{"${process.pid}"}</span>
              <span>{"${process.name}"}</span>
              <span>{"${process.path}"}</span>
              <span>{"${process.dir}"}</span>
              <span>{"${selection.text}"}</span>
              <span>{"${clipboard.text}"}</span>
              <span>{"${cursor.x}"}</span>
              <span>{"${cursor.y}"}</span>
              <span>{"${time.date}"}</span>
              <span>{"${time.time}"}</span>
              <span>{"${time.datetime}"}</span>
              <span>{"${time.unix}"}</span>
              <span>{"${time.unixMs}"}</span>
              <span>{"${env.NAME}"}</span>
            </FieldDescription>
          </Field>
        </>
      )}

      {editorMode === "custom" &&
        action.type === ActionType.ActionWindow && (
        <Field>
          <FieldLabel>窗口操作</FieldLabel>
          <Select
            items={windowCommandItems}
            value={action.windowCommand ?? "close"}
            onValueChange={(value) => {
              if (!value) return
              onChange({
                ...action,
                windowCommand: value,
                ...(value === "alwaysOnTop"
                  ? {
                      windowBorderColor:
                        action.windowBorderColor || "#34A9DE",
                      windowBorderSize: action.windowBorderSize || 2,
                    }
                  : {}),
              })
            }}
          >
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                {Object.entries(windowCommandItems).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>
      )}

      {action.type === ActionType.ActionWindow &&
        action.windowCommand === "alwaysOnTop" && (
          <div className="grid grid-cols-[minmax(0,1fr)_8rem] gap-3">
            <Field>
              <FieldLabel htmlFor={fieldID + "-window-border-color"}>
                置顶边框颜色
              </FieldLabel>
              <Input
                id={fieldID + "-window-border-color"}
                type="color"
                value={action.windowBorderColor ?? "#34A9DE"}
                onChange={(event) =>
                  onChange({
                    ...action,
                    windowBorderColor: event.target.value.toUpperCase(),
                  })
                }
                className="h-9 cursor-pointer p-1"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor={fieldID + "-window-border-size"}>
                边框尺寸
              </FieldLabel>
              <Input
                id={fieldID + "-window-border-size"}
                type="number"
                min={1}
                max={12}
                value={action.windowBorderSize ?? 2}
                onChange={(event) =>
                  onChange({
                    ...action,
                    windowBorderSize: Math.max(
                      1,
                      Math.min(12, Number(event.target.value) || 2),
                    ),
                  })
                }
              />
            </Field>
          </div>
        )}

      {editorMode === "custom" &&
        action.type === ActionType.ActionVolume && (
        <Field>
          <FieldLabel htmlFor={fieldID + "-volume"}>变化步数</FieldLabel>
          <Input
            id={fieldID + "-volume"}
            type="number"
            min={-20}
            max={20}
            value={action.delta ?? 1}
            onChange={(event) =>
              onChange({
                ...action,
                delta: Math.max(-20, Math.min(20, Number(event.target.value) || 0)),
              })
            }
          />
          <FieldDescription>正数增加，负数降低；1 步使用系统音量键的一次增量</FieldDescription>
        </Field>
      )}

      {editorMode === "custom" &&
        action.type === ActionType.ActionBrightness && (
        <Field>
          <FieldLabel htmlFor={fieldID + "-brightness"}>亮度变化</FieldLabel>
          <Input
            id={fieldID + "-brightness"}
            type="number"
            min={-100}
            max={100}
            value={action.delta ?? 5}
            onChange={(event) =>
              onChange({
                ...action,
                delta: Math.max(-100, Math.min(100, Number(event.target.value) || 0)),
              })
            }
          />
          <FieldDescription>百分比；正数增加，负数降低</FieldDescription>
        </Field>
      )}

      {editorMode === "custom" &&
        action.type === ActionType.ActionSequence && (
        <Field>
          <FieldLabel>动作序列</FieldLabel>
          <div className="flex flex-col gap-2">
            {(action.sequence ?? []).map((child, index) => (
              <Card key={index} size="sm">
                <CardHeader>
                  <CardTitle className="text-sm">动作 {index + 1}</CardTitle>
                  <CardAction>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label={"删除动作 " + (index + 1)}
                      onClick={() => {
                        const sequence = action.sequence ?? []
                        const deleted = sequence[index]
                        if (!deleted) return
                        onChange({
                          ...action,
                          sequence: sequence.filter((_, childIndex) => childIndex !== index),
                        })
                        showUndoDeleteToast("动作", () => {
                          const current = actionRef.current
                          const restored = [...(current.sequence ?? [])]
                          restored.splice(Math.min(index, restored.length), 0, deleted)
                          onChange({ ...current, sequence: restored })
                        })
                      }}
                    >
                      <Trash2Icon />
                    </Button>
                  </CardAction>
                </CardHeader>
                <CardContent>
                  <ActionEditor
                    action={child}
                    presets={presets}
                    depth={depth + 1}
                    onChange={(next) =>
                      onChange({
                        ...action,
                        sequence: (action.sequence ?? []).map((item, childIndex) =>
                          childIndex === index ? next : item,
                        ),
                      })
                    }
                  />
                </CardContent>
              </Card>
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="self-start"
              onClick={() =>
                onChange({
                  ...action,
                  sequence: [...(action.sequence ?? []), defaultAction()],
                })
              }
            >
              <PlusIcon data-icon="inline-start" />
              添加动作
            </Button>
          </div>
          <FieldDescription>动作按从上到下的顺序执行</FieldDescription>
        </Field>
      )}

      {depth > 0 && (
        <Field>
          <FieldLabel htmlFor={fieldID + "-delay"}>执行前延迟</FieldLabel>
          <Input
            id={fieldID + "-delay"}
            type="number"
            min={0}
            max={60000}
            value={action.delayMs ?? 0}
            onChange={(event) =>
              onChange({
                ...action,
                delayMs: Math.max(0, Math.min(60000, Number(event.target.value) || 0)),
              })
            }
          />
          <FieldDescription>毫秒，用于动作序列中的节奏控制</FieldDescription>
        </Field>
      )}
    </FieldGroup>
  )
}

function parseCommandArgs(value: string) {
  return value
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean)
}

function sameStringArray(left: string[], right: string[]) {
  return (
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  )
}

function parseWindowsArgs(value: string) {
  const result: string[] = []
  let index = 0

  while (index < value.length) {
    while (/\s/.test(value[index] ?? "")) index += 1
    if (index >= value.length) break

    let current = ""
    let quoted = false
    while (index < value.length) {
      const char = value[index]
      if (!quoted && /\s/.test(char)) break

      if (char === "\\") {
        let slashCount = 0
        while (value[index + slashCount] === "\\") slashCount += 1
        if (value[index + slashCount] === '"') {
          current += "\\".repeat(Math.floor(slashCount / 2))
          if (slashCount % 2 === 0) {
            quoted = !quoted
          } else {
            current += '"'
          }
          index += slashCount + 1
          continue
        }
        current += "\\".repeat(slashCount)
        index += slashCount
        continue
      }

      if (char === '"') {
        quoted = !quoted
        index += 1
        continue
      }

      current += char
      index += 1
    }
    result.push(current)
    while (/\s/.test(value[index] ?? "")) index += 1
  }

  return result
}

function formatWindowsArgs(args: string[]) {
  return args.map(quoteWindowsArg).join(" ")
}

function quoteWindowsArg(value: string) {
  if (value !== "" && !/[\s"]/.test(value)) return value

  let result = '"'
  let slashCount = 0
  for (const char of value) {
    if (char === "\\") {
      slashCount += 1
      continue
    }
    if (char === '"') {
      result += "\\".repeat(slashCount * 2 + 1) + '"'
      slashCount = 0
      continue
    }
    result += "\\".repeat(slashCount) + char
    slashCount = 0
  }
  result += "\\".repeat(slashCount * 2) + '"'
  return result
}
