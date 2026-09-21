package main

import "testing"

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
					Name:       "关闭窗口",
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
			MarginPX:   4,
			Adjustment: EdgeAdjustmentVolume,
			Step:       1,
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

	edge, ok := compiled.edgeWheels["bottomRight"]
	if !ok {
		t.Fatal("bottom-right edge wheel was not compiled")
	}
	if edge.up.Type != ActionVolume || edge.up.Delta != 1 {
		t.Fatalf("edge up action = %#v", edge.up)
	}
}

func TestCompileConfigRecentAppsEdge(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{
		{
			ID:         "recent-apps",
			Enabled:    true,
			Zone:       "right",
			MarginPX:   4,
			Adjustment: EdgeAdjustmentRecentApps,
			Step:       1,
		},
	}

	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	edge, ok := compiled.edgeWheels["right"]
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

func TestCompileConfigBidirectionalKeysEdge(t *testing.T) {
	config := defaultConfig()
	config.Profiles[0].EdgeWheels = []EdgeWheelRule{
		{
			ID:         "tabs",
			Enabled:    true,
			Zone:       "left",
			MarginPX:   4,
			Adjustment: EdgeAdjustmentBidirectionalKeys,
			UpKeys:     []string{"Ctrl", "Shift", "Tab"},
			DownKeys:   []string{"Ctrl", "Tab"},
		},
	}

	if err := validateConfig(config); err != nil {
		t.Fatal(err)
	}
	compiled, err := compileConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	edge := compiled.edgeWheels["left"]
	if got := edge.up.Keys; len(got) != 3 || got[0] != "Ctrl" || got[1] != "Shift" || got[2] != "Tab" {
		t.Fatalf("bidirectional wheel-up keys = %#v", got)
	}
	if got := edge.down.Keys; len(got) != 2 || got[0] != "Ctrl" || got[1] != "Tab" {
		t.Fatalf("bidirectional wheel-down keys = %#v", got)
	}
}
