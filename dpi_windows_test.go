//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"testing"
)

func TestGestureLogicalPixelsAcrossScaling(t *testing.T) {
	data, err := os.ReadFile("testdata/gesture-recognition.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name     string      `json:"name"`
		Distance float64     `json:"distance"`
		Points   [][]float64 `json:"points"`
		Want     []string    `json:"want"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, scale := range []float64{1, 1.25, 1.5, 2, 2.5, 3, 4} {
		t.Run(fmt.Sprintf("缩放%.0f%%", scale*100), func(t *testing.T) {
			for _, fixture := range fixtures {
				gesture := &activeGesture{}
				for _, sample := range fixture.Points {
					gesture.appendSample(point{physicalPixels(sample[0], scale), physicalPixels(sample[1], scale)}, scale)
				}
				got := gestureDirectionsFromPoints(gesture.logicalPoints, fixture.Distance)
				if !slices.Equal(got, fixture.Want) {
					t.Errorf("%s: 方向 = %v，期望 %v", fixture.Name, got, fixture.Want)
				}
			}
		})
	}
}

func TestGestureMixedMonitorScaling(t *testing.T) {
	gesture := &activeGesture{}
	gesture.appendSample(point{-30, 0}, 1)
	gesture.appendSample(point{0, 0}, 1)
	gesture.appendSample(point{60, 0}, 2)
	gesture.appendSample(point{60, 90}, 3)
	last := gesture.logicalPoints[len(gesture.logicalPoints)-1]
	if last.X != 60 || last.Y != 30 {
		t.Fatalf("跨显示器逻辑位移 = %+v，期望 (60, 30)", last)
	}
	if got := gestureDirectionsFromPoints(gesture.logicalPoints, 28); !slices.Equal(got, []string{"R", "D"}) {
		t.Fatalf("跨显示器方向 = %v", got)
	}
	if got := gesture.points[len(gesture.points)-1]; got.X != 60 || got.Y != 90 {
		t.Fatalf("绘制坐标应保持物理坐标: %+v", got)
	}
}

func TestGestureSamplingAndDragThresholdUseLogicalPixels(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 2, 3, 4} {
		gesture := &activeGesture{}
		gesture.appendSample(point{}, scale)
		if gesture.appendSample(point{X: physicalPixels(1, scale)}, scale) {
			t.Errorf("缩放 %v：不足 2 px 不应采样", scale)
		}
		if !gesture.appendSample(point{X: physicalPixels(3, scale)}, scale) {
			t.Errorf("缩放 %v：超过 2 px 应采样", scale)
		}
		if mouseDragExceeded(point{}, point{X: physicalPixels(10, scale)}, scale) {
			t.Errorf("缩放 %v：10 px 不应开始右键拖拽", scale)
		}
		if !mouseDragExceeded(point{}, point{Y: physicalPixels(11, scale)}, scale) {
			t.Errorf("缩放 %v：超过 10 px 应开始右键拖拽", scale)
		}
	}
}

func TestEdgeLogicalPixelsAcrossScaling(t *testing.T) {
	rules := map[string]compiledEdgeWheel{
		"bottom:wheelVertical":     {zone: "bottom", marginPX: 4, endMarginPX: 100},
		"left:wheelVertical":       {zone: "left", marginPX: 4, endMarginPX: 100},
		"bottomLeft:wheelVertical": {zone: "bottomLeft", marginPX: 4, cornerExtendPX: 80},
	}
	for _, scale := range []float64{1, 1.25, 1.5, 2, 2.5, 3, 4} {
		bounds := rect{Left: -physicalPixels(1920, scale), Top: -physicalPixels(1080, scale)}
		for _, sample := range []struct {
			x, y float64
			want string
		}{
			{80, 0, "bottomLeft"}, {0, 80, "bottomLeft"},
			{90, 0, ""}, {0, 90, ""},
			{100, 0, "bottom"}, {0, 100, "left"},
			{200, 4, "bottom"}, {200, 5, ""},
			{4, 200, "left"}, {5, 200, ""}, {50, 50, ""},
		} {
			pt := point{bounds.Left + physicalPixels(sample.x, scale), bounds.Bottom - 1 - physicalPixels(sample.y, scale)}
			got, ok := edgeWheelInBounds(pt, bounds, "wheelVertical", rules, scale)
			if got.zone != sample.want || ok != (sample.want != "") {
				t.Errorf("缩放 %v，距左下角 (%v, %v): 区域 = %q，期望 %q", scale, sample.x, sample.y, got.zone, sample.want)
			}
		}
		if got := physicalPixels(160, scale); math.Abs(float64(got)/scale-160) > 0.5 {
			t.Errorf("窗口最小尺寸换算错误: %d", got)
		}
	}
}
