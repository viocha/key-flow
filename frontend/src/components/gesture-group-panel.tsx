import { useRef, useState } from "react"
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

import { EditableListRow, stopListRowClick } from "@/components/editable-list-row"
import { GestureEndpointArrow } from "@/components/gesture-endpoint-arrow"
import { GestureDialog } from "@/components/feature-dialogs"
import { PanelLayout } from "@/components/panel-layout"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import {
  actionSummary,
  defaultGesture,
  gestureBindingLabel,
  triggerLabel,
  type UiGestureGroup,
} from "@/lib/app-model"
import { showUndoDeleteToast } from "@/lib/undo-toast"

function GesturePathPreview({
  preview,
  directions,
}: {
  preview: GesturePoint[]
  directions?: string[] | null
}) {
  const endpoint = preview[preview.length - 1]
  const endpointDirection = directions?.[directions.length - 1]

  return (
    <div className="relative h-9 w-20 shrink-0 overflow-hidden rounded-md border bg-muted/30">
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
          strokeWidth="1.6"
          vectorEffect="non-scaling-stroke"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="text-foreground"
        />
      </svg>
      {endpoint && (
        <GestureEndpointArrow
          x={endpoint.x}
          y={endpoint.y}
          direction={endpointDirection}
          size={12}
        />
      )}
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
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const groupRef = useRef(group)
  groupRef.current = group

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

  const deleteBinding = (binding: GestureBinding, index: number) => {
    onChange({
      ...group,
      bindings: group.bindings.filter((item) => item.id !== binding.id),
    })
    showUndoDeleteToast("手势", () => {
      const current = groupRef.current
      if (current.bindings.some((item) => item.id === binding.id)) return
      const bindings = [...current.bindings]
      bindings.splice(Math.min(index, bindings.length), 0, binding)
      onChange({ ...current, bindings })
    })
  }

  return (
    <PanelLayout
      header={
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
          <Button
            type="button"
            size="sm"
            onClick={() => setDialog({ open: true, binding: defaultGesture() })}
          >
            <PlusIcon data-icon="inline-start" />
            添加手势
          </Button>
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
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => setDeleteConfirmOpen(true)}
                >
                  <Trash2Icon />
                  删除
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
        </div>
      }
    >
      <Card className="gap-0 py-0">
        <CardContent className="px-0">
          {group.bindings.length === 0 ? (
            <Empty className="min-h-40">
              <EmptyHeader>
                <EmptyTitle>暂无手势</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            group.bindings.map((binding, index) => {
              const label = gestureBindingLabel(binding)
              const customName = binding.name?.trim()
              return (
                <div key={binding.id}>
                  {index > 0 && <Separator />}
                  <EditableListRow
                    onEdit={() => setDialog({ open: true, binding })}
                  >
                    <div onClick={stopListRowClick} onKeyDown={stopListRowClick}>
                      <Switch
                        size="sm"
                        checked={binding.enabled}
                        onCheckedChange={(enabled) =>
                          saveBinding({ ...binding, enabled })
                        }
                        aria-label={"启用 " + label}
                      />
                    </div>
                    <GesturePathPreview
                      preview={binding.preview!}
                      directions={binding.directions}
                    />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium">{label}</div>
                      {customName && (
                        <div className="truncate text-xs text-muted-foreground">
                          {actionSummary(binding.action)}
                        </div>
                      )}
                    </div>
                    <div onClick={stopListRowClick} onKeyDown={stopListRowClick}>
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
                              onClick={() => setDialog({ open: true, binding })}
                            >
                              <PencilIcon />
                              编辑
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              variant="destructive"
                              onClick={() => deleteBinding(binding, index)}
                            >
                              <Trash2Icon />
                              删除
                            </DropdownMenuItem>
                          </DropdownMenuGroup>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </EditableListRow>
                </div>
              )
            })
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

      <Dialog open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>删除手势分组？</DialogTitle>
            <DialogDescription>该分组及其中所有手势都会被删除。</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setDeleteConfirmOpen(false)}
            >
              取消
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => {
                setDeleteConfirmOpen(false)
                onDelete()
              }}
            >
              删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PanelLayout>
  )
}
