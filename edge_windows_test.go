//go:build windows

package main

import "testing"

func TestEdgeEndMargin(t *testing.T) {
	bounds := rect{Left: -1920, Top: -1080, Right: 0, Bottom: 0}
	for _, tc := range []struct {
		zone                                   string
		nearStart, start, middle, end, nearEnd point
	}{
		{"top", point{-1821, -1080}, point{-1820, -1080}, point{-960, -1080}, point{-101, -1080}, point{-100, -1080}},
		{"bottom", point{-1821, -1}, point{-1820, -1}, point{-960, -1}, point{-101, -1}, point{-100, -1}},
		{"left", point{-1920, -981}, point{-1920, -980}, point{-1920, -540}, point{-1920, -101}, point{-1920, -100}},
		{"right", point{-1, -981}, point{-1, -980}, point{-1, -540}, point{-1, -101}, point{-1, -100}},
	} {
		t.Run(tc.zone, func(t *testing.T) {
			for _, pt := range []point{tc.nearStart, tc.nearEnd} {
				if pointInEdgeBounds(pt, bounds, tc.zone, 4, 100, 0) {
					t.Errorf("两端留空区域不应触发: %+v", pt)
				}
				if !pointInEdgeBounds(pt, bounds, tc.zone, 4, 0, 0) {
					t.Errorf("间距为 0 时应触发: %+v", pt)
				}
			}
			for _, pt := range []point{tc.start, tc.middle, tc.end} {
				if !pointInEdgeBounds(pt, bounds, tc.zone, 4, 100, 0) {
					t.Errorf("达到两端间距时应触发: %+v", pt)
				}
			}
			if pointInEdgeBounds(tc.middle, bounds, tc.zone, 4, 1000, 0) {
				t.Error("间距占满整条边时不应触发")
			}
		})
	}
}

func TestEdgeCornersAndThickness(t *testing.T) {
	bounds := rect{Right: 1920, Bottom: 1080}
	for zone, pt := range map[string]point{
		"topLeft": {0, 0}, "topRight": {1919, 0},
		"bottomLeft": {0, 1079}, "bottomRight": {1919, 1079},
	} {
		if !pointInEdgeBounds(pt, bounds, zone, 4, 1000, 0) {
			t.Errorf("角落规则不应受两端间距影响: %s", zone)
		}
	}
	for _, pt := range []point{{0, 1079}, {1919, 1079}, {960, 1074}, {-1, 1079}, {1920, 1079}, {960, 1080}} {
		if pointInEdgeBounds(pt, bounds, "bottom", 4, 100, 0) {
			t.Errorf("角落、触发厚度或屏幕外不应触发下边缘: %+v", pt)
		}
	}
	if !pointInEdgeBounds(point{960, 1075}, bounds, "bottom", 4, 100, 0) {
		t.Error("两端间距不应影响触发厚度")
	}
}

func TestCornerExtensionFormsLShape(t *testing.T) {
	bounds := rect{Left: -1920, Top: -1080, Right: 0, Bottom: 0}
	for _, zone := range []string{"topLeft", "topRight", "bottomLeft", "bottomRight"} {
		t.Run(zone, func(t *testing.T) {
			for _, sample := range []struct {
				dx, dy int32
				want   bool
			}{
				{0, 0, true}, {100, 0, true}, {100, 4, true},
				{0, 100, true}, {4, 100, true},
				{101, 0, false}, {0, 101, false},
				{100, 5, false}, {5, 100, false}, {50, 50, false},
			} {
				pt := point{bounds.Left + sample.dx, bounds.Top + sample.dy}
				if zone == "topRight" || zone == "bottomRight" {
					pt.X = bounds.Right - 1 - sample.dx
				}
				if zone == "bottomLeft" || zone == "bottomRight" {
					pt.Y = bounds.Bottom - 1 - sample.dy
				}
				if got := pointInEdgeBounds(pt, bounds, zone, 4, 0, 100); got != sample.want {
					t.Errorf("距角落 (%d, %d): 触发 = %t，期望 %t", sample.dx, sample.dy, got, sample.want)
				}
			}
		})
	}
}

func TestCornerRulesTakePriorityOverStraightEdges(t *testing.T) {
	bounds := rect{Right: 1920, Bottom: 1080}
	rules := map[string]compiledEdgeWheel{
		"bottomLeft:wheelVertical": {zone: "bottomLeft", marginPX: 4, cornerExtendPX: 100},
		"bottom:wheelVertical":     {zone: "bottom", marginPX: 4},
		"left:wheelVertical":       {zone: "left", marginPX: 4},
		"bottom:wheelHorizontal":   {zone: "bottom", marginPX: 4},
	}
	for _, sample := range []struct {
		pt          point
		input, want string
	}{
		{point{100, 1079}, "wheelVertical", "bottomLeft"},
		{point{0, 979}, "wheelVertical", "bottomLeft"},
		{point{101, 1079}, "wheelVertical", "bottom"},
		{point{0, 978}, "wheelVertical", "left"},
		{point{100, 1079}, "wheelHorizontal", "bottom"},
		{point{50, 1029}, "wheelVertical", ""},
	} {
		got, ok := edgeWheelInBounds(sample.pt, bounds, sample.input, rules)
		if got.zone != sample.want || ok != (sample.want != "") {
			t.Errorf("点 %+v，输入 %s: 区域 = %q，期望 %q", sample.pt, sample.input, got.zone, sample.want)
		}
	}
}
