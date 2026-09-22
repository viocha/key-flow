import type { KeyboardEvent, ReactNode } from "react"
import { cn } from "cn"

export function EditableListRow({
  children,
  onEdit,
  className,
}: {
  children: ReactNode
  onEdit: () => void
  className?: string
}) {
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "Enter" && event.key !== " ") return
    event.preventDefault()
    onEdit()
  }

  return (
    <div
      role="button"
      tabIndex={0}
      className={cn(
        "flex min-w-0 cursor-pointer items-center gap-3 px-3 py-2 outline-none transition-colors hover:bg-muted/50 focus-visible:bg-muted/50",
        className,
      )}
      onClick={onEdit}
      onKeyDown={onKeyDown}
    >
      {children}
    </div>
  )
}

export function stopListRowClick(event: { stopPropagation: () => void }) {
  event.stopPropagation()
}
