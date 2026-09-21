import { Kbd, KbdGroup } from "@/components/ui/kbd"

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
      {keys.map((key) => (
        <Kbd key={key}>{displayKey(key)}</Kbd>
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
