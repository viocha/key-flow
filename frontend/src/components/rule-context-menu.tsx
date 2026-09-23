import { useRef, useState, type ReactNode } from "react"
import { PencilIcon, PowerIcon, Trash2Icon } from "lucide-react"

import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuGroup,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu"

export function RuleContextMenu({
  children,
  enabled,
  toggleDisabled,
  onToggle,
  onEdit,
  onDelete,
}: {
  children: ReactNode
  enabled: boolean
  toggleDisabled?: string
  onToggle: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const pendingAction = useRef<null | (() => void)>(null)
  const [menuVisible, setMenuVisible] = useState(false)

  return (
    <ContextMenu
      onOpenChange={(open) => {
        if (open) setMenuVisible(true)
      }}
      onOpenChangeComplete={(open) => {
        if (open) return
        setMenuVisible(false)
        const action = pendingAction.current
        pendingAction.current = null
        action?.()
      }}
    >
      <ContextMenuTrigger className={menuVisible ? "[&>div]:bg-muted/50" : undefined}>
        {children}
      </ContextMenuTrigger>
      <ContextMenuContent>
        <ContextMenuGroup>
          <ContextMenuItem
            disabled={Boolean(toggleDisabled)}
            onClick={() => { pendingAction.current = onToggle }}
          >
            <PowerIcon />
            {toggleDisabled ? `启用（${toggleDisabled}）` : enabled ? "禁用" : "启用"}
          </ContextMenuItem>
          <ContextMenuItem onClick={() => { pendingAction.current = onEdit }}>
            <PencilIcon />
            编辑
          </ContextMenuItem>
          <ContextMenuItem variant="destructive" onClick={() => { pendingAction.current = onDelete }}>
            <Trash2Icon />
            删除
          </ContextMenuItem>
        </ContextMenuGroup>
      </ContextMenuContent>
    </ContextMenu>
  )
}
