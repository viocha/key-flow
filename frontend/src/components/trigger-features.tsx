import { useEffect, useMemo, useRef, useState } from "react"
import type {
  ActionPreset,
  MouseShortcutBinding,
} from "../../bindings/key-flow"
import { PlusIcon } from "lucide-react"

import { ActionEditor } from "@/components/action-editor"
import { BidirectionalActionEditor } from "@/components/bidirectional-action-editor"
import { MouseModifierInputs } from "@/components/mouse-modifier-inputs"
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
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
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
import {
  actionsForMouseInputChange,
  actionReady,
  defaultMouseShortcutAction,
  isBidirectionalMouseInput,
  mouseInputForTrigger,
  mouseInputLabel,
  mouseShortcutInputs,
  mouseShortcutActionSummary,
  mouseShortcutKey,
  mouseShortcutKeys,
  mouseShortcutLabel,
  type UiLayer,
} from "@/lib/app-model"
import { useDialogTransition } from "@/hooks/use-dialog-transition"
import { showUndoDeleteToast } from "@/lib/undo-toast"

const mouseInputs = [
  "left",
  "right",
  "middle",
  "back",
  "forward",
  "wheelVertical",
  "wheelHorizontal",
  "wheelUp",
  "wheelDown",
  "wheelLeft",
  "wheelRight",
] as const

const mouseButtonInputs = [
  "left",
  "right",
  "middle",
  "back",
  "forward",
] as const

type MouseInput = (typeof mouseInputs)[number]

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
    layer.mouseShortcuts
      .filter((binding) => binding.enabled)
      .flatMap((binding) => mouseShortcutInputs(binding.input)),
  )

  const conflictsWithWindowOperations = (binding: MouseShortcutBinding) =>
    layer.windowOperations.enabled &&
    (mouseShortcutInputs(binding.input).includes(layer.windowOperations.moveButton ?? "") ||
      mouseShortcutInputs(binding.input).includes(layer.windowOperations.resizeButton ?? ""))

  const availableShortcutInputs = (): MouseInput[] =>
    mouseInputs.filter((input) => {
      if (input === triggerInput) return false
      if (layer.windowOperations.enabled) {
        if (input === layer.windowOperations.moveButton) return false
        if (input === layer.windowOperations.resizeButton) return false
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
          添加
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
                        ? "与窗口拖拽冲突"
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
                      {mouseShortcutActionSummary(
                        binding.input,
                        binding.action,
                        binding.oppositeAction,
                      )}
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
        <div className="text-sm font-medium">窗口拖拽</div>
        <Switch
          size="sm"
          checked={layer.windowOperations.enabled}
          onCheckedChange={setWindowOperationsEnabled}
          aria-label="启用窗口拖拽"
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
        existingBindings={layer.mouseShortcuts}
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
  existingBindings,
  originalKey,
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  binding?: MouseShortcutBinding
  availableInputs: MouseInput[]
  existingBindings: MouseShortcutBinding[]
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
    input: "",
    enabled: true,
    wheelSteps: 1,
    modifiers: [],
    action: defaultMouseShortcutAction(),
  })
  const dialogTransition = useDialogTransition(open, onClose)

  useEffect(() => {
    if (!open) return
    if (binding) {
      setDraft(structuredClone(binding))
      return
    }
    setDraft({
      input: "",
      enabled: true,
      wheelSteps: 1,
      modifiers: [],
      action: defaultMouseShortcutAction(),
    })
  }, [availableInputs, binding, open])

  const draftKeys = new Set(mouseShortcutKeys(draft))
  const duplicate = existingBindings.some(
    (item) =>
      mouseShortcutKey(item) !== originalKey &&
      mouseShortcutKeys(item).some((key) => draftKeys.has(key)),
  )
  const bidirectional = isBidirectionalMouseInput(draft.input)

  return (
    <Dialog
      open={dialogTransition.dialogOpen}
      onOpenChange={dialogTransition.onOpenChange}
      onOpenChangeComplete={dialogTransition.onOpenChangeComplete}
    >
      <DialogContent className="h-[calc(100vh-3rem)] max-h-[44rem] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>
            {binding ? "编辑鼠标快捷键" : "添加鼠标快捷键"}
          </DialogTitle>
        </DialogHeader>

        <ScrollArea className="-mr-3 min-h-0">
          <div className="flex flex-col gap-4 p-1 pr-4">
            <Field>
              <FieldLabel>鼠标输入</FieldLabel>
              <Select
                items={inputItems}
                value={draft.input}
                onValueChange={(input) => {
                  if (!input) return
                  const [action, oppositeAction] = actionsForMouseInputChange(
                    draft.input,
                    input,
                    draft.action,
                    draft.oppositeAction,
                  )
                  setDraft({
                    ...draft,
                    input,
                    wheelSteps: input.startsWith("wheel")
                      ? draft.wheelSteps ?? 1
                      : undefined,
                    action,
                    oppositeAction,
                  })
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="选择鼠标输入" />
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
              {duplicate && (
                <FieldError>与已有鼠标快捷键冲突</FieldError>
              )}
            </Field>

            <MouseModifierInputs
              value={draft.modifiers}
              onChange={(modifiers) => setDraft({ ...draft, modifiers })}
              wheelSteps={
                draft.input.startsWith("wheel")
                  ? draft.wheelSteps ?? 1
                  : undefined
              }
              onWheelStepsChange={(wheelSteps) =>
                setDraft({ ...draft, wheelSteps })
              }
            />

            {bidirectional && draft.oppositeAction ? (
              <BidirectionalActionEditor
                input={draft.input}
                action={draft.action}
                oppositeAction={draft.oppositeAction}
                presets={presets}
                onChange={(action, oppositeAction) =>
                  setDraft({ ...draft, action, oppositeAction })
                }
              />
            ) : (
              <ActionEditor
                action={draft.action}
                presets={presets}
                onChange={(action) => setDraft({ ...draft, action })}
              />
            )}
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={dialogTransition.requestClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={
              !availableInputs.includes(draft.input as MouseInput) ||
              duplicate ||
              !actionReady(draft.action) ||
              (bidirectional && (!draft.oppositeAction || !actionReady(draft.oppositeAction)))
            }
            onClick={() => {
              onSave(draft)
              dialogTransition.requestClose()
            }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
