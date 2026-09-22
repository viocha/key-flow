import type { CSSProperties } from "react"

const directionAngles: Record<string, number> = {
  R: 0,
  DR: 45,
  D: 90,
  DL: 135,
  L: 180,
  UL: 225,
  U: 270,
  UR: 315,
}

export function GestureEndpointArrow({
  x,
  y,
  direction,
  size = 14,
}: {
  x: number
  y: number
  direction?: string | null
  size?: number
}) {
  const angle = direction ? directionAngles[direction] : undefined
  if (angle === undefined) return null

  const radians = (angle * Math.PI) / 180
  const tipOffset = (size * 3) / 14
  const style: CSSProperties = {
    left: `calc(${x * 100}% - ${Math.cos(radians) * tipOffset}px)`,
    top: `calc(${y * 100}% - ${Math.sin(radians) * tipOffset}px)`,
    width: size,
    height: size,
    marginLeft: -size / 2,
    marginTop: -size / 2,
    transform: `rotate(${angle}deg)`,
  }

  return (
    <svg
      viewBox="0 0 14 14"
      className="pointer-events-none absolute overflow-visible text-foreground"
      style={style}
      aria-hidden="true"
    >
      <path
        d="M 3.5 2.5 L 10 7 L 3.5 11.5"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
