import { useRef, useState } from "react"
import {
  EdgeAdjustmentType,
  type EdgeWheelRule,
} from "../../bindings/key-flow"
import {
  PlusIcon,
} from "lucide-react"

import { EditableListRow } from "@/components/editable-list-row"
import { EdgeWheelDialog } from "@/components/feature-dialogs"
import { KeyChord } from "@/components/key-chord"
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
  edgeZoneLabel,
} from "@/lib/app-model"
import { showUndoDeleteToast } from "@/lib/undo-toast"

export function EdgeWheelPanel({
  enabled,
  marginPx,
  rules,
  onEnabledChange,
  onMarginChange,
  onChange,
}: {
  enabled: boolean
  marginPx: number
  rules: EdgeWheelRule[]
  onEnabledChange: (enabled: boolean) => void
  onMarginChange: (marginPx: number) => void
  onChange: (rules: EdgeWheelRule[]) => void
}) {
  const [dialog, setDialog] = useState<
    { open: false } | { open: true; rule?: EdgeWheelRule }
  >({ open: false })
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [marginDraft, setMarginDraft] = useState(marginPx)
  const rulesRef = useRef(rules)
  rulesRef.current = rules

  const saveRule = (rule: EdgeWheelRule) => {
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
            添加规则
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
        <CardContent className="px-0">
          {rules.length === 0 ? (
            <Empty className="min-h-40">
              <EmptyHeader>
                <EmptyTitle>暂无屏幕边缘规则</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            rules.map((rule, index) => (
              <div key={rule.id}>
                {index > 0 && <Separator />}
                <RuleContextMenu
                  enabled={rule.enabled}
                  onToggle={() => saveRule({ ...rule, enabled: !rule.enabled })}
                  onEdit={() => setDialog({ open: true, rule })}
                  onDelete={() => deleteRule(rule, index)}
                >
                <EditableListRow onEdit={() => setDialog({ open: true, rule })}>
                  <div className="w-24 shrink-0 text-sm font-medium">
                    {edgeZoneLabel(rule.zone)}
                  </div>
                  <div className="flex min-w-0 flex-1 items-center gap-2 text-sm">
                    <div className="min-w-0 flex-1">
                      {rule.adjustment ===
                      EdgeAdjustmentType.EdgeAdjustmentBidirectionalKeys ? (
                        <div className="flex min-w-0 items-center gap-4">
                          <div className="flex min-w-0 items-center gap-2">
                            <span className="shrink-0 text-muted-foreground">↑</span>
                            <KeyChord keys={rule.upKeys} />
                          </div>
                          <div className="flex min-w-0 items-center gap-2">
                            <span className="shrink-0 text-muted-foreground">↓</span>
                            <KeyChord keys={rule.downKeys} />
                          </div>
                        </div>
                      ) : (
                        edgeAdjustmentLabel(rule.adjustment)
                      )}
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
        onClose={() => setDialog({ open: false })}
        onSave={saveRule}
      />

      <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
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
              onClick={() => setSettingsOpen(false)}
            >
              取消
            </Button>
            <Button
              type="button"
              onClick={() => {
                onMarginChange(marginDraft)
                setSettingsOpen(false)
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
