import { useEffect, useMemo, useRef, useState } from "react"
import { CommandIcon, KeyboardIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { KeyChord } from "@/components/key-chord"
import { Toggle } from "@/components/ui/toggle"
import { useInputCapture } from "@/hooks/use-input-capture"
import { cn } from "cn"

const modifierNames = new Set(["Win", "Ctrl", "Alt", "Shift"])

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
  const buttonRef = useRef<HTMLButtonElement>(null)
  const { begin: beginInputCapture, end: endInputCapture } = useInputCapture()
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
      if (event.metaKey || (allowWinModifier && keys.includes("Win"))) next.push("Win")
      if (event.ctrlKey) next.push("Ctrl")
      if (event.altKey) next.push("Alt")
      if (event.shiftKey) next.push("Shift")
      next.push(main)
      onChange(next)
      setRecording(false)
      endInputCapture()
      buttonRef.current?.blur()
    }

    window.addEventListener("keydown", onKeyDown, true)
    return () => window.removeEventListener("keydown", onKeyDown, true)
  }, [allowWinModifier, endInputCapture, keys, onChange, recording])

  const label = useMemo(
    () => (recording ? "请输入按键..." : keys.length ? null : "录制按键"),
    [keys.length, recording],
  )

  return (
    <div className="flex w-full min-w-0 items-center gap-2">
      <Button
        ref={buttonRef}
        type="button"
        variant="outline"
        aria-pressed={recording}
        className={cn(
          "h-9 min-w-0 flex-1 justify-start font-normal",
          recording
            ? "border-ring bg-muted ring-3 ring-ring/30 hover:bg-muted"
            : "focus-visible:border-border focus-visible:ring-0",
          className,
        )}
        onClick={() => {
          if (recording) {
            setRecording(false)
            endInputCapture()
            buttonRef.current?.blur()
            return
          }
          beginInputCapture()
          setRecording(true)
        }}
        onBlur={() => {
          if (!recording) return
          setRecording(false)
          endInputCapture()
        }}
      >
        <KeyboardIcon data-icon="inline-start" />
        {label ?? <KeyChord keys={keys} />}
      </Button>
      {allowWinModifier && (
        <Toggle
          type="button"
          variant="outline"
          className="size-9 shrink-0"
          pressed={keys.includes("Win")}
          onPressedChange={(pressed) => {
            setRecording(false)
            endInputCapture()
            const next = keys.filter((key) => key !== "Win")
            if (pressed) next.unshift("Win")
            onChange(next)
          }}
          aria-label="包含 Win 键"
          title="Win"
        >
          <CommandIcon aria-hidden="true" />
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
