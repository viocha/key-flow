import { useRef, useState } from "react"
import {
  EdgeAdjustmentType,
  type ActionPreset,
  type EdgeWheelRule,
} from "../../bindings/key-flow"
import {
  PlusIcon,
} from "lucide-react"

import { EditableListRow } from "@/components/editable-list-row"
import { EdgeWheelDialog } from "@/components/feature-dialogs"
import { ReorderHandle, useListReorder } from "@/components/list-reorder"
import { PanelLayout } from "@/components/panel-layout"
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
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import {
  defaultEdgeWheel,
  edgeAdjustmentLabel,
  edgeWheelConflicts,
  edgeWheelInputLabel,
  edgeZoneLabel,
  mouseShortcutActionSummary,
} from "@/lib/app-model"
import { useDialogTransition } from "@/hooks/use-dialog-transition"
import { showUndoDeleteToast } from "@/lib/undo-toast"

export function EdgeWheelPanel({
  enabled,
  marginPx,
  rules,
  presets,
  onEnabledChange,
  onMarginChange,
  onChange,
}: {
  enabled: boolean
  marginPx: number
  rules: EdgeWheelRule[]
  presets: ActionPreset[]
  onEnabledChange: (enabled: boolean) => void
  onMarginChange: (marginPx: number) => void
  onChange: (rules: EdgeWheelRule[]) => void
}) {
  const [dialog, setDialog] = useState<
    { open: false } | { open: true; rule?: EdgeWheelRule }
  >({ open: false })
  const [settingsOpen, setSettingsOpen] = useState(false)
  const settingsDialog = useDialogTransition(settingsOpen, () => setSettingsOpen(false))
  const [marginDraft, setMarginDraft] = useState(marginPx)
  const rulesRef = useRef(rules)
  rulesRef.current = rules
  const reorder = useListReorder(rules, (rule) => rule.id, onChange)

  const saveRule = (rule: EdgeWheelRule) => {
    if (edgeWheelConflicts(rule, rules)) return
    const exists = rules.some((item) => item.id === rule.id)
    onChange(
      exists
        ? rules.map((item) => (item.id === rule.id ? rule : item))
        : [...rules, rule],
    )
  }

  const deleteRule = (rule: EdgeWheelRule, index: number) => {
    onChange(rules.filter((item) => item.id !== rule.id))
    showUndoDeleteToast("屏幕边缘规则", () => {
      const current = rulesRef.current
      if (current.some((item) => item.id === rule.id)) return
      const next = [...current]
      next.splice(Math.min(index, next.length), 0, rule)
      onChange(next)
    })
  }

  return (
    <PanelLayout
      header={
        <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <h1 className="font-heading text-lg font-semibold">屏幕边缘</h1>
          <Switch
            size="sm"
            checked={enabled}
            onCheckedChange={onEnabledChange}
            aria-label="启用屏幕边缘"
          />
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            type="button"
            size="sm"
            onClick={() =>
              setDialog({ open: true, rule: defaultEdgeWheel() })
            }
          >
            <PlusIcon data-icon="inline-start" />
            添加
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              setMarginDraft(marginPx)
              setSettingsOpen(true)
            }}
          >
            设置
          </Button>
        </div>
        </div>
      }
    >
      <Card className="gap-0 py-0">
        <CardContent className="px-0" {...reorder.listProps()}>
          {rules.length === 0 ? (
            <Empty className="min-h-40">
              <EmptyHeader>
                <EmptyTitle>暂无屏幕边缘规则</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            rules.map((rule, index) => (
              <div key={rule.id} {...reorder.itemProps(rule.id)}>
                {index > 0 && <Separator className={reorder.separatorClassName(index)} />}
                <RuleContextMenu
                  enabled={rule.enabled}
                  onToggle={() => saveRule({ ...rule, enabled: !rule.enabled })}
                  onEdit={() => setDialog({ open: true, rule })}
                  onDelete={() => deleteRule(rule, index)}
                >
                <EditableListRow onEdit={() => setDialog({ open: true, rule })}>
                  {reorder.canReorder && <ReorderHandle props={reorder.handleProps(rule.id)} />}
                  <div className="flex min-w-0 flex-1 items-center gap-4 text-sm">
                    <div className="shrink-0 font-medium">
                      {edgeZoneLabel(rule.zone)}
                    </div>
                    <div className="shrink-0 text-muted-foreground">
                      {edgeWheelInputLabel(rule.input)}
                    </div>
                    <div className="min-w-0 flex-1">
                      {rule.adjustment === EdgeAdjustmentType.EdgeAdjustmentCustom &&
                      rule.action &&
                      rule.oppositeAction
                        ? mouseShortcutActionSummary(
                            rule.input,
                            rule.action,
                            rule.oppositeAction,
                          )
                        : edgeAdjustmentLabel(rule.adjustment)}
                    </div>
                  </div>
                  {!rule.enabled && <Badge variant="secondary" className="ml-auto shrink-0">已禁用</Badge>}
                </EditableListRow>
                </RuleContextMenu>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      <EdgeWheelDialog
        open={dialog.open}
        rule={dialog.open ? dialog.rule : undefined}
        existingRules={rules}
        presets={presets}
        onClose={() => setDialog({ open: false })}
        onSave={saveRule}
      />

      <Dialog
        open={settingsDialog.dialogOpen}
        onOpenChange={settingsDialog.onOpenChange}
        onOpenChangeComplete={settingsDialog.onOpenChangeComplete}
      >
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>屏幕边缘设置</DialogTitle>
          </DialogHeader>
          <Field>
            <FieldLabel htmlFor="edge-wheel-margin">触发范围</FieldLabel>
            <NumberInput
              id="edge-wheel-margin"
              min={1}
              max={64}
              value={marginDraft}
              onValueChange={setMarginDraft}
            />
          </Field>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={settingsDialog.requestClose}
            >
              取消
            </Button>
            <Button
              type="button"
              onClick={() => {
                onMarginChange(marginDraft)
                settingsDialog.requestClose()
              }}
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PanelLayout>
  )
}
