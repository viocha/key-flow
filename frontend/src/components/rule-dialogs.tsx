import { useEffect, useState } from "react"
import type {
  ActionPreset,
  HotkeyRule,
  LayerBinding,
} from "../../bindings/key-flow"
import { PlusIcon } from "lucide-react"

import { ActionEditor } from "@/components/action-editor"
import { KeyRecorder } from "@/components/key-recorder"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { NumberInput } from "@/components/ui/number-input"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { WheelStepsInput } from "@/components/wheel-setting-inputs"
import {
  defaultBinding,
  defaultHotkey,
  defaultLayer,
  mouseInputLabel,
  normalizeMouseModifiers,
  type UiLayer,
} from "@/lib/app-model"

const shortcutInputTypes = {
  keyboard: "键盘",
  mouse: "鼠标",
}

const shortcutMouseInputs = [
  "left",
  "right",
  "middle",
  "back",
  "forward",
  "wheelUp",
  "wheelDown",
] as const

const shortcutMouseModifiers = ["Ctrl", "Shift", "Alt", "Win"] as const

export function ShortcutDialog({
  open,
  rule,
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  rule?: HotkeyRule
  presets: ActionPreset[]
  onClose: () => void
  onSave: (rule: HotkeyRule) => void
}) {
  const [draft, setDraft] = useState<HotkeyRule>(defaultHotkey())
  const inputType = draft.mouseInput ? "mouse" : "keyboard"
  const mouseInputItems = Object.fromEntries(
    shortcutMouseInputs.map((input) => [input, mouseInputLabel(input)]),
  )

  useEffect(() => {
    if (open) {
      setDraft(rule ? structuredClone(rule) : defaultHotkey())
    }
  }, [open, rule])

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="h-[calc(100vh-3rem)] max-h-[42rem] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{rule ? "编辑快捷键" : "添加快捷键"}</DialogTitle>
        </DialogHeader>
        <ScrollArea className="min-h-0">
          <div className="p-1">
            <FieldGroup>
            <Field>
              <FieldLabel>输入类型</FieldLabel>
              <ToggleGroup
                variant="outline"
                spacing={0}
                value={[inputType]}
                onValueChange={(values) => {
                  const value = values[0]
                  if (!value) return
                  if (value === "mouse") {
                    setDraft({
                      ...draft,
                      keys: [],
                      mouseInput: "wheelUp",
                      wheelSteps: 1,
                      modifiers: ["Alt"],
                    })
                    return
                  }
                  setDraft({
                    ...draft,
                    keys: draft.keys?.length ? draft.keys : ["Ctrl", "Alt", "K"],
                    mouseInput: "",
                    wheelSteps: undefined,
                    modifiers: [],
                  })
                }}
              >
                {Object.entries(shortcutInputTypes).map(([value, label]) => (
                  <ToggleGroupItem key={value} value={value}>
                    {label}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
            </Field>
            {inputType === "keyboard" ? (
              <Field>
                <FieldLabel>快捷键</FieldLabel>
                <KeyRecorder
                  value={draft.keys}
                  onChange={(keys) => setDraft({ ...draft, keys })}
                />
              </Field>
            ) : (
              <>
                <Field>
                  <FieldLabel>修饰键</FieldLabel>
                  <ToggleGroup
                    multiple
                    variant="outline"
                    className="w-full"
                    value={normalizeMouseModifiers(draft.modifiers)}
                    onValueChange={(modifiers) =>
                      setDraft({
                        ...draft,
                        modifiers: normalizeMouseModifiers(modifiers),
                      })
                    }
                  >
                    {shortcutMouseModifiers.map((modifier) => (
                      <ToggleGroupItem
                        key={modifier}
                        value={modifier}
                        className="flex-1"
                      >
                        {modifier}
                      </ToggleGroupItem>
                    ))}
                  </ToggleGroup>
                </Field>
                <Field>
                  <FieldLabel>鼠标输入</FieldLabel>
                  <Select
                    items={mouseInputItems}
                    value={draft.mouseInput ?? "wheelUp"}
                    onValueChange={(mouseInput) => {
                      if (!mouseInput) return
                      setDraft({
                        ...draft,
                        mouseInput,
                        wheelSteps: mouseInput.startsWith("wheel")
                          ? draft.wheelSteps ?? 1
                          : undefined,
                      })
                    }}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {Object.entries(mouseInputItems).map(([value, label]) => (
                          <SelectItem key={value} value={value}>
                            {label}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </Field>
                {(draft.mouseInput === "wheelUp" ||
                  draft.mouseInput === "wheelDown") && (
                  <WheelStepsInput
                    value={draft.wheelSteps!}
                    onValueChange={(wheelSteps) =>
                      setDraft({ ...draft, wheelSteps })
                    }
                  />
                )}
              </>
            )}
              <ActionEditor
                action={draft.action}
                presets={presets}
                onChange={(action) => setDraft({ ...draft, action })}
              />
            </FieldGroup>
          </div>
        </ScrollArea>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={
              inputType === "keyboard"
                ? !(draft.keys?.length)
                : !draft.mouseInput || !normalizeMouseModifiers(draft.modifiers).length
            }
            onClick={() => {
              onSave(draft)
              onClose()
            }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

const triggerItems = {
  CapsLock: "CapsLock",
  MouseRight: "鼠标右键",
  MouseMiddle: "鼠标中键",
  MouseBack: "鼠标后退键",
  MouseForward: "鼠标前进键",
}

const tapItems = {
  original: "执行原始操作",
  none: "不执行任何操作",
}

export function LayerDialog({
  open,
  layer,
  defaultHoldMs,
  onClose,
  onSave,
}: {
  open: boolean
  layer?: UiLayer
  defaultHoldMs: number
  onClose: () => void
  onSave: (layer: UiLayer) => void
}) {
  const [draft, setDraft] = useState<UiLayer>(() => defaultLayer(defaultHoldMs))

  useEffect(() => {
    if (open) {
      setDraft(layer ? structuredClone(layer) : defaultLayer(defaultHoldMs))
    }
  }, [defaultHoldMs, layer, open])

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{layer ? "编辑触发键" : "添加触发键"}</DialogTitle>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel>触发键</FieldLabel>
            <Select
              items={triggerItems}
              value={draft.trigger}
              onValueChange={(value) => {
                const trigger = value ?? "CapsLock"
                setDraft({
                  ...draft,
                  trigger,
                  name: triggerItems[trigger as keyof typeof triggerItems] ?? trigger,
                })
              }}
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {Object.entries(triggerItems).map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel htmlFor="layer-hold">进入 Layer 的长按时间</FieldLabel>
            <NumberInput
              id="layer-hold"
              min={50}
              max={1500}
              value={draft.holdMs}
              onValueChange={(holdMs) =>
                setDraft({
                  ...draft,
                  holdMs,
                })
              }
            />
            <FieldDescription>50–1500 ms</FieldDescription>
          </Field>
          <Field>
            <FieldLabel>短按行为</FieldLabel>
            <Select
              items={tapItems}
              value={draft.tapBehavior}
              onValueChange={(value) =>
                setDraft({ ...draft, tapBehavior: value ?? "original" })
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value="original">执行原始操作</SelectItem>
                  <SelectItem value="none">不执行任何操作</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
        </FieldGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={draft.holdMs < 50 || draft.holdMs > 1500}
            onClick={() => {
              onSave({
                ...draft,
                name:
                  triggerItems[draft.trigger as keyof typeof triggerItems] ??
                  draft.trigger,
              })
              onClose()
            }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function BindingDialog({
  open,
  binding,
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  binding?: LayerBinding
  presets: ActionPreset[]
  onClose: () => void
  onSave: (binding: LayerBinding) => void
}) {
  const [draft, setDraft] = useState<LayerBinding>(defaultBinding())

  useEffect(() => {
    if (open) {
      setDraft(binding ? structuredClone(binding) : defaultBinding())
    }
  }, [binding, open])

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="h-[calc(100vh-3rem)] max-h-[42rem] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{binding ? "编辑 Layer 映射" : "添加 Layer 映射"}</DialogTitle>
        </DialogHeader>
        <ScrollArea className="min-h-0">
          <div className="p-1">
            <FieldGroup>
            <Field>
              <FieldLabel>Layer 内按键</FieldLabel>
              <KeyRecorder
                value={draft.keys}
                onChange={(keys) => setDraft({ ...draft, keys })}
              />
            </Field>
              <ActionEditor
                action={draft.action}
                presets={presets}
                onChange={(action) => setDraft({ ...draft, action })}
              />
            </FieldGroup>
          </div>
        </ScrollArea>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={!(draft.keys?.length)}
            onClick={() => {
              onSave(draft)
              onClose()
            }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function AddButton({
  children,
  onClick,
}: {
  children: React.ReactNode
  onClick: () => void
}) {
  return (
    <Button type="button" size="sm" onClick={onClick}>
      <PlusIcon data-icon="inline-start" />
      {children}
    </Button>
  )
}
