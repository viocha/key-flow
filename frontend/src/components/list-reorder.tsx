import { useRef, useState, type DragEvent, type KeyboardEvent } from "react"
import { GripVerticalIcon } from "lucide-react"
import { cn } from "cn"

import { Button } from "@/components/ui/button"

export function useListReorder<T>(
  items: T[],
  keyOf: (item: T) => string,
  onReorder: (items: T[]) => void,
) {
  const draggingKey = useRef<string | null>(null)
  const [dropIndex, setDropIndex] = useState<number | null>(null)

  const clearDrag = () => {
    draggingKey.current = null
    setDropIndex(null)
  }

  const insertionIndexAt = (event: DragEvent<HTMLDivElement>): number => {
    const rows = event.currentTarget.querySelectorAll<HTMLElement>("[data-sortable-item]")
    const index = Array.from(rows).findIndex((row) => {
      const rect = row.getBoundingClientRect()
      return event.clientY < rect.top + rect.height / 2
    })
    return index < 0 ? rows.length : index
  }

  const listProps = () => ({
    onDragEnterCapture: (event: DragEvent<HTMLDivElement>) => {
      if (!draggingKey.current) return
      event.preventDefault()
      event.dataTransfer.dropEffect = "move"
    },
    onDragOverCapture: (event: DragEvent<HTMLDivElement>) => {
      if (!draggingKey.current) return
      event.preventDefault()
      event.dataTransfer.dropEffect = "move"
      const index = insertionIndexAt(event)
      const sourceIndex = items.findIndex((item) => keyOf(item) === draggingKey.current)
      setDropIndex(index === sourceIndex || index === sourceIndex + 1 ? null : index)
    },
    onDropCapture: (event: DragEvent<HTMLDivElement>) => {
      const source = draggingKey.current
      if (!source) return
      event.preventDefault()
      const target = insertionIndexAt(event)
      clearDrag()
      const sourceIndex = items.findIndex((item) => keyOf(item) === source)
      if (sourceIndex < 0 || target === sourceIndex || target === sourceIndex + 1) return
      const next = [...items]
      const [moved] = next.splice(sourceIndex, 1)
      next.splice(target > sourceIndex ? target - 1 : target, 0, moved)
      onReorder(next)
    },
  })

  const itemProps = (key: string) => {
    const index = items.findIndex((item) => keyOf(item) === key)
    return {
      "data-sortable-item": true,
      className: cn(
        "relative",
        dropIndex === 0 && index === 0 &&
          "before:absolute before:inset-x-0 before:top-0 before:z-10 before:h-0.5 before:bg-primary",
        dropIndex === items.length && index === items.length - 1 &&
          "after:absolute after:inset-x-0 after:bottom-0 after:z-10 after:h-0.5 after:bg-primary",
      ),
    }
  }

  const separatorClassName = (index: number) =>
    dropIndex === index
      ? "relative bg-transparent before:pointer-events-none before:absolute before:inset-x-0 before:top-1/2 before:z-10 before:h-0.5 before:-translate-y-1/2 before:bg-primary"
      : undefined

  const handleProps = (key: string) => ({
    draggable: items.length > 1,
    onClick: (event: { stopPropagation: () => void }) => event.stopPropagation(),
    onKeyDown: (event: KeyboardEvent<HTMLButtonElement>) => {
      event.stopPropagation()
      if (event.key !== "ArrowUp" && event.key !== "ArrowDown") return
      event.preventDefault()
      const index = items.findIndex((item) => keyOf(item) === key)
      const other = index + (event.key === "ArrowUp" ? -1 : 1)
      if (index < 0 || other < 0 || other >= items.length) return
      const next = [...items]
      ;[next[index], next[other]] = [next[other], next[index]]
      onReorder(next)
    },
    onDragStart: (event: DragEvent<HTMLButtonElement>) => {
      draggingKey.current = key
      event.dataTransfer.effectAllowed = "move"
      event.dataTransfer.setData("text/plain", key)
      const row = event.currentTarget.closest("[data-sortable-item]")
      if (row) event.dataTransfer.setDragImage(row, 16, 16)
    },
    onDragEnd: clearDrag,
  })

  return { listProps, itemProps, separatorClassName, handleProps, canReorder: items.length > 1 }
}

export function ReorderHandle({
  props,
}: {
  props: ReturnType<ReturnType<typeof useListReorder>["handleProps"]>
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      className="shrink-0 cursor-grab text-muted-foreground active:cursor-grabbing"
      aria-label="拖动排序"
      title="拖动排序；方向键调整"
      {...props}
    >
      <GripVerticalIcon />
    </Button>
  )
}
