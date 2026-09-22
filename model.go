package main

type ActionType string

const (
	ActionKeyStroke  ActionType = "keyStroke"
	ActionText       ActionType = "text"
	ActionLaunch     ActionType = "launch"
	ActionCommand    ActionType = "command"
	ActionSequence   ActionType = "sequence"
	ActionEdit       ActionType = "edit"
	ActionWindow     ActionType = "window"
	ActionVolume     ActionType = "volume"
	ActionBrightness ActionType = "brightness"
)

type Action struct {
	Type              ActionType `json:"type"`
	Name              string     `json:"name,omitempty"`
	Keys              []string   `json:"keys,omitempty"`
	Text              string     `json:"text,omitempty"`
	Target            string     `json:"target,omitempty"`
	Program           string     `json:"program,omitempty"`
	ArgsMode          string     `json:"argsMode,omitempty"`
	CommandLine       string     `json:"commandLine,omitempty"`
	Args              []string   `json:"args,omitempty"`
	WorkingDirectory  string     `json:"workingDirectory,omitempty"`
	Sequence          []Action   `json:"sequence,omitempty"`
	DelayMS           int        `json:"delayMs,omitempty"`
	EditCommand       string     `json:"editCommand,omitempty"`
	WindowCommand     string     `json:"windowCommand,omitempty"`
	WindowBorderColor string     `json:"windowBorderColor,omitempty"`
	WindowBorderSize  int        `json:"windowBorderSize,omitempty"`
	Delta             int        `json:"delta,omitempty"`
}

type HotkeyRule struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Enabled bool     `json:"enabled"`
	Keys    []string `json:"keys"`
	Action  Action   `json:"action"`
}

type LayerBinding struct {
	ID     string   `json:"id"`
	Keys   []string `json:"keys"`
	Action Action   `json:"action"`
}

type MouseShortcutBinding struct {
	Input     string   `json:"input"`
	Modifiers []string `json:"modifiers,omitempty"`
	Action    Action   `json:"action"`
}

type WindowPointerOperations struct {
	Enabled          bool   `json:"enabled"`
	MoveButton       string `json:"moveButton,omitempty"`
	ResizeButton     string `json:"resizeButton,omitempty"`
	ScaleWheel       bool   `json:"scaleWheel"`
	ScaleStepPercent int    `json:"scaleStepPercent"`
}

type WindowManipulationMode string

const (
	WindowManipulationMove   WindowManipulationMode = "move"
	WindowManipulationResize WindowManipulationMode = "resize"
)

type GesturePoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type GestureBinding struct {
	ID         string         `json:"id"`
	Name       string         `json:"name,omitempty"`
	Enabled    bool           `json:"enabled"`
	Directions []string       `json:"directions"`
	Preview    []GesturePoint `json:"preview"`
	Action     Action         `json:"action"`
}

type GestureGroup struct {
	ID                string           `json:"id"`
	Enabled           bool             `json:"enabled"`
	Trigger           string           `json:"trigger"`
	ShowTrail         bool             `json:"showTrail"`
	StartDistancePX   int              `json:"startDistancePx"`
	SegmentDistancePX int              `json:"segmentDistancePx"`
	Bindings          []GestureBinding `json:"bindings"`
}

type TriggerLayer struct {
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	Enabled          bool                    `json:"enabled"`
	Trigger          string                  `json:"trigger"`
	HoldMS           int                     `json:"holdMs"`
	TapBehavior      string                  `json:"tapBehavior"`
	Bindings         []LayerBinding          `json:"bindings"`
	MouseShortcuts   []MouseShortcutBinding  `json:"mouseShortcuts"`
	WindowOperations WindowPointerOperations `json:"windowOperations"`
}

type EdgeAdjustmentType string

const (
	EdgeAdjustmentVolume            EdgeAdjustmentType = "volume"
	EdgeAdjustmentBrightness        EdgeAdjustmentType = "brightness"
	EdgeAdjustmentRecentApps        EdgeAdjustmentType = "recentApps"
	EdgeAdjustmentBidirectionalKeys EdgeAdjustmentType = "bidirectionalKeys"
)

type EdgeWheelRule struct {
	ID         string             `json:"id"`
	Enabled    bool               `json:"enabled"`
	Zone       string             `json:"zone"`
	Adjustment EdgeAdjustmentType `json:"adjustment"`
	Step       int                `json:"step"`
	UpKeys     []string           `json:"upKeys,omitempty"`
	DownKeys   []string           `json:"downKeys,omitempty"`
}

type Profile struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	HotkeysEnabled    bool            `json:"hotkeysEnabled"`
	Hotkeys           []HotkeyRule    `json:"hotkeys"`
	Layers            []TriggerLayer  `json:"layers"`
	Gestures          []GestureGroup  `json:"gestures"`
	EdgeWheelsEnabled bool            `json:"edgeWheelsEnabled"`
	EdgeWheelMarginPX int             `json:"edgeWheelMarginPx"`
	EdgeWheels        []EdgeWheelRule `json:"edgeWheels"`
}

type Settings struct {
	DefaultHoldMS    int `json:"defaultHoldMs"`
	GestureTimeoutMS int `json:"gestureTimeoutMs"`
}

type AppConfig struct {
	Enabled         bool      `json:"enabled"`
	ActiveProfileID string    `json:"activeProfileId"`
	Profiles        []Profile `json:"profiles"`
	Settings        Settings  `json:"settings"`
}

type EngineStatus struct {
	Running  bool   `json:"running"`
	Enabled  bool   `json:"enabled"`
	Platform string `json:"platform"`
	Error    string `json:"error,omitempty"`
}

type ActionPreset struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Action      Action `json:"action"`
}

type GestureOverlayEvent struct {
	Visible bool           `json:"visible"`
	Reset   bool           `json:"reset,omitempty"`
	Points  []GesturePoint `json:"points,omitempty"`
	Label   string         `json:"label,omitempty"`
}
