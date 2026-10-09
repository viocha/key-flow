package main

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestGestureRecognitionFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/gesture-recognition.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name     string      `json:"name"`
		Distance float64     `json:"distance"`
		Points   [][]float64 `json:"points"`
		Want     []string    `json:"want"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			points := make([]GesturePoint, len(tc.Points))
			for i, point := range tc.Points {
				points[i] = GesturePoint{X: point[0], Y: point[1]}
			}
			if got := gestureDirectionsFromPoints(points, tc.Distance); !slices.Equal(got, tc.Want) {
				t.Fatalf("手势方向 = %v，期望 %v", got, tc.Want)
			}
		})
	}
}

func TestGestureDirectionsFromPoints(t *testing.T) {
	points := []GesturePoint{
		{X: 100, Y: 100},
		{X: 100, Y: 145},
		{X: 118, Y: 165},
		{X: 145, Y: 165},
	}

	got := gestureDirectionsFromPoints(points, 20)
	if len(got) != 2 || got[0] != "D" || got[1] != "R" {
		t.Fatalf("gesture directions = %#v, want [D R]", got)
	}
}

func TestGestureDirectionsPreferCardinalWithNaturalSkew(t *testing.T) {
	points := []GesturePoint{
		{X: 100, Y: 100},
		{X: 118, Y: 150},
		{X: 150, Y: 156},
	}

	got := gestureDirectionsFromPoints(points, 20)
	if len(got) != 2 || got[0] != "D" || got[1] != "R" {
		t.Fatalf("gesture directions = %#v, want [D R]", got)
	}
}

func TestGestureKeySimplifiesRoundedCorner(t *testing.T) {
	got, err := gestureKey([]string{"D", "DR", "R"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "D,R" {
		t.Fatalf("gesture key = %q, want D,R", got)
	}
}

func TestGestureKeyRemovesAdjacentJitter(t *testing.T) {
	got, err := gestureKey([]string{"D", "DR", "D"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "D" {
		t.Fatalf("gesture key = %q, want D", got)
	}
}

func TestQuantizeGestureDirectionKeepsTrueDiagonal(t *testing.T) {
	if got := quantizeGestureDirection(40, 40); got != "DR" {
		t.Fatalf("direction = %q, want DR", got)
	}
}

func TestCompileConfigIncludesGesturesAndEdgeWheel(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheelMarginPX = 7
	config.Profiles[0].Gestures = []GestureGroup{
		{
			ID:                "middle",
			Enabled:           true,
			Trigger:           "MouseMiddle",
			ShowTrail:         true,
			StartDistancePX:   18,
			SegmentDistancePX: 28,
			Bindings: []GestureBinding{
				{
					ID:         "close",
					Enabled:    true,
					Directions: []string{"D", "R"},
					Preview: []GesturePoint{
						{X: 0.5, Y: 0.2},
						{X: 0.5, Y: 0.8},
						{X: 0.8, Y: 0.8},
					},
					Action: Action{
						Type:          ActionWindow,
						WindowCommand: "close",
					},
				},
			},
		},
	}
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{
		{
			ID:         "volume",
			Enabled:    true,
			Zone:       "bottomRight",
			Input:      "wheelVertical",
			Adjustment: EdgeAdjustmentVolume,
			Step:       1,
			WheelSteps: 1,
		},
	}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	middle, ok := compiled.gestures["MouseMiddle"]
	if !ok || !middle.enabled {
		t.Fatal("mouse middle gesture group was not compiled")
	}
	if gesture, ok := middle.bindings["D,R"]; !ok || gesture.name != "关闭窗口" {
		t.Fatalf("compiled gesture = %#v, found=%v", gesture, ok)
	}

	edge, ok := compiled.edgeWheels["bottomRight:wheelVertical"]
	if !ok {
		t.Fatal("bottom-right edge wheel was not compiled")
	}
	if edge.up.Type != ActionVolume || edge.up.Delta != 1 {
		t.Fatalf("edge up action = %#v", edge.up)
	}
	if edge.marginPX != 7 {
		t.Fatalf("edge margin = %d, want 7", edge.marginPX)
	}
	if edge.wheelSteps != 1 {
		t.Fatalf("edge wheel steps = %d, want 1", edge.wheelSteps)
	}
}

func TestCompileConfigRecentAppsEdge(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{
		{
			ID:         "recent-apps",
			Enabled:    true,
			Zone:       "right",
			Input:      "wheelVertical",
			Adjustment: EdgeAdjustmentRecentApps,
			Step:       1,
			WheelSteps: 1,
		},
	}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	edge, ok := compiled.edgeWheels["right:wheelVertical"]
	if !ok || !edge.recentApps {
		t.Fatalf("recent apps edge = %#v, found=%v", edge, ok)
	}
	if got := edge.up.Keys; len(got) != 4 || got[0] != "Ctrl" || got[1] != "Shift" || got[2] != "Alt" || got[3] != "Tab" {
		t.Fatalf("recent apps wheel-up keys = %#v", got)
	}
	if got := edge.down.Keys; len(got) != 3 || got[0] != "Ctrl" || got[1] != "Alt" || got[2] != "Tab" {
		t.Fatalf("recent apps wheel-down keys = %#v", got)
	}
}

func TestCompileConfigDirectAppSwitchEdge(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{{
		ID: "switch-apps", Enabled: true, Zone: "right",
		Input:      "wheelVertical",
		Adjustment: EdgeAdjustmentSwitchApps, WheelSteps: 2,
	}}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	edge := compiled.edgeWheels["right:wheelVertical"]
	if edge.recentApps || edge.wheelSteps != 2 {
		t.Fatalf("直接切换不应启用任务选择点击模式: %#v", edge)
	}
	if edge.up.Type != ActionWindow || edge.up.WindowCommand != "previousWindow" {
		t.Fatalf("向上切换动作 = %#v", edge.up)
	}
	if edge.down.Type != ActionWindow || edge.down.WindowCommand != "nextWindow" {
		t.Fatalf("向下切换动作 = %#v", edge.down)
	}
}

func TestCompileConfigTabSwitchEdge(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{{
		ID: "switch-tabs", Enabled: true, Zone: "right",
		Input:      "wheelVertical",
		Adjustment: EdgeAdjustmentSwitchTabs, WheelSteps: 1,
	}}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	edge := compiled.edgeWheels["right:wheelVertical"]
	if got := edge.up.Keys; len(got) != 3 || got[0] != "Ctrl" || got[1] != "Shift" || got[2] != "Tab" {
		t.Fatalf("上一个标签页动作 = %#v", edge.up)
	}
	if got := edge.down.Keys; len(got) != 2 || got[0] != "Ctrl" || got[1] != "Tab" {
		t.Fatalf("下一个标签页动作 = %#v", edge.down)
	}
}

func TestCompileConfigPageEdge(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{{
		ID: "page", Enabled: true, Zone: "left",
		Input:      "wheelVertical",
		Adjustment: EdgeAdjustmentPage, WheelSteps: 1,
	}}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	edge := compiled.edgeWheels["left:wheelVertical"]
	if got := edge.up.Keys; len(got) != 1 || got[0] != "PageUp" {
		t.Fatalf("page up action = %#v", edge.up)
	}
	if got := edge.down.Keys; len(got) != 1 || got[0] != "PageDown" {
		t.Fatalf("page down action = %#v", edge.down)
	}
}

func TestCompileConfigCustomEdgeActions(t *testing.T) {
	config := defaultConfig()
	first := Action{Type: ActionWindow, WindowCommand: "scale", Delta: 5}
	second := Action{Type: ActionWindow, WindowCommand: "scale", Delta: -5}
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{{
		ID: "custom", Enabled: true, Zone: "left",
		Input:          "wheelHorizontal",
		Adjustment:     EdgeAdjustmentCustom,
		WheelSteps:     1,
		Action:         &first,
		OppositeAction: &second,
	}}

	if err := validateConfig(config); err != nil {
		t.Fatal(err)
	}
	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	edge := compiled.edgeWheels["left:wheelHorizontal"]
	if edge.up.WindowCommand != "scale" || edge.up.Delta != 5 {
		t.Fatalf("first custom action = %#v", edge.up)
	}
	if edge.down.WindowCommand != "scale" || edge.down.Delta != -5 {
		t.Fatalf("second custom action = %#v", edge.down)
	}
}

func TestCompileConfigAllowsBothWheelAxesOnSameEdge(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{
		{
			ID: "vertical", Enabled: true, Zone: "right", Input: "wheelVertical",
			Adjustment: EdgeAdjustmentVolume, Step: 5, WheelSteps: 1,
		},
		{
			ID: "horizontal", Enabled: true, Zone: "right", Input: "wheelHorizontal",
			Adjustment: EdgeAdjustmentBrightness, Step: 5, WheelSteps: 1,
		},
	}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.edgeWheels["right:wheelVertical"].input != "wheelVertical" {
		t.Fatal("vertical edge wheel was not compiled")
	}
	if compiled.edgeWheels["right:wheelHorizontal"].input != "wheelHorizontal" {
		t.Fatal("horizontal edge wheel was not compiled")
	}
}
