package main

import (
	"errors"
	"math"
	"strings"
)

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
	result := make([]string, 0, 8)
	anchor := points[0]
	for i := 1; i < len(points); i++ {
		current := points[i]
		dx := current.X - anchor.X
		dy := current.Y - anchor.Y
		if math.Hypot(dx, dy) < segmentDistance {
			continue
		}
		direction := quantizeGestureDirection(dx, dy)
		if direction != "" && (len(result) == 0 || result[len(result)-1] != direction) {
			result = append(result, direction)
		}
		anchor = current
	}
	return simplifyGestureDirections(result)
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
	// 直向区域给 60°，对角区域给 30°。鼠标手势通常更接近
	// “往下/往右”的意图，这比八等分更能容忍自然的斜偏和圆角。
	switch {
	case angle >= 330 || angle < 30:
		return "R"
	case angle < 60:
		return "UR"
	case angle < 120:
		return "U"
	case angle < 150:
		return "UL"
	case angle < 210:
		return "L"
	case angle < 240:
		return "DL"
	case angle < 300:
		return "D"
	default:
		return "DR"
	}
}
