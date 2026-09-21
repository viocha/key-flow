import { useEffect, useMemo, useState } from "react"
import type {
  ActionPreset,
  MouseShortcutBinding,
} from "../../bindings/key-flow"
import {
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"

import { ActionEditor } from "@/components/action-editor"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Field, FieldLabel } from "@/components/ui/field"
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
  actionSummary,
  defaultMouseShortcutAction,
  mouseInputForTrigger,
  mouseInputLabel,
  type UiLayer,
} from "@/lib/app-model"

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
        originalInput?: string
      }
  >({ open: false })

  const triggerInput = mouseInputForTrigger(layer.trigger)
  const shortcutInputs = new Set(
    layer.mouseShortcuts.map((binding) => binding.input),
  )

  const availableShortcutInputs = (
    currentInput?: string,
  ): MouseInput[] =>
    mouseInputs.filter((input) => {
      if (input === currentInput) return true
      if (input === triggerInput) return false
      if (shortcutInputs.has(input)) return false
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
    originalInput?: string,
  ) => {
    const next = layer.mouseShortcuts.filter(
      (item) => item.input !== (originalInput ?? binding.input),
    )
    onChange({
      ...layer,
      mouseShortcuts: [...next, binding],
    })
  }

  const deleteShortcut = (input: string) => {
    onChange({
      ...layer,
      mouseShortcuts: layer.mouseShortcuts.filter(
        (binding) => binding.input !== input,
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
        <CardContent className="px-0">
          {layer.mouseShortcuts.length === 0 ? (
            <Empty className="min-h-40">
              <EmptyHeader>
                <EmptyTitle>暂无鼠标快捷键</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            layer.mouseShortcuts.map((binding, index) => (
              <div key={binding.input}>
                {index > 0 && <Separator />}
                <div className="flex min-w-0 items-center gap-3 px-3 py-2">
                  <div className="w-28 shrink-0 text-sm font-medium">
                    {mouseInputLabel(binding.input)}
                  </div>
                  <div className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                    {actionSummary(binding.action)}
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      className={buttonVariants({
                        variant: "ghost",
                        size: "icon-sm",
                        className:
                          "data-popup-open:bg-accent data-popup-open:text-accent-foreground",
                      })}
                      aria-label="更多操作"
                    >
                      <MoreHorizontalIcon />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-32">
                      <DropdownMenuGroup>
                        <DropdownMenuItem
                          onClick={() =>
                            setDialog({
                              open: true,
                              binding,
                              originalInput: binding.input,
                            })
                          }
                        >
                          <PencilIcon />
                          编辑
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => deleteShortcut(binding.input)}
                        >
                          <Trash2Icon />
                          删除
                        </DropdownMenuItem>
                      </DropdownMenuGroup>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            ))
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
            <div className="text-sm">拖动窗口</div>
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
            <div className="text-sm">调整窗口尺寸</div>
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

          <div className="flex items-center gap-3 px-3 py-2">
            <div className="w-32 shrink-0 text-sm">窗口中心缩放</div>
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
            {layer.windowOperations.scaleWheel && (
              <div className="ml-auto flex items-center gap-2">
                <span className="text-xs text-muted-foreground">每格</span>
                <Input
                  className="w-20"
                  type="number"
                  min={1}
                  max={50}
                  value={layer.windowOperations.scaleStepPercent}
                  onChange={(event) =>
                    onChange({
                      ...layer,
                      windowOperations: {
                        ...layer.windowOperations,
                        scaleStepPercent: Math.max(
                          1,
                          Math.min(50, Number(event.target.value) || 1),
                        ),
                      },
                    })
                  }
                />
                <span className="text-xs text-muted-foreground">%</span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
      )}

      <MouseShortcutDialog
        open={dialog.open}
        binding={dialog.open ? dialog.binding : undefined}
        availableInputs={
          dialog.open
            ? availableShortcutInputs(dialog.originalInput)
            : []
        }
        presets={presets}
        onClose={() => setDialog({ open: false })}
        onSave={(binding) =>
          saveShortcut(
            binding,
            dialog.open ? dialog.originalInput : undefined,
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
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  binding?: MouseShortcutBinding
  availableInputs: MouseInput[]
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
      action: defaultMouseShortcutAction(),
    })
  }, [availableInputs, binding, open])

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
                onValueChange={(input) =>
                  input && setDraft({ ...draft, input })
                }
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
            disabled={!availableInputs.includes(draft.input as MouseInput)}
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
