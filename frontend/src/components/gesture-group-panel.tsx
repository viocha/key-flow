import { useState } from "react"
import type {
  ActionPreset,
  GestureBinding,
  GesturePoint,
} from "../../bindings/key-flow"
import {
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react"

import { GestureDialog } from "@/components/feature-dialogs"
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
  actionSummary,
  defaultGesture,
  triggerLabel,
  type UiGestureGroup,
} from "@/lib/app-model"

function GesturePathPreview({
  preview,
}: {
  preview: GesturePoint[]
}) {
  return (
    <div className="h-9 w-20 shrink-0 overflow-hidden rounded-md border bg-muted/30">
      <svg
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        className="size-full"
        aria-label="手势路径"
      >
        <polyline
          points={preview
            .map((point) => point.x * 100 + "," + point.y * 100)
            .join(" ")}
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          vectorEffect="non-scaling-stroke"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-foreground"
        />
      </svg>
    </div>
  )
}

export function GestureGroupPanel({
  group,
  presets,
  onChange,
  onEdit,
  onDelete,
}: {
  group: UiGestureGroup
  presets: ActionPreset[]
  onChange: (group: UiGestureGroup) => void
  onEdit: () => void
  onDelete: () => void
}) {
  const [dialog, setDialog] = useState<
    { open: false } | { open: true; binding?: GestureBinding }
  >({ open: false })

  const saveBinding = (binding: GestureBinding) => {
    const exists = group.bindings.some((item) => item.id === binding.id)
    onChange({
      ...group,
      bindings: exists
        ? group.bindings.map((item) =>
            item.id === binding.id ? binding : item,
          )
        : [...group.bindings, binding],
    })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <h1 className="font-heading text-lg font-semibold">
            {triggerLabel(group.trigger)}
          </h1>
          <Switch
            size="sm"
            checked={group.enabled}
            onCheckedChange={(enabled) => onChange({ ...group, enabled })}
            aria-label={"启用 " + triggerLabel(group.trigger) + "手势"}
          />
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={onEdit}>
            设置
          </Button>
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
                <DropdownMenuItem variant="destructive" onClick={onDelete}>
                  <Trash2Icon />
                  删除
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className="flex justify-end">
        <Button
          type="button"
          size="sm"
          onClick={() => setDialog({ open: true, binding: defaultGesture() })}
        >
          <PlusIcon data-icon="inline-start" />
          添加手势
        </Button>
      </div>

      <Card className="gap-0 py-0">
        <CardContent className="px-0">
          {group.bindings.length === 0 ? (
            <Empty className="min-h-40">
              <EmptyHeader>
                <EmptyTitle>暂无手势</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            group.bindings.map((binding, index) => (
              <div key={binding.id}>
                {index > 0 && <Separator />}
                <div className="flex min-w-0 items-center gap-3 px-3 py-2">
                  <Switch
                    size="sm"
                    checked={binding.enabled}
                    onCheckedChange={(enabled) =>
                      saveBinding({ ...binding, enabled })
                    }
                    aria-label={"启用 " + binding.name}
                  />
                  <div className="w-32 shrink-0 truncate text-sm font-medium">
                    {binding.name}
                  </div>
                  <GesturePathPreview preview={binding.preview!} />
                  <div className="min-w-0 flex-1 truncate text-sm text-muted-foreground">
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
                            setDialog({ open: true, binding })
                          }
                        >
                          <PencilIcon />
                          编辑
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() =>
                            onChange({
                              ...group,
                              bindings: group.bindings.filter(
                                (item) => item.id !== binding.id,
                              ),
                            })
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

      <GestureDialog
        open={dialog.open}
        binding={dialog.open ? dialog.binding : undefined}
        presets={presets}
        onClose={() => setDialog({ open: false })}
        onSave={saveBinding}
      />
    </div>
  )
}
