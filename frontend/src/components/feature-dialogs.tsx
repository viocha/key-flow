import { useEffect, useState } from "react"
import {
  ActionType,
  EdgeAdjustmentType,
  type ActionPreset,
  type EdgeWheelRule,
  type GestureBinding,
} from "../../bindings/key-flow"

import { ActionEditor } from "@/components/action-editor"
import { GestureRecorder } from "@/components/gesture-recorder"
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
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { NumberInput } from "@/components/ui/number-input"
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

import {
  defaultEdgeWheel,
  actionReady,
  defaultGesture,
  emptyAction,
  defaultGestureGroup,
  edgeWheelConflicts,
  gestureBindingConflicts,
  type UiGestureGroup,
} from "@/lib/app-model"
import { useDialogTransition } from "@/hooks/use-dialog-transition"

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

const edgeWheelInputItems = {
  wheelVertical: "垂直滚动",
  wheelHorizontal: "水平滚动",
}

const edgeAdjustmentItems = {
  volume: "音量",
  brightness: "亮度",
  recentApps: "浏览最近应用",
  switchApps: "切换应用",
  switchTabs: "切换标签页",
  page: "翻页",
  custom: "自定义动作",
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
  const dialog = useDialogTransition(open, onClose)

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
    <Dialog
      open={dialog.dialogOpen}
      onOpenChange={dialog.onOpenChange}
      onOpenChangeComplete={dialog.onOpenChangeComplete}
    >
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
                <SelectValue placeholder="选择启动键" />
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
              <NumberInput
                id="gesture-group-start"
                min={4}
                max={100}
                value={draft.startDistancePx}
                onValueChange={(startDistancePx) =>
                  setDraft({
                    ...draft,
                    startDistancePx,
                  })
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="gesture-group-segment">分段距离</FieldLabel>
              <NumberInput
                id="gesture-group-segment"
                min={4}
                max={200}
                value={draft.segmentDistancePx}
                onValueChange={(segmentDistancePx) =>
                  setDraft({
                    ...draft,
                    segmentDistancePx,
                  })
                }
              />
            </Field>
          </div>
        </FieldGroup>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={dialog.requestClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={!draft.trigger}
            onClick={() => {
              onSave(draft)
              dialog.requestClose()
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
  existingBindings,
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  binding?: GestureBinding
  existingBindings: GestureBinding[]
  presets: ActionPreset[]
  onClose: () => void
  onSave: (binding: GestureBinding) => void
}) {
  const [draft, setDraft] = useState<GestureBinding>(() => defaultGesture())
  const dialog = useDialogTransition(open, onClose)
  const duplicate = gestureBindingConflicts(draft, existingBindings)

  useEffect(() => {
    if (open) {
      setDraft(binding ? structuredClone(binding) : defaultGesture())
    }
  }, [binding, open])

  return (
    <Dialog
      open={dialog.dialogOpen}
      onOpenChange={dialog.onOpenChange}
      onOpenChangeComplete={dialog.onOpenChangeComplete}
    >
      <DialogContent className="h-[calc(100vh-3rem)] max-h-[44rem] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{binding ? "编辑手势" : "添加手势"}</DialogTitle>
        </DialogHeader>
        <ScrollArea gutter="dialog" className="min-h-0">
          <div className="p-1">
            <FieldGroup>
              <Field>
                <FieldLabel>路径</FieldLabel>
                <GestureRecorder
                  preview={draft.preview}
                  directions={draft.directions}
                  onChange={(preview, directions) =>
                    setDraft({ ...draft, preview, directions })
                  }
                />
                {duplicate && <FieldError>与已有手势冲突</FieldError>}
              </Field>
              <Field>
                <FieldLabel htmlFor="gesture-name">名称</FieldLabel>
                <Input
                  id="gesture-name"
                  value={draft.name}
                  placeholder="可选，用于覆盖默认动作名称"
                  onChange={(event) =>
                    setDraft({ ...draft, name: event.target.value })
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
          <Button type="button" variant="outline" onClick={dialog.requestClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={
              !(draft.directions?.length) ||
              !draft.preview ||
              draft.preview.length < 2 ||
              duplicate ||
              !actionReady(draft.action)
            }
            onClick={() => {
              onSave(draft)
              dialog.requestClose()
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
  existingRules,
  presets,
  onClose,
  onSave,
}: {
  open: boolean
  rule?: EdgeWheelRule
  existingRules: EdgeWheelRule[]
  presets: ActionPreset[]
  onClose: () => void
  onSave: (rule: EdgeWheelRule) => void
}) {
  const [draft, setDraft] = useState<EdgeWheelRule>(() => defaultEdgeWheel())
  const dialog = useDialogTransition(open, onClose)
  const duplicate = edgeWheelConflicts(draft, existingRules)

  useEffect(() => {
    if (open) {
      setDraft(rule ? structuredClone(rule) : defaultEdgeWheel())
    }
  }, [open, rule])

  const isRecentApps =
    draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentRecentApps
  const isSwitchApps =
    draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentSwitchApps
  const isSwitchTabs =
    draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentSwitchTabs
  const isPage = draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentPage
  const isCustom = draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentCustom
  const customActionsIncomplete =
    isCustom &&
    (!draft.action ||
      !draft.oppositeAction ||
      (draft.action.type === ActionType.ActionKeyStroke && !draft.action.keys?.length) ||
      (draft.oppositeAction.type === ActionType.ActionKeyStroke &&
        !draft.oppositeAction.keys?.length))
  const incomplete = !draft.zone || !draft.input || customActionsIncomplete
  const [firstDirectionLabel, secondDirectionLabel] =
    draft.input === "wheelHorizontal"
      ? ["滚轮向左", "滚轮向右"]
      : ["滚轮向上", "滚轮向下"]
  const noStep =
    isRecentApps ||
    isSwitchApps ||
    isSwitchTabs ||
    isPage ||
    isCustom
  const maxStep = draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentVolume ? 20 : 100

  return (
    <Dialog
      open={dialog.dialogOpen}
      onOpenChange={dialog.onOpenChange}
      onOpenChangeComplete={dialog.onOpenChangeComplete}
    >
      <DialogContent className="h-[calc(100vh-3rem)] max-h-[42rem] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{rule ? "编辑屏幕边缘" : "添加屏幕边缘"}</DialogTitle>
        </DialogHeader>

        <ScrollArea gutter="dialog" className="min-h-0">
          <div className="p-1">
            <FieldGroup>
              <Field>
                <FieldLabel>方位</FieldLabel>
                <Select
                  items={edgeZoneItems}
                  value={draft.zone || null}
                  onValueChange={(value) =>
                    value && setDraft({ ...draft, zone: value })
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="选择方位" />
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
                <FieldLabel>滚轮方向</FieldLabel>
                <Select
                  items={edgeWheelInputItems}
                  value={draft.input || null}
                  onValueChange={(input) => input && setDraft({ ...draft, input })}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="选择滚轮方向" />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {Object.entries(edgeWheelInputItems).map(([value, label]) => (
                        <SelectItem key={value} value={value}>
                          {label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                {duplicate && <FieldError>与已有屏幕边缘规则冲突</FieldError>}
              </Field>

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
                    adjustment === EdgeAdjustmentType.EdgeAdjustmentBrightness
                      ? 5
                      : 1,
                  ...(adjustment === EdgeAdjustmentType.EdgeAdjustmentCustom
                    ? {
                        action: draft.action ?? emptyAction(),
                        oppositeAction: draft.oppositeAction ?? emptyAction(),
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
                    浏览最近应用
                  </SelectItem>
                  <SelectItem value={EdgeAdjustmentType.EdgeAdjustmentSwitchApps}>
                    切换应用
                  </SelectItem>
                  <SelectItem value={EdgeAdjustmentType.EdgeAdjustmentSwitchTabs}>
                    切换标签页
                  </SelectItem>
                  <SelectItem value={EdgeAdjustmentType.EdgeAdjustmentPage}>
                    翻页
                  </SelectItem>
                  <SelectItem value={EdgeAdjustmentType.EdgeAdjustmentCustom}>
                    自定义动作
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>

          <Field>
            <FieldLabel htmlFor="edge-wheel-steps">灵敏度</FieldLabel>
            <div className="flex items-center text-sm">
              <div className="flex items-center gap-2 whitespace-nowrap">
                <span>每</span>
                <NumberInput
                  id="edge-wheel-steps"
                  className="w-16"
                  min={1}
                  max={20}
                  step={1}
                  value={draft.wheelSteps}
                  onValueChange={(wheelSteps) =>
                    setDraft({ ...draft, wheelSteps: Math.round(wheelSteps) })
                  }
                />
                <span>格触发一次</span>
              </div>

              {!noStep && (
                <>
                  <span>，</span>
                  <div className="flex items-center gap-2 whitespace-nowrap">
                    <span>
                      {draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentVolume
                        ? "每次触发"
                        : "每次变化"}
                    </span>
                    <NumberInput
                      className="w-16"
                      min={1}
                      max={maxStep}
                      step={1}
                      value={draft.step}
                      onValueChange={(step) =>
                        setDraft({ ...draft, step: Math.round(step) })
                      }
                      aria-label={
                        draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentVolume
                          ? "每次触发按键次数"
                          : "每次变化量"
                      }
                    />
                    <span>
                      {draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentBrightness
                        ? "%"
                        : draft.adjustment === EdgeAdjustmentType.EdgeAdjustmentVolume
                          ? "次"
                          : "档"}
                    </span>
                  </div>
                </>
              )}
            </div>
          </Field>

          {isRecentApps && (
            <Field className="w-full">
              <FieldLabel>操作</FieldLabel>
              <FieldDescription className="w-full">
                滚轮浏览最近应用；左键确认，右键取消
              </FieldDescription>
            </Field>
          )}


              {isCustom && draft.action && draft.oppositeAction && (
                <FieldGroup className="gap-4">
                  <ActionEditor
                    action={draft.action}
                    presets={presets}
                    label={`${firstDirectionLabel}动作`}
                    onChange={(action) => setDraft({ ...draft, action })}
                  />
                  <ActionEditor
                    action={draft.oppositeAction}
                    presets={presets}
                    label={`${secondDirectionLabel}动作`}
                    onChange={(oppositeAction) =>
                      setDraft({ ...draft, oppositeAction })
                    }
                  />
                </FieldGroup>
              )}
            </FieldGroup>
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={dialog.requestClose}>
            取消
          </Button>
          <Button
            type="button"
            disabled={duplicate || incomplete}
            onClick={() => {
              onSave(draft)
              dialog.requestClose()
            }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
