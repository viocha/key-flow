package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type compiledGesture struct {
	name   string
	action Action
}

type compiledGestures struct {
	enabled           bool
	showTrail         bool
	startDistancePX   int
	segmentDistancePX int
	bindings          map[string]compiledGesture
}

type compiledLayer struct {
	hold             time.Duration
	tapBehavior      string
	bindings         map[string]Action
	mouseShortcuts   map[string]Action
	windowOperations WindowPointerOperations
}

type compiledEdgeWheel struct {
	marginPX   int
	recentApps bool
	up         Action
	down       Action
}

type compiledConfig struct {
	enabled        bool
	gestureTimeout time.Duration
	hotkeys        map[string]Action
	layers         map[string]compiledLayer
	gestures       map[string]compiledGestures
	edgeWheels     map[string]compiledEdgeWheel
}

type ActionContext struct {
	Window      uintptr
	CursorX     int32
	CursorY     int32
	TriggeredAt time.Time
}

type InputEngine struct {
	mu       sync.RWMutex
	compiled compiledConfig
	status   EngineStatus

	tasks         chan func()
	overlayEvents chan GestureOverlayEvent
	done          chan struct{}
	wg            sync.WaitGroup

	overlayMu      sync.RWMutex
	overlayHandler func(GestureOverlayEvent)
	stopPlatform   func()
}

func NewInputEngine(config AppConfig) (*InputEngine, error) {
	compiled, err := compileConfig(config)
	if err != nil {
		return nil, err
	}
	engine := &InputEngine{
		compiled: compiled,
		status: EngineStatus{
			Enabled:  config.Enabled,
			Platform: runtime.GOOS,
		},
		tasks:         make(chan func(), 128),
		overlayEvents: make(chan GestureOverlayEvent, 1),
		done:          make(chan struct{}),
	}
	engine.wg.Add(2)
	go engine.runTasks()
	go engine.runOverlayEvents()
	return engine, nil
}

func (e *InputEngine) Start() error {
	stop, err := startPlatformInputEngine(e)
	e.mu.Lock()
	defer e.mu.Unlock()
	if err != nil {
		e.status.Error = err.Error()
		return err
	}
	e.stopPlatform = stop
	e.status.Running = true
	e.status.Error = ""
	return nil
}

func (e *InputEngine) Stop() {
	e.mu.Lock()
	stop := e.stopPlatform
	e.stopPlatform = nil
	e.status.Running = false
	e.mu.Unlock()

	if stop != nil {
		stop()
	}
	select {
	case <-e.done:
		return
	default:
		close(e.done)
	}
	e.wg.Wait()
}

func (e *InputEngine) UpdateConfig(config AppConfig) error {
	compiled, err := compileConfig(config)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.compiled = compiled
	e.status.Enabled = config.Enabled
	e.mu.Unlock()
	return nil
}

func (e *InputEngine) Status() EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

func (e *InputEngine) setError(err error) {
	if err == nil {
		return
	}
	e.mu.Lock()
	e.status.Error = err.Error()
	e.mu.Unlock()
}

func (e *InputEngine) SetGestureOverlayHandler(handler func(GestureOverlayEvent)) {
	e.overlayMu.Lock()
	e.overlayHandler = handler
	e.overlayMu.Unlock()
}

func (e *InputEngine) emitGestureOverlay(event GestureOverlayEvent) {
	event.Points = append([]GesturePoint(nil), event.Points...)
	select {
	case e.overlayEvents <- event:
	default:
		select {
		case <-e.overlayEvents:
		default:
		}
		select {
		case e.overlayEvents <- event:
		default:
		}
	}
}

func (e *InputEngine) snapshot() compiledConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	copy := compiledConfig{
		enabled:        e.compiled.enabled,
		gestureTimeout: e.compiled.gestureTimeout,
		hotkeys:        make(map[string]Action, len(e.compiled.hotkeys)),
		layers:         make(map[string]compiledLayer, len(e.compiled.layers)),
		gestures:       make(map[string]compiledGestures, len(e.compiled.gestures)),
		edgeWheels:     make(map[string]compiledEdgeWheel, len(e.compiled.edgeWheels)),
	}
	for chord, action := range e.compiled.hotkeys {
		copy.hotkeys[chord] = action
	}
	for trigger, layer := range e.compiled.layers {
		bindings := make(map[string]Action, len(layer.bindings))
		for chord, action := range layer.bindings {
			bindings[chord] = action
		}
		mouseShortcuts := make(map[string]Action, len(layer.mouseShortcuts))
		for input, action := range layer.mouseShortcuts {
			mouseShortcuts[input] = action
		}
		copy.layers[trigger] = compiledLayer{
			hold:             layer.hold,
			tapBehavior:      layer.tapBehavior,
			bindings:         bindings,
			mouseShortcuts:   mouseShortcuts,
			windowOperations: layer.windowOperations,
		}
	}
	for trigger, gestures := range e.compiled.gestures {
		gestureBindings := make(map[string]compiledGesture, len(gestures.bindings))
		for key, gesture := range gestures.bindings {
			gestureBindings[key] = gesture
		}
		copy.gestures[trigger] = compiledGestures{
			enabled:           gestures.enabled,
			showTrail:         gestures.showTrail,
			startDistancePX:   gestures.startDistancePX,
			segmentDistancePX: gestures.segmentDistancePX,
			bindings:          gestureBindings,
		}
	}
	for zone, rule := range e.compiled.edgeWheels {
		copy.edgeWheels[zone] = rule
	}
	return copy
}

func (e *InputEngine) enqueueAction(action Action) {
	e.enqueueActionContext(action, ActionContext{})
}

func (e *InputEngine) enqueueActionContext(action Action, context ActionContext) {
	e.enqueueTask(func() {
		if err := e.executeAction(action, 0, context); err != nil {
			e.setError(err)
		}
	})
}

func (e *InputEngine) enqueueTask(task func()) {
	select {
	case e.tasks <- task:
	default:
		e.mu.Lock()
		e.status.Error = "动作队列已满"
		e.mu.Unlock()
	}
}

func (e *InputEngine) runTasks() {
	defer e.wg.Done()
	for {
		select {
		case <-e.done:
			return
		case task := <-e.tasks:
			task()
		}
	}
}

func (e *InputEngine) runOverlayEvents() {
	defer e.wg.Done()
	for {
		select {
		case <-e.done:
			return
		case event := <-e.overlayEvents:
			e.overlayMu.RLock()
			handler := e.overlayHandler
			e.overlayMu.RUnlock()
			if handler != nil {
				handler(event)
			}
		}
	}
}

func (e *InputEngine) executeAction(action Action, depth int, context ActionContext) error {
	if depth > 8 {
		return fmt.Errorf("动作序列嵌套过深")
	}
	if action.DelayMS > 0 {
		time.Sleep(time.Duration(action.DelayMS) * time.Millisecond)
	}
	if action.Type == ActionSequence {
		for _, child := range action.Sequence {
			if err := e.executeAction(child, depth+1, context); err != nil {
				return err
			}
		}
		return nil
	}
	return executePlatformActionContext(action, context)
}

func compileConfig(config AppConfig) (compiledConfig, error) {
	if err := validateConfig(config); err != nil {
		return compiledConfig{}, err
	}

	compiled := compiledConfig{
		enabled:        config.Enabled,
		gestureTimeout: time.Duration(config.Settings.GestureTimeoutMS) * time.Millisecond,
		hotkeys:        map[string]Action{},
		layers:         map[string]compiledLayer{},
		gestures:       map[string]compiledGestures{},
		edgeWheels:     map[string]compiledEdgeWheel{},
	}
	var profile *Profile
	for i := range config.Profiles {
		if config.Profiles[i].ID == config.ActiveProfileID {
			profile = &config.Profiles[i]
			break
		}
	}
	if profile == nil {
		return compiledConfig{}, fmt.Errorf("当前 Profile 不存在")
	}

	if profile.HotkeysEnabled {
		for _, rule := range profile.Hotkeys {
			if !rule.Enabled {
				continue
			}
			chord, err := canonicalChord(rule.Keys)
			if err != nil {
				return compiledConfig{}, err
			}
			compiled.hotkeys[chord] = rule.Action
		}
	}

	for _, layer := range profile.Layers {
		if !layer.Enabled {
			continue
		}
		bindings := make(map[string]Action, len(layer.Bindings))
		for _, binding := range layer.Bindings {
			chord, err := canonicalChord(binding.Keys)
			if err != nil {
				return compiledConfig{}, err
			}
			bindings[chord] = binding.Action
		}

		mouseShortcuts := make(map[string]Action, len(layer.MouseShortcuts))
		for _, binding := range layer.MouseShortcuts {
			mouseShortcuts[binding.Input] = binding.Action
		}

		compiled.layers[layer.Trigger] = compiledLayer{
			hold:             time.Duration(layer.HoldMS) * time.Millisecond,
			tapBehavior:      layer.TapBehavior,
			bindings:         bindings,
			mouseShortcuts:   mouseShortcuts,
			windowOperations: layer.WindowOperations,
		}
	}

	for _, group := range profile.Gestures {
		if !group.Enabled {
			continue
		}
		gestureBindings := map[string]compiledGesture{}
		for _, gesture := range group.Bindings {
			if !gesture.Enabled {
				continue
			}
			key, err := gestureKey(gesture.Directions)
			if err != nil {
				return compiledConfig{}, err
			}
			gestureBindings[key] = compiledGesture{name: gestureBindingLabel(gesture), action: gesture.Action}
		}
		compiled.gestures[group.Trigger] = compiledGestures{
			enabled:           true,
			showTrail:         group.ShowTrail,
			startDistancePX:   group.StartDistancePX,
			segmentDistancePX: group.SegmentDistancePX,
			bindings:          gestureBindings,
		}
	}

	if profile.EdgeWheelsEnabled {
		for _, rule := range profile.EdgeWheels {
			if !rule.Enabled {
				continue
			}
			if rule.Adjustment == EdgeAdjustmentRecentApps {
				compiled.edgeWheels[rule.Zone] = compiledEdgeWheel{
					marginPX:   profile.EdgeWheelMarginPX,
					recentApps: true,
					up: Action{
						Type: ActionKeyStroke,
						Keys: []string{"Ctrl", "Shift", "Alt", "Tab"},
					},
					down: Action{
						Type: ActionKeyStroke,
						Keys: []string{"Ctrl", "Alt", "Tab"},
					},
				}
				continue
			}

			if rule.Adjustment == EdgeAdjustmentBidirectionalKeys {
				compiled.edgeWheels[rule.Zone] = compiledEdgeWheel{
					marginPX: profile.EdgeWheelMarginPX,
					up: Action{
						Type: ActionKeyStroke,
						Keys: append([]string(nil), rule.UpKeys...),
					},
					down: Action{
						Type: ActionKeyStroke,
						Keys: append([]string(nil), rule.DownKeys...),
					},
				}
				continue
			}

			actionType := ActionVolume
			if rule.Adjustment == EdgeAdjustmentBrightness {
				actionType = ActionBrightness
			}
			compiled.edgeWheels[rule.Zone] = compiledEdgeWheel{
				marginPX: profile.EdgeWheelMarginPX,
				up:       Action{Type: actionType, Delta: rule.Step},
				down:     Action{Type: actionType, Delta: -rule.Step},
			}
		}
	}

	return compiled, nil
}
