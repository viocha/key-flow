import { useMemo, useRef, useState } from "react"
import type { GesturePoint } from "../../bindings/key-flow"

import { GestureEndpointArrow } from "@/components/gesture-endpoint-arrow"
import { useInputCapture } from "@/hooks/use-input-capture"
import { gestureDirectionLabel } from "@/lib/app-model"

type RawPoint = {
  x: number
  y: number
}

export function GestureRecorder({
  preview,
  directions,
  onChange,
}: {
  preview?: GesturePoint[] | null
  directions?: string[] | null
  onChange: (preview: GesturePoint[], directions: string[]) => void
}) {
  const [drawing, setDrawing] = useState<{
    pointerId: number
    points: RawPoint[]
  } | null>(null)
  const boxRef = useRef<HTMLDivElement>(null)
  const { begin: beginInputCapture, end: endInputCapture } = useInputCapture()

  const visiblePreview = useMemo(() => {
    if (!drawing || !boxRef.current) return preview ?? []
    const rect = boxRef.current.getBoundingClientRect()
    return drawing.points.map((point) => ({
      x: clamp01((point.x - rect.left) / Math.max(1, rect.width)),
      y: clamp01((point.y - rect.top) / Math.max(1, rect.height)),
    }))
  }, [drawing, preview])
  const visibleDirections = drawing
    ? directionsFromPoints(drawing.points)
    : (directions ?? [])
  const endpoint = visiblePreview[visiblePreview.length - 1]
  const endpointDirection = visibleDirections[visibleDirections.length - 1]

  const finish = (points: RawPoint[]) => {
    const element = boxRef.current
    if (!element || points.length < 2) {
      setDrawing(null)
      return
    }
    const rect = element.getBoundingClientRect()
    const normalized = points.map((point) => ({
      x: clamp01((point.x - rect.left) / Math.max(1, rect.width)),
      y: clamp01((point.y - rect.top) / Math.max(1, rect.height)),
    }))
    onChange(normalized, directionsFromPoints(points))
    setDrawing(null)
  }

  return (
    <div className="flex flex-col gap-2">
      <div
        ref={boxRef}
        className="relative h-48 overflow-hidden rounded-lg border bg-muted/30 select-none"
        style={{ touchAction: "none" }}
        onContextMenu={(event) => event.preventDefault()}
        onPointerEnter={beginInputCapture}
        onPointerLeave={endInputCapture}
        onPointerDown={(event) => {
          event.preventDefault()
          event.currentTarget.setPointerCapture(event.pointerId)
          setDrawing({
            pointerId: event.pointerId,
            points: [{ x: event.clientX, y: event.clientY }],
          })
        }}
        onPointerMove={(event) => {
          setDrawing((current) => {
            if (!current || current.pointerId !== event.pointerId) return current
            const last = current.points[current.points.length - 1]
            if (Math.hypot(event.clientX - last.x, event.clientY - last.y) < 2) {
              return current
            }
            return {
              ...current,
              points: [
                ...current.points,
                { x: event.clientX, y: event.clientY },
              ],
            }
          })
        }}
        onPointerUp={(event) => {
          setDrawing((current) => {
            if (current?.pointerId === event.pointerId) {
              const points = [
                ...current.points,
                { x: event.clientX, y: event.clientY },
              ]
              queueMicrotask(() => finish(points))
            }
            return current
          })
        }}
        onPointerCancel={() => {
          setDrawing(null)
          endInputCapture()
        }}
      >
        {visiblePreview.length >= 2 ? (
          <>
          <svg
            viewBox="0 0 100 100"
            preserveAspectRatio="none"
            className="absolute inset-0 size-full"
            aria-hidden="true"
          >
            <polyline
              points={visiblePreview
                .map((point) => point.x * 100 + "," + point.y * 100)
                .join(" ")}
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
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
              size={16}
            />
          )}
          </>
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
            按住鼠标拖动录制
          </div>
        )}
      </div>
      <div className="min-h-5 text-xs text-muted-foreground">
        {directions?.length
          ? gestureDirectionLabel(directions)
          : drawing
            ? "正在录制"
            : "尚未录制"}
      </div>
    </div>
  )
}

function clamp01(value: number) {
  return Math.max(0, Math.min(1, value))
}

function directionsFromPoints(points: RawPoint[]) {
  if (points.length < 2) return []

  const segments: { direction: string; distance: number }[] = []
  let anchor = points[0]
  const threshold = 24
  const terminalTurnThreshold = 36

  for (const point of points.slice(1)) {
    const dx = point.x - anchor.x
    const dy = point.y - anchor.y
    const distance = Math.hypot(dx, dy)
    if (distance < threshold) continue

    const direction = quantizeDirection(dx, dy)
    const last = segments[segments.length - 1]
    if (last?.direction === direction) {
      last.distance += distance
    } else {
      segments.push({ direction, distance })
    }
    anchor = point
  }

  // 收手时常会产生一小段横向或斜向抖动。方向发生变化后需要再持续
  // 一段距离才把它视为有效的末段，避免污染保存结果和终点箭头。
  if (
    segments.length > 1 &&
    segments[segments.length - 1].distance < terminalTurnThreshold
  ) {
    segments.pop()
  }

  return simplifyDirections(segments.map((segment) => segment.direction))
}

function simplifyDirections(directions: string[]) {
  let result = [...directions]

  while (true) {
    const clean = result.filter(
      (direction, index) => index === 0 || direction !== result[index - 1],
    )
    result = clean

    let changed = false
    for (let index = 1; index + 1 < result.length; index += 1) {
      const previous = result[index - 1]
      const current = result[index]
      const next = result[index + 1]

      if (
        previous === next &&
        areAdjacentDirections(previous, current)
      ) {
        result.splice(index, 1)
        changed = true
        break
      }

      if (isCardinal(previous) && isDiagonal(current) && isCardinal(next)) {
        const pair = previous + "," + current + "," + next
        if (
          pair === "D,DR,R" ||
          pair === "R,DR,D" ||
          pair === "D,DL,L" ||
          pair === "L,DL,D" ||
          pair === "U,UR,R" ||
          pair === "R,UR,U" ||
          pair === "U,UL,L" ||
          pair === "L,UL,U"
        ) {
          result.splice(index, 1)
          changed = true
          break
        }
      }
    }

    if (!changed) return result
  }
}

function isCardinal(direction: string) {
  return direction === "U" || direction === "D" || direction === "L" || direction === "R"
}

function isDiagonal(direction: string) {
  return direction === "UL" || direction === "UR" || direction === "DL" || direction === "DR"
}

function areAdjacentDirections(a: string, b: string) {
  const order = ["R", "UR", "U", "UL", "L", "DL", "D", "DR"]
  const aIndex = order.indexOf(a)
  const bIndex = order.indexOf(b)
  if (aIndex < 0 || bIndex < 0) return false
  const difference = Math.abs(aIndex - bIndex)
  return difference === 1 || difference === 7
}

function quantizeDirection(dx: number, dy: number) {
  let angle = (Math.atan2(-dy, dx) * 180) / Math.PI
  if (angle < 0) angle += 360

  if (angle >= 330 || angle < 30) return "R"
  if (angle < 60) return "UR"
  if (angle < 120) return "U"
  if (angle < 150) return "UL"
  if (angle < 210) return "L"
  if (angle < 240) return "DL"
  if (angle < 300) return "D"
  return "DR"
}
