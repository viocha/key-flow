package main

import (
	"errors"
	"math"
	"strings"
)

func gestureBindingLabel(binding GestureBinding) string {
	if name := strings.TrimSpace(binding.Name); name != "" {
		return name
	}
	if binding.Action.Type == ActionWindow {
		switch binding.Action.WindowCommand {
		case "close":
			return "关闭窗口"
		case "terminateProcess":
			return "终止进程"
		case "previousWindow":
			return "切换上一个窗口"
		case "nextWindow":
			return "切换下一个窗口"
		case "minimize":
			return "最小化窗口"
		case "restoreMinimized":
			return "恢复最小化窗口"
		case "maximize":
			return "最大化窗口"
		case "restore":
			return "恢复窗口"
		case "toggleMaximize":
			return "切换最大化"
		case "toggleVerticalMaximize":
			return "切换垂直最大化"
		case "opacity":
			return "调整窗口透明度"
		case "scale":
			return "窗口中心缩放"
		case "alwaysOnTop":
			return "切换置顶"
		}
	}
	if name := strings.TrimSpace(binding.Action.Name); name != "" {
		return name
	}
	return "手势"
}

var gestureDirections = map[string]struct{}{
	"U": {}, "D": {}, "L": {}, "R": {},
	"UL": {}, "UR": {}, "DL": {}, "DR": {},
}

func gestureKey(directions []string) (string, error) {
	if len(directions) == 0 {
		return "", errors.New("手势路径不能为空")
	}
	clean := make([]string, 0, len(directions))
	for _, direction := range directions {
		direction = strings.ToUpper(strings.TrimSpace(direction))
		if _, ok := gestureDirections[direction]; !ok {
			return "", errors.New("未知手势方向: " + direction)
		}
		if len(clean) == 0 || clean[len(clean)-1] != direction {
			clean = append(clean, direction)
		}
	}
	clean = simplifyGestureDirections(clean)
	if len(clean) == 0 {
		return "", errors.New("手势路径不能为空")
	}
	return strings.Join(clean, ","), nil
}

func gestureDirectionsFromPoints(points []GesturePoint, segmentDistance float64) []string {
	if len(points) < 2 {
		return nil
	}
	if segmentDistance < 1 {
		segmentDistance = 1
	}
	// 先保留有意义的拐点，直线附近半个分段距离内的偏移视为手抖。
	// 使用整条线段的净位移，避免在转弯起步时放大局部的斜向抖动。
	points = simplifyGesturePoints(points, segmentDistance/2)
	result := make([]string, 0, 8)
	anchor := points[0]
	for i := 1; i < len(points); i++ {
		current := points[i]
		dx := current.X - anchor.X
		dy := current.Y - anchor.Y
		direction := quantizeGestureDirection(dx, dy)
		confirmationDistance := segmentDistance
		// 斜向需要多积累 40% 的净位移，避免转弯起步的抖动过早确认方向。
		if isDiagonalGestureDirection(direction) {
			confirmationDistance *= 1.4
		}
		if math.Hypot(dx, dy) < confirmationDistance {
			continue
		}
		if direction != "" && (len(result) == 0 || result[len(result)-1] != direction) {
			result = append(result, direction)
		}
		anchor = current
	}
	return simplifyGestureDirections(result)
}

// 使用到有限线段的距离，保留折返及闭合路径中的有效拐点。
func simplifyGesturePoints(points []GesturePoint, tolerance float64) []GesturePoint {
	if len(points) < 3 {
		return points
	}
	keep := make([]bool, len(points))
	keep[0], keep[len(points)-1] = true, true
	ranges := [][2]int{{0, len(points) - 1}}
	for len(ranges) > 0 {
		span := ranges[len(ranges)-1]
		ranges = ranges[:len(ranges)-1]
		start, end := points[span[0]], points[span[1]]
		dx, dy := end.X-start.X, end.Y-start.Y
		lengthSquared := dx*dx + dy*dy
		farthest, maxDistanceSquared := -1, tolerance*tolerance
		for i := span[0] + 1; i < span[1]; i++ {
			point := points[i]
			projection := 0.0
			if lengthSquared > 0 {
				projection = math.Max(0, math.Min(1,
					((point.X-start.X)*dx+(point.Y-start.Y)*dy)/lengthSquared))
			}
			x, y := point.X-start.X-projection*dx, point.Y-start.Y-projection*dy
			if distanceSquared := x*x + y*y; distanceSquared > maxDistanceSquared {
				farthest, maxDistanceSquared = i, distanceSquared
			}
		}
		if farthest >= 0 {
			keep[farthest] = true
			ranges = append(ranges, [2]int{span[0], farthest}, [2]int{farthest, span[1]})
		}
	}
	result := make([]GesturePoint, 0, len(points))
	for i, point := range points {
		if keep[i] {
			result = append(result, point)
		}
	}
	return result
}

func simplifyGestureDirections(directions []string) []string {
	if len(directions) < 2 {
		return directions
	}

	result := append([]string(nil), directions...)
	for {
		changed := false
		clean := make([]string, 0, len(result))
		for _, direction := range result {
			if len(clean) == 0 || clean[len(clean)-1] != direction {
				clean = append(clean, direction)
			}
		}
		result = clean

		for i := 1; i+1 < len(result); i++ {
			previous := result[i-1]
			current := result[i]
			next := result[i+1]

			// A-B-A 通常是手抖产生的一个短暂 45° 偏移。
			if previous == next && adjacentGestureDirections(previous, current) {
				result = append(result[:i], result[i+1:]...)
				changed = true
				break
			}

			// 直角转弯很容易在圆角处短暂经过对角方向。
			// 例如 D,DR,R 与用户意图 D,R 应视为同一个手势。
			if isCardinalGestureDirection(previous) &&
				isDiagonalGestureDirection(current) &&
				isCardinalGestureDirection(next) &&
				diagonalBetweenGestureDirections(previous, current, next) {
				result = append(result[:i], result[i+1:]...)
				changed = true
				break
			}
		}

		if !changed {
			return result
		}
	}
}

func isCardinalGestureDirection(direction string) bool {
	switch direction {
	case "U", "D", "L", "R":
		return true
	default:
		return false
	}
}

func isDiagonalGestureDirection(direction string) bool {
	switch direction {
	case "UL", "UR", "DL", "DR":
		return true
	default:
		return false
	}
}

func adjacentGestureDirections(a, b string) bool {
	angles := map[string]int{
		"R": 0, "UR": 1, "U": 2, "UL": 3,
		"L": 4, "DL": 5, "D": 6, "DR": 7,
	}
	aa, okA := angles[a]
	bb, okB := angles[b]
	if !okA || !okB {
		return false
	}
	diff := aa - bb
	if diff < 0 {
		diff = -diff
	}
	return diff == 1 || diff == 7
}

func diagonalBetweenGestureDirections(previous, diagonal, next string) bool {
	switch diagonal {
	case "UR":
		return (previous == "U" && next == "R") || (previous == "R" && next == "U")
	case "UL":
		return (previous == "U" && next == "L") || (previous == "L" && next == "U")
	case "DL":
		return (previous == "D" && next == "L") || (previous == "L" && next == "D")
	case "DR":
		return (previous == "D" && next == "R") || (previous == "R" && next == "D")
	default:
		return false
	}
}

func quantizeGestureDirection(dx, dy float64) string {
	if dx == 0 && dy == 0 {
		return ""
	}
	angle := math.Atan2(-dy, dx) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	// 直向区域给 70°，对角区域给 20°，容忍直线的自然斜偏并保留 45° 手势。
	switch {
	case angle >= 325 || angle < 35:
		return "R"
	case angle < 55:
		return "UR"
	case angle < 125:
		return "U"
	case angle < 145:
		return "UL"
	case angle < 215:
		return "L"
	case angle < 235:
		return "DL"
	case angle < 305:
		return "D"
	default:
		return "DR"
	}
}
