import { CommandIcon } from "lucide-react"

import { Kbd, KbdGroup } from "@/components/ui/kbd"
import { orderShortcutKeys } from "@/lib/app-model"

export function KeyChord({
  keys,
  empty = "未设置",
}: {
  keys?: string[] | null
  empty?: string
}) {
  if (!keys?.length) {
    return <span className="text-muted-foreground">{empty}</span>
  }

  return (
    <KbdGroup>
      {orderShortcutKeys(keys).map((key) => (
        <Kbd key={key}>
          {key === "Win" ? (
            <>
              <CommandIcon aria-hidden="true" />
              <span className="sr-only">Win</span>
            </>
          ) : (
            displayKey(key)
          )}
        </Kbd>
      ))}
    </KbdGroup>
  )
}

function displayKey(key: string) {
  switch (key) {
    case "ArrowLeft":
      return "←"
    case "ArrowRight":
      return "→"
    case "ArrowUp":
      return "↑"
    case "ArrowDown":
      return "↓"
    case "PageUp":
      return "PgUp"
    case "PageDown":
      return "PgDn"
    case "Escape":
      return "Esc"
    default:
      return key
  }
}
