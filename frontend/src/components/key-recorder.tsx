import { useEffect, useMemo, useState } from "react"
import { KeyboardIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { KeyChord } from "@/components/key-chord"
import { Toggle } from "@/components/ui/toggle"
import { cn } from "cn"

const modifierNames = new Set(["Ctrl", "Shift", "Alt", "Win"])

export function KeyRecorder({
  value,
  onChange,
  className,
  allowWinModifier = false,
}: {
  value?: string[] | null
  onChange: (keys: string[]) => void
  className?: string
  allowWinModifier?: boolean
}) {
  const [recording, setRecording] = useState(false)
  const keys = value ?? []

  useEffect(() => {
    if (!recording) {
      return
    }

    const onKeyDown = (event: KeyboardEvent) => {
      event.preventDefault()
      event.stopPropagation()

      const main = normalizeKeyboardEvent(event)
      if (!main || modifierNames.has(main)) {
        return
      }

      const next: string[] = []
      if (event.ctrlKey) next.push("Ctrl")
      if (event.shiftKey) next.push("Shift")
      if (event.altKey) next.push("Alt")
      if (event.metaKey || (allowWinModifier && keys.includes("Win"))) next.push("Win")
      next.push(main)
      onChange(next)
      setRecording(false)
    }

    window.addEventListener("keydown", onKeyDown, true)
    return () => window.removeEventListener("keydown", onKeyDown, true)
  }, [allowWinModifier, keys, onChange, recording])

  const label = useMemo(
    () => (recording ? "请按组合键…" : keys.length ? null : "录制按键"),
    [keys.length, recording],
  )

  return (
    <div className="flex w-full min-w-0 items-center gap-2">
      <Button
        type="button"
        variant="outline"
        className={cn("h-9 min-w-0 flex-1 justify-start font-normal", className)}
        onClick={() => setRecording((current) => !current)}
      >
        <KeyboardIcon data-icon="inline-start" />
        {label ?? <KeyChord keys={keys} />}
      </Button>
      {allowWinModifier && (
        <Toggle
          type="button"
          variant="outline"
          className="h-9 shrink-0"
          pressed={keys.includes("Win")}
          onPressedChange={(pressed) => {
            setRecording(false)
            const next = keys.filter((key) => key !== "Win")
            if (pressed) next.splice(Math.max(0, next.length - 1), 0, "Win")
            onChange(next)
          }}
          aria-label="包含 Win 键"
        >
          Win
        </Toggle>
      )}
    </div>
  )
}

function normalizeKeyboardEvent(event: KeyboardEvent) {
  switch (event.key) {
    case "Control":
      return "Ctrl"
    case "Shift":
      return "Shift"
    case "Alt":
      return "Alt"
    case "Meta":
      return "Win"
    case " ":
      return "Space"
    case "Esc":
      return "Escape"
    default:
      break
  }

  if (/^Key[A-Z]$/.test(event.code)) {
    return event.code.slice(3)
  }
  if (/^Digit[0-9]$/.test(event.code)) {
    return event.code.slice(5)
  }
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(event.key)) {
    return event.key
  }

  const supported = new Set([
    "CapsLock",
    "Escape",
    "Enter",
    "Backspace",
    "Delete",
    "Insert",
    "Home",
    "End",
    "PageUp",
    "PageDown",
    "ArrowLeft",
    "ArrowRight",
    "ArrowUp",
    "ArrowDown",
    "Tab",
    "PrintScreen",
    "ScrollLock",
    "Pause",
    "NumLock",
    "ContextMenu",
  ])
  return supported.has(event.key) ? event.key : ""
}
