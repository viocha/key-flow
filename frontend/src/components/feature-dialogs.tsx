import { useEffect, useState } from "react"
import {
  EdgeAdjustmentType,
  type ActionPreset,
  type EdgeWheelRule,
  type GestureBinding,
} from "../../bindings/key-flow"

import { ActionEditor } from "@/components/action-editor"
import { GestureRecorder } from "@/components/gesture-recorder"
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
import { ScrollArea } from "@/components/ui/scroll-area"
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "cn"

import {
  defaultEdgeWheel,
  defaultGesture,
  defaultGestureGroup,
  type UiGestureGroup,
} from "@/lib/app-model"

const edgeZoneItems = {
  topLeft: "左上角",
  top: "上边缘",
  topRight: "右上角",
  left: "左边缘",
  right: "右边缘",
  bottomLeft: "左下角",
  bottom: "下边缘",
  bottomRight: "右下角",
}

const edgeAdjustmentItems = {
  volume: "音量",
  brightness: "亮度",
  recentApps: "最近应用",
  bidirectionalKeys: "双向快捷键",
}

const gestureTriggerItems = {
  MouseMiddle: "鼠标中键",
  MouseRight: "鼠标右键",
  MouseBack: "鼠标后退键",
  MouseForward: "鼠标前进键",
}

export function GestureGroupDialog({
  open,
  group,
  isNew,
  availableTriggers,
  onClose,
  onSave,
}: {
  open: boolean
  group?: UiGestureGroup
  isNew: boolean
  availableTriggers: string[]
  onClose: () => void
  onSave: (group: UiGestureGroup) => void
}) {
  const [draft, setDraft] = useState<UiGestureGroup>(() => defaultGestureGroup())

  useEffect(() => {
    if (open) {
      setDraft(group ? structuredClone(group) : defaultGestureGroup())
    }
  }, [group, open])

  const triggerItems = Object.fromEntries(
    Object.entries(gestureTriggerItems).filter(
      ([value]) => value === draft.trigger || availableTriggers.includes(value),
    ),
  )

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isNew ? "添加手势分组" : "手势设置"}</DialogTitle>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel>启动键</FieldLabel>
            <Select
              items={triggerItems}
              value={draft.trigger}
              onValueChange={(trigger) =>
                trigger && setDraft({ ...draft, trigger })
              }
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

          <div className="flex items-center justify-between gap-3">
            <div className="text-sm font-medium">显示轨迹</div>
            <Switch
              checked={draft.showTrail}
              onCheckedChange={(showTrail) =>
                setDraft({ ...draft, showTrail })
              }
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <Field>
              <FieldLabel htmlFor="gesture-group-start">启动距离</FieldLabel>
              <Input
                id="gesture-group-start"
                type="number"
                min={4}
                max={100}
                value={draft.startDistancePx}
                onChange={(event) =>
                  setDraft({
                    ...draft,
                    startDistancePx: Math.max(
                      4,
                      Math.min(100, Number(event.target.value) || 4),
                    ),
                  })
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="gesture-group-segment">分段距离</FieldLabel>
              <Input
                id="gesture-group-segment"
                type="number"
                min={4}
                max={200}
                value={draft.segmentDistancePx}
                onChange={(event) =>
                  setDraft({
                    ...draft,
                    segmentDistancePx: Math.max(
                      4,
                      Math.min(200, Number(event.target.value) || 4),
                    ),
                  })
                }
              />
            </Field>
          </div>
        </FieldGroup>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={!draft.trigger}
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

export function GestureDialog({
  open,
  binding,
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  binding?: GestureBinding
  presets: ActionPreset[]
  onClose: () => void
  onSave: (binding: GestureBinding) => void
}) {
  const [draft, setDraft] = useState<GestureBinding>(() => defaultGesture())

  useEffect(() => {
    if (open) {
      setDraft(binding ? structuredClone(binding) : defaultGesture())
    }
  }, [binding, open])

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="h-[calc(100vh-3rem)] max-h-[44rem] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{binding ? "编辑手势" : "添加手势"}</DialogTitle>
        </DialogHeader>
        <ScrollArea className="min-h-0">
          <div className="p-1">
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="gesture-name">名称</FieldLabel>
                <Input
                  id="gesture-name"
                  value={draft.name}
                  onChange={(event) =>
                    setDraft({ ...draft, name: event.target.value })
                  }
                />
              </Field>
              <Field>
                <FieldLabel>路径</FieldLabel>
                <GestureRecorder
                  preview={draft.preview}
                  directions={draft.directions}
                  onChange={(preview, directions) =>
                    setDraft({ ...draft, preview, directions })
                  }
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
            disabled={
              !draft.name.trim() ||
              !(draft.directions?.length) ||
              !draft.preview ||
              draft.preview.length < 2
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

export function EdgeWheelDialog({
  open,
  rule,
  onClose,
  onSave,
}: {
  open: boolean
  rule?: EdgeWheelRule
  onClose: () => void
  onSave: (rule: EdgeWheelRule) => void
}) {
  const [draft, setDraft] = useState<EdgeWheelRule>(() => defaultEdgeWheel())

  useEffect(() => {
    if (open) {
      setDraft(rule ? structuredClone(rule) : defaultEdgeWheel())
    }
  }, [open, rule])

  const isRecentApps =
    draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentRecentApps
  const isBidirectionalKeys =
    draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentBidirectionalKeys
  const noStep = isRecentApps || isBidirectionalKeys
  const maxStep =
    draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentBrightness ? 100 : 20

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{rule ? "编辑屏幕边缘" : "添加屏幕边缘"}</DialogTitle>
        </DialogHeader>

        <FieldGroup>
          <div className="grid grid-cols-[minmax(0,1fr)_8rem] gap-3">
            <Field>
              <FieldLabel>方位</FieldLabel>
              <Select
                items={edgeZoneItems}
                value={draft.zone}
                onValueChange={(value) =>
                  value && setDraft({ ...draft, zone: value })
                }
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {Object.entries(edgeZoneItems).map(([value, label]) => (
                      <SelectItem key={value} value={value}>
                        {label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel htmlFor="edge-margin">边缘范围</FieldLabel>
              <Input
                id="edge-margin"
                type="number"
                min={1}
                max={64}
                value={draft.marginPx}
                onChange={(event) =>
                  setDraft({
                    ...draft,
                    marginPx: Math.max(
                      1,
                      Math.min(64, Number(event.target.value) || 1),
                    ),
                  })
                }
              />
            </Field>
          </div>

          <div
            className={cn(
              "grid gap-3",
              noStep
                ? "grid-cols-1"
                : "grid-cols-[minmax(0,1fr)_8rem]",
            )}
          >
            <Field>
              <FieldLabel>调节项</FieldLabel>
              <Select
                items={edgeAdjustmentItems}
                value={draft.adjustment}
                onValueChange={(value) => {
                  if (!value) return
                  const adjustment = value as EdgeAdjustmentType
                  setDraft({
                    ...draft,
                    adjustment,
                    step:
                      adjustment ===
                      EdgeAdjustmentType.EdgeAdjustmentBrightness
                        ? 5
                        : 1,
                    ...(adjustment ===
                    EdgeAdjustmentType.EdgeAdjustmentBidirectionalKeys
                      ? {
                          upKeys:
                            draft.upKeys?.length
                              ? draft.upKeys
                              : ["Ctrl", "Shift", "Tab"],
                          downKeys:
                            draft.downKeys?.length
                              ? draft.downKeys
                              : ["Ctrl", "Tab"],
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
                    <SelectItem value={EdgeAdjustmentType.EdgeAdjustmentVolume}>
                      音量
                    </SelectItem>
                    <SelectItem
                      value={EdgeAdjustmentType.EdgeAdjustmentBrightness}
                    >
                      亮度
                    </SelectItem>
                    <SelectItem
                      value={EdgeAdjustmentType.EdgeAdjustmentRecentApps}
                    >
                      最近应用
                    </SelectItem>
                    <SelectItem
                      value={
                        EdgeAdjustmentType.EdgeAdjustmentBidirectionalKeys
                      }
                    >
                      双向快捷键
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            {!noStep && (
              <Field>
                <FieldLabel htmlFor="edge-step">每格步长</FieldLabel>
                <Input
                  id="edge-step"
                  type="number"
                  min={1}
                  max={maxStep}
                  value={draft.step}
                  onChange={(event) =>
                    setDraft({
                      ...draft,
                      step: Math.max(
                        1,
                        Math.min(maxStep, Number(event.target.value) || 1),
                      ),
                    })
                  }
                />
              </Field>
            )}
          </div>

          {isRecentApps && (
            <Field className="w-full">
              <FieldLabel>操作</FieldLabel>
              <FieldDescription className="w-full">
                滚轮切换最近应用；左键确认，右键取消
              </FieldDescription>
            </Field>
          )}

          {isBidirectionalKeys && (
            <FieldGroup className="gap-3">
              <Field>
                <FieldLabel>滚轮向上</FieldLabel>
                <KeyRecorder
                  value={draft.upKeys}
                  onChange={(upKeys) => setDraft({ ...draft, upKeys })}
                  className="w-full"
                />
              </Field>
              <Field>
                <FieldLabel>滚轮向下</FieldLabel>
                <KeyRecorder
                  value={draft.downKeys}
                  onChange={(downKeys) => setDraft({ ...draft, downKeys })}
                  className="w-full"
                />
              </Field>
            </FieldGroup>
          )}
        </FieldGroup>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            type="button"
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
