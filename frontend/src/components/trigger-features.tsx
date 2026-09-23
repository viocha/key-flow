import { useEffect, useMemo, useRef, useState } from "react"
import type {
  ActionPreset,
  MouseShortcutBinding,
} from "../../bindings/key-flow"
import { PlusIcon } from "lucide-react"

import { ActionEditor } from "@/components/action-editor"
import { EditableListRow } from "@/components/editable-list-row"
import { ReorderHandle, useListReorder } from "@/components/list-reorder"
import { RuleContextMenu } from "@/components/rule-context-menu"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { NumberInput } from "@/components/ui/number-input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { WheelStepsInput } from "@/components/wheel-setting-inputs"
import {
  actionSummary,
  defaultMouseShortcutAction,
  mouseInputForTrigger,
  mouseInputLabel,
  mouseShortcutKey,
  mouseShortcutLabel,
  normalizeMouseModifiers,
  type UiLayer,
} from "@/lib/app-model"
import { showUndoDeleteToast } from "@/lib/undo-toast"

const mouseInputs = [
  "left",
  "right",
  "middle",
  "back",
  "forward",
  "wheelUp",
  "wheelDown",
] as const

const mouseButtonInputs = [
  "left",
  "right",
  "middle",
  "back",
  "forward",
] as const

type MouseInput = (typeof mouseInputs)[number]
const mouseModifiers = ["Ctrl", "Shift", "Alt", "Win"] as const

export function MouseBindingsPanel({
  layer,
  presets,
  onChange,
}: {
  layer: UiLayer
  presets: ActionPreset[]
  onChange: (layer: UiLayer) => void
}) {
  const [dialog, setDialog] = useState<
    | { open: false }
    | {
        open: true
        binding?: MouseShortcutBinding
        originalKey?: string
      }
  >({ open: false })
  const layerRef = useRef(layer)
  layerRef.current = layer
  const reorder = useListReorder(
    layer.mouseShortcuts,
    mouseShortcutKey,
    (mouseShortcuts) => onChange({ ...layer, mouseShortcuts }),
  )

  const triggerInput = mouseInputForTrigger(layer.trigger)
  const shortcutInputs = new Set(
    layer.mouseShortcuts.filter((binding) => binding.enabled).map((binding) => binding.input),
  )

  const conflictsWithWindowOperations = (binding: MouseShortcutBinding) =>
    layer.windowOperations.enabled &&
    (binding.input === layer.windowOperations.moveButton ||
      binding.input === layer.windowOperations.resizeButton ||
      (layer.windowOperations.scaleWheel &&
        (binding.input === "wheelUp" || binding.input === "wheelDown")))

  const availableShortcutInputs = (): MouseInput[] =>
    mouseInputs.filter((input) => {
      if (input === triggerInput) return false
      if (layer.windowOperations.enabled) {
        if (input === layer.windowOperations.moveButton) return false
        if (input === layer.windowOperations.resizeButton) return false
        if (
          layer.windowOperations.scaleWheel &&
          (input === "wheelUp" || input === "wheelDown")
        ) {
          return false
        }
      }
      return true
    })

  const saveShortcut = (
    binding: MouseShortcutBinding,
    originalKey?: string,
  ) => {
    const exists = layer.mouseShortcuts.some(
      (item) => mouseShortcutKey(item) === originalKey,
    )
    onChange({
      ...layer,
      mouseShortcuts: exists
        ? layer.mouseShortcuts.map((item) =>
            mouseShortcutKey(item) === originalKey ? binding : item,
          )
        : [...layer.mouseShortcuts, binding],
    })
  }

  const deleteShortcut = (key: string) => {
    const index = layer.mouseShortcuts.findIndex(
      (binding) => mouseShortcutKey(binding) === key,
    )
    if (index < 0) return
    const deleted = layer.mouseShortcuts[index]
    onChange({
      ...layer,
      mouseShortcuts: layer.mouseShortcuts.filter(
        (binding) => mouseShortcutKey(binding) !== key,
      ),
    })
    showUndoDeleteToast("鼠标快捷键", () => {
      const current = layerRef.current
      if (
        current.mouseShortcuts.some(
          (binding) => mouseShortcutKey(binding) === mouseShortcutKey(deleted),
        )
      ) return
      const mouseShortcuts = [...current.mouseShortcuts]
      mouseShortcuts.splice(Math.min(index, mouseShortcuts.length), 0, deleted)
      onChange({ ...current, mouseShortcuts })
    })
  }

  const toggleShortcut = (key: string) => {
    onChange({
      ...layer,
      mouseShortcuts: layer.mouseShortcuts.map((binding) =>
        mouseShortcutKey(binding) === key
          ? { ...binding, enabled: !binding.enabled }
          : binding,
      ),
    })
  }

  const windowButtonItems = (
    current: string | undefined,
    other: string | undefined,
  ): Record<string, string> =>
    Object.fromEntries([
      ["none", "未设置"],
      ...mouseButtonInputs
        .filter((input) => {
          if (input === current) return true
          if (input === triggerInput) return false
          if (input === other) return false
          return !shortcutInputs.has(input)
        })
        .map((input) => [input, mouseInputLabel(input)]),
    ])

  const wheelShortcutUsed =
    shortcutInputs.has("wheelUp") || shortcutInputs.has("wheelDown")

  const setWindowOperationsEnabled = (enabled: boolean) => {
    const next = { ...layer.windowOperations, enabled }
    if (enabled) {
      if (
        next.moveButton === triggerInput ||
        shortcutInputs.has(next.moveButton ?? "")
      ) {
        next.moveButton = ""
      }
      if (
        next.resizeButton === triggerInput ||
        shortcutInputs.has(next.resizeButton ?? "") ||
        next.resizeButton === next.moveButton
      ) {
        next.resizeButton = ""
      }
      if (wheelShortcutUsed) {
        next.scaleWheel = false
      }
    }
    onChange({ ...layer, windowOperations: next })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <div className="text-sm font-medium">鼠标快捷键</div>
        <Button
          type="button"
          size="sm"
          disabled={availableShortcutInputs().length === 0}
          onClick={() => setDialog({ open: true })}
        >
          <PlusIcon data-icon="inline-start" />
          添加快捷键
        </Button>
      </div>

      <Card className="gap-0 py-0">
        <CardContent className="px-0" {...reorder.listProps()}>
          {layer.mouseShortcuts.length === 0 ? (
            <Empty className="min-h-40">
              <EmptyHeader>
                <EmptyTitle>暂无鼠标快捷键</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            layer.mouseShortcuts.map((binding, index) => {
              const bindingKey = mouseShortcutKey(binding)
              return (
                <div key={bindingKey} {...reorder.itemProps(bindingKey)}>
                  {index > 0 && <Separator className={reorder.separatorClassName(index)} />}
                  <RuleContextMenu
                    enabled={binding.enabled}
                    toggleDisabled={
                      !binding.enabled && conflictsWithWindowOperations(binding)
                        ? "与窗口操作冲突"
                        : undefined
                    }
                    onToggle={() => toggleShortcut(bindingKey)}
                    onEdit={() => setDialog({ open: true, binding, originalKey: bindingKey })}
                    onDelete={() => deleteShortcut(bindingKey)}
                  >
                  <EditableListRow
                    onEdit={() =>
                      setDialog({
                        open: true,
                        binding,
                        originalKey: bindingKey,
                      })
                    }
                  >
                    {reorder.canReorder && <ReorderHandle props={reorder.handleProps(bindingKey)} />}
                    <div className="w-40 shrink-0 text-sm font-medium">
                      {mouseShortcutLabel(binding)}
                    </div>
                    <div className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                      {actionSummary(binding.action)}
                    </div>
                    {!binding.enabled && <Badge variant="secondary" className="ml-auto shrink-0">已禁用</Badge>}
                  </EditableListRow>
                  </RuleContextMenu>
                </div>
              )
            })
          )}
        </CardContent>
      </Card>

      <div className="flex items-center justify-between gap-3">
        <div className="text-sm font-medium">窗口操作</div>
        <Switch
          size="sm"
          checked={layer.windowOperations.enabled}
          onCheckedChange={setWindowOperationsEnabled}
          aria-label="启用窗口操作"
        />
      </div>

      {layer.windowOperations.enabled && (
      <Card className="gap-0 py-0">
        <CardContent className="px-0">
          <div className="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3 px-3 py-2">
            <div className="text-sm font-medium">拖动窗口</div>
            <Select
              items={windowButtonItems(
                layer.windowOperations.moveButton,
                layer.windowOperations.resizeButton,
              )}
              value={layer.windowOperations.moveButton || "none"}
              onValueChange={(value) =>
                value &&
                onChange({
                  ...layer,
                  windowOperations: {
                    ...layer.windowOperations,
                    moveButton: value === "none" ? "" : value,
                  },
                })
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {Object.entries(
                    windowButtonItems(
                      layer.windowOperations.moveButton,
                      layer.windowOperations.resizeButton,
                    ),
                  ).map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <Separator />

          <div className="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-3 px-3 py-2">
            <div className="text-sm font-medium">调整窗口尺寸</div>
            <Select
              items={windowButtonItems(
                layer.windowOperations.resizeButton,
                layer.windowOperations.moveButton,
              )}
              value={layer.windowOperations.resizeButton || "none"}
              onValueChange={(value) =>
                value &&
                onChange({
                  ...layer,
                  windowOperations: {
                    ...layer.windowOperations,
                    resizeButton: value === "none" ? "" : value,
                  },
                })
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {Object.entries(
                    windowButtonItems(
                      layer.windowOperations.resizeButton,
                      layer.windowOperations.moveButton,
                    ),
                  ).map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <Separator />

          <div className="grid grid-cols-[8rem_minmax(0,1fr)_auto] items-center gap-3 px-3 py-2">
            <div className="text-sm font-medium">窗口中心缩放</div>
            <div className="flex min-w-0 items-center text-sm">
              {layer.windowOperations.scaleWheel && (
                <>
                  <div className="flex shrink-0 items-center gap-2 whitespace-nowrap">
                    <span>每</span>
                    <NumberInput
                      className="w-16"
                      min={1}
                      max={20}
                      step={1}
                      value={layer.windowOperations.wheelSteps}
                      onValueChange={(wheelSteps) =>
                        onChange({
                          ...layer,
                          windowOperations: {
                            ...layer.windowOperations,
                            wheelSteps: Math.round(wheelSteps),
                          },
                        })
                      }
                      aria-label="缩放触发格数"
                    />
                    <span>格触发一次</span>
                  </div>
                  <span>，</span>
                  <div className="flex shrink-0 items-center gap-2 whitespace-nowrap">
                    <span>每次变化</span>
                    <NumberInput
                      className="w-16"
                      min={1}
                      max={50}
                      step={1}
                      value={layer.windowOperations.scaleStepPercent}
                      onValueChange={(scaleStepPercent) =>
                        onChange({
                          ...layer,
                          windowOperations: {
                            ...layer.windowOperations,
                            scaleStepPercent: Math.round(scaleStepPercent),
                          },
                        })
                      }
                      aria-label="每次缩放百分比"
                    />
                    <span>%</span>
                  </div>
                </>
              )}
            </div>
            <Switch
              size="sm"
              checked={layer.windowOperations.scaleWheel}
              disabled={wheelShortcutUsed}
              onCheckedChange={(scaleWheel) =>
                onChange({
                  ...layer,
                  windowOperations: {
                    ...layer.windowOperations,
                    scaleWheel,
                  },
                })
              }
              aria-label="启用滚轮中心缩放"
            />
          </div>
        </CardContent>
      </Card>
      )}

      <MouseShortcutDialog
        open={dialog.open}
        binding={dialog.open ? dialog.binding : undefined}
        availableInputs={
          dialog.open
            ? mouseInputs.filter((input) =>
                input === dialog.binding?.input || availableShortcutInputs().includes(input),
              )
            : []
        }
        existingKeys={new Set(layer.mouseShortcuts.map(mouseShortcutKey))}
        originalKey={dialog.open ? dialog.originalKey : undefined}
        presets={presets}
        onClose={() => setDialog({ open: false })}
        onSave={(binding) =>
          saveShortcut(
            binding,
            dialog.open ? dialog.originalKey : undefined,
          )
        }
      />
    </div>
  )
}

function MouseShortcutDialog({
  open,
  binding,
  availableInputs,
  existingKeys,
  originalKey,
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  binding?: MouseShortcutBinding
  availableInputs: MouseInput[]
  existingKeys: Set<string>
  originalKey?: string
  presets: ActionPreset[]
  onClose: () => void
  onSave: (binding: MouseShortcutBinding) => void
}) {
  const inputItems = useMemo<Record<string, string>>(
    () =>
      Object.fromEntries(
        availableInputs.map((input) => [input, mouseInputLabel(input)]),
      ),
    [availableInputs],
  )
  const [draft, setDraft] = useState<MouseShortcutBinding>({
    input: "left",
    enabled: true,
    wheelSteps: 1,
    modifiers: [],
    action: defaultMouseShortcutAction(),
  })

  useEffect(() => {
    if (!open) return
    if (binding) {
      setDraft(structuredClone(binding))
      return
    }
    setDraft({
      input: availableInputs[0] ?? "left",
      enabled: true,
      wheelSteps: 1,
      modifiers: [],
      action: defaultMouseShortcutAction(),
    })
  }, [availableInputs, binding, open])

  const draftKey = mouseShortcutKey(draft)
  const duplicate = existingKeys.has(draftKey) && draftKey !== originalKey

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="h-[calc(100vh-3rem)] max-h-[44rem] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>
            {binding ? "编辑鼠标快捷键" : "添加鼠标快捷键"}
          </DialogTitle>
        </DialogHeader>

        <ScrollArea className="min-h-0">
          <div className="flex flex-col gap-4 p-1 pr-3">
            <Field>
              <FieldLabel>鼠标输入</FieldLabel>
              <Select
                items={inputItems}
                value={draft.input}
                onValueChange={(input) => {
                  if (!input) return
                  setDraft({
                    ...draft,
                    input,
                    wheelSteps: input.startsWith("wheel")
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
                    {Object.entries(inputItems).map(([value, label]) => (
                      <SelectItem key={value} value={value}>
                        {label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            {(draft.input === "wheelUp" || draft.input === "wheelDown") && (
              <WheelStepsInput
                value={draft.wheelSteps!}
                onValueChange={(wheelSteps) =>
                  setDraft({ ...draft, wheelSteps })
                }
              />
            )}

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
                {mouseModifiers.map((modifier) => (
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

            <ActionEditor
              action={draft.action}
              presets={presets}
              onChange={(action) => setDraft({ ...draft, action })}
            />
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={
              !availableInputs.includes(draft.input as MouseInput) || duplicate
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
