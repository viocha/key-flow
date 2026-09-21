import { useState } from "react"
import {
  EdgeAdjustmentType,
  type EdgeWheelRule,
} from "../../bindings/key-flow"
import {
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"

import { EdgeWheelDialog } from "@/components/feature-dialogs"
import { KeyChord } from "@/components/key-chord"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import {
  defaultEdgeWheel,
  edgeAdjustmentLabel,
  edgeZoneLabel,
} from "@/lib/app-model"

export function EdgeWheelPanel({
  enabled,
  rules,
  onEnabledChange,
  onChange,
}: {
  enabled: boolean
  rules: EdgeWheelRule[]
  onEnabledChange: (enabled: boolean) => void
  onChange: (rules: EdgeWheelRule[]) => void
}) {
  const [dialog, setDialog] = useState<
    { open: false } | { open: true; rule?: EdgeWheelRule }
  >({ open: false })

  const saveRule = (rule: EdgeWheelRule) => {
    const exists = rules.some((item) => item.id === rule.id)
    onChange(
      exists
        ? rules.map((item) => (item.id === rule.id ? rule : item))
        : [...rules, rule],
    )
  }

  return (
    <div className="flex flex-col gap-4">
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
      </div>

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
                <div className="flex min-w-0 items-center gap-3 px-3 py-2">
                  <Switch
                    size="sm"
                    checked={rule.enabled}
                    onCheckedChange={(enabled) =>
                      saveRule({ ...rule, enabled })
                    }
                    aria-label={"启用 " + edgeZoneLabel(rule.zone)}
                  />
                  <div className="w-24 shrink-0 text-sm font-medium">
                    {edgeZoneLabel(rule.zone)}
                  </div>
                  <div className="min-w-0 flex-1 text-sm">
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
                          onClick={() => setDialog({ open: true, rule })}
                        >
                          <PencilIcon />
                          编辑
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() =>
                            onChange(
                              rules.filter((item) => item.id !== rule.id),
                            )
                          }
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

      <EdgeWheelDialog
        open={dialog.open}
        rule={dialog.open ? dialog.rule : undefined}
        onClose={() => setDialog({ open: false })}
        onSave={saveRule}
      />
    </div>
  )
}
