export type GestureSample = { x: number; y: number }

// 与 gesture.go 使用相同的空间容差、有效距离和方向区域。
export function directionsFromPoints(
  points: readonly GestureSample[],
  segmentDistance: number,
  pixelRatio: number,
) {
  if (points.length < 2) return []
  // 采样坐标是 CSS 像素，配置距离是屏幕像素，空间容差也按同一比例换算。
  segmentDistance = Math.max(1, segmentDistance) / pixelRatio
  const vertices = simplifyPoints(points, segmentDistance / 2)
  const directions: string[] = []
  let anchor = vertices[0]
  for (let index = 1; index < vertices.length; index += 1) {
    const dx = vertices[index].x - anchor.x
    const dy = vertices[index].y - anchor.y
    const direction = quantizeDirection(dx, dy)
    // 斜向多积累 40% 的净位移；确认前保留锚点，后续移动可修正短暂斜偏。
    const confirmationDistance = isDiagonal(direction) ? segmentDistance * 1.4 : segmentDistance
    if (Math.hypot(dx, dy) < confirmationDistance) continue
    if (directions[directions.length - 1] !== direction) directions.push(direction)
    anchor = vertices[index]
  }
  return simplifyDirections(directions)
}

function simplifyPoints(points: readonly GestureSample[], tolerance: number) {
  if (points.length < 3) return points
  const keep = new Set([0, points.length - 1])
  const ranges: [number, number][] = [[0, points.length - 1]]
  while (ranges.length) {
    const [first, last] = ranges.pop()!
    const start = points[first]
    const end = points[last]
    const dx = end.x - start.x
    const dy = end.y - start.y
    const lengthSquared = dx * dx + dy * dy
    let farthest = -1
    let maxDistanceSquared = tolerance * tolerance
    for (let index = first + 1; index < last; index += 1) {
      const point = points[index]
      const projection = lengthSquared > 0
        ? Math.max(0, Math.min(1,
          ((point.x - start.x) * dx + (point.y - start.y) * dy) / lengthSquared))
        : 0
      const x = point.x - start.x - projection * dx
      const y = point.y - start.y - projection * dy
      const distanceSquared = x * x + y * y
      if (distanceSquared > maxDistanceSquared) {
        farthest = index
        maxDistanceSquared = distanceSquared
      }
    }
    if (farthest >= 0) {
      keep.add(farthest)
      ranges.push([first, farthest], [farthest, last])
    }
  }
  return points.filter((_, index) => keep.has(index))
}

function simplifyDirections(directions: string[]) {
  let result = [...directions]
  while (true) {
    result = result.filter(
      (direction, index) => index === 0 || direction !== result[index - 1],
    )
    let changed = false
    for (let index = 1; index + 1 < result.length; index += 1) {
      const previous = result[index - 1]
      const current = result[index]
      const next = result[index + 1]
      if (previous === next && areAdjacentDirections(previous, current)) {
        result.splice(index, 1)
        changed = true
        break
      }
      if (isCardinal(previous) && isDiagonal(current) && isCardinal(next)) {
        const pair = previous + "," + current + "," + next
        if (
          pair === "D,DR,R" || pair === "R,DR,D" ||
          pair === "D,DL,L" || pair === "L,DL,D" ||
          pair === "U,UR,R" || pair === "R,UR,U" ||
          pair === "U,UL,L" || pair === "L,UL,U"
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
  const difference = Math.abs(order.indexOf(a) - order.indexOf(b))
  return difference === 1 || difference === 7
}

function quantizeDirection(dx: number, dy: number) {
  let angle = (Math.atan2(-dy, dx) * 180) / Math.PI
  if (angle < 0) angle += 360
  if (angle >= 325 || angle < 35) return "R"
  if (angle < 55) return "UR"
  if (angle < 125) return "U"
  if (angle < 145) return "UL"
  if (angle < 215) return "L"
  if (angle < 235) return "DL"
  if (angle < 305) return "D"
  return "DR"
}
