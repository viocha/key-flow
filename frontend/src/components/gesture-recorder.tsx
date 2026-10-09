import { useMemo, useRef, useState } from "react"
import type { GesturePoint } from "../../bindings/key-flow"

import { GestureEndpointArrow } from "@/components/gesture-endpoint-arrow"
import { useInputCapture } from "@/hooks/use-input-capture"
import { gestureDirectionLabel } from "@/lib/app-model"
import { directionsFromPoints, type GestureSample } from "@/lib/gesture-recognition"

type Drawing = {
  pointerId: number
  pixelRatio: number
  points: GestureSample[]
}

export function GestureRecorder({
  preview,
  directions,
  segmentDistance,
  onChange,
}: {
  preview?: GesturePoint[] | null
  directions?: string[] | null
  segmentDistance: number
  onChange: (preview: GesturePoint[], directions: string[]) => void
}) {
  const [drawing, setDrawing] = useState<Drawing | null>(null)
  const drawingRef = useRef<Drawing | null>(null)
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
  const visibleDirections = useMemo(() => drawing
    ? directionsFromPoints(drawing.points, segmentDistance, drawing.pixelRatio)
    : (directions ?? []), [drawing, directions, segmentDistance])
  const endpoint = visiblePreview[visiblePreview.length - 1]
  const endpointDirection = visibleDirections[visibleDirections.length - 1]

  const updateDrawing = (next: Drawing | null) => {
    drawingRef.current = next
    setDrawing(next)
  }

  const finish = (points: GestureSample[], pixelRatio: number) => {
    const element = boxRef.current
    if (!element || points.length < 2) {
      updateDrawing(null)
      return
    }
    const rect = element.getBoundingClientRect()
    const normalized = points.map((point) => ({
      x: clamp01((point.x - rect.left) / Math.max(1, rect.width)),
      y: clamp01((point.y - rect.top) / Math.max(1, rect.height)),
    }))
    onChange(normalized, directionsFromPoints(points, segmentDistance, pixelRatio))
    updateDrawing(null)
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
          updateDrawing({
            pointerId: event.pointerId,
            pixelRatio: window.devicePixelRatio,
            points: [{ x: event.clientX, y: event.clientY }],
          })
        }}
        onPointerMove={(event) => {
          const current = drawingRef.current
          if (!current || current.pointerId !== event.pointerId) return
          const last = current.points[current.points.length - 1]
          if (Math.hypot(event.clientX - last.x, event.clientY - last.y) * current.pixelRatio < 2) return
          updateDrawing({
            ...current,
            points: [...current.points, { x: event.clientX, y: event.clientY }],
          })
        }}
        onPointerUp={(event) => {
          const current = drawingRef.current
          if (current?.pointerId !== event.pointerId) return
          finish([...current.points, { x: event.clientX, y: event.clientY }], current.pixelRatio)
        }}
        onPointerCancel={() => {
          updateDrawing(null)
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
        {visibleDirections.length
          ? gestureDirectionLabel(visibleDirections)
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
