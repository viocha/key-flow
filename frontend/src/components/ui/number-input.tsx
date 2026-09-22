import * as React from "react"

import { Input } from "@/components/ui/input"

type NumberInputProps = Omit<
  React.ComponentProps<typeof Input>,
  "type" | "value" | "defaultValue" | "onChange" | "min" | "max"
> & {
  value: number
  min?: number
  max?: number
  onValueChange: (value: number) => void
}

function NumberInput({
  value,
  min,
  max,
  onValueChange,
  onFocus,
  onBlur,
  onKeyDown,
  ...props
}: NumberInputProps) {
  const [editing, setEditing] = React.useState(false)
  const [draft, setDraft] = React.useState(String(value))

  React.useEffect(() => {
    if (!editing) {
      setDraft(String(value))
    }
  }, [editing, value])

  const commit = () => {
    const parsed = Number(draft)
    if (draft.trim() === "" || !Number.isFinite(parsed)) {
      setDraft(String(value))
      return
    }

    const next = Math.max(min ?? -Infinity, Math.min(max ?? Infinity, parsed))
    setDraft(String(next))
    if (next !== value) {
      onValueChange(next)
    }
  }

  return (
    <Input
      {...props}
      type="number"
      min={min}
      max={max}
      value={draft}
      onChange={(event) => setDraft(event.target.value)}
      onFocus={(event) => {
        setEditing(true)
        onFocus?.(event)
      }}
      onBlur={(event) => {
        commit()
        setEditing(false)
        onBlur?.(event)
      }}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.currentTarget.blur()
        }
        onKeyDown?.(event)
      }}
    />
  )
}

export { NumberInput }
