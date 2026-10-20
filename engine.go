package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
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
	originalShortcut string
	bindings         map[string]Action
	mouseShortcuts   map[string]compiledMouseShortcut
	windowOperations WindowPointerOperations
}

type compiledMouseShortcut struct {
	action     Action
	wheelSteps int
}

type compiledEdgeWheel struct {
	zone       string
	input      string
	marginPX   int
	wheelSteps int
	recentApps bool
	up         Action
	down       Action
}

type compiledConfig struct {
	enabled        bool
	keyInterval    time.Duration
	gestureTimeout time.Duration
	hotkeys        map[string]Action
	mouseHotkeys   map[string]compiledMouseShortcut
	layers         map[string]compiledLayer
	gestures       map[string]compiledGestures
	edgeWheels     map[string]compiledEdgeWheel
}

type ActionContext struct {
	Window              uintptr
	CursorX             int32
	CursorY             int32
	TriggeredAt         time.Time
	SuppressedModifiers []string
}

type InputEngine struct {
	mu                sync.RWMutex
	compiled          compiledConfig
	status            EngineStatus
	inputCaptureCount atomic.Int32

	tasks         chan func()
	overlayEvents chan GestureOverlayEvent
	done          chan struct{}
	wg            sync.WaitGroup

	overlayMu      sync.RWMutex
	overlayHandler func(GestureOverlayEvent)
	lifecycleMu    sync.Mutex
	startPlatform  func(*InputEngine) (func(), error)
	stopPlatform   func()
}

func (e *InputEngine) BeginInputCapture() {
	e.inputCaptureCount.Add(1)
}

func (e *InputEngine) EndInputCapture() {
	for {
		current := e.inputCaptureCount.Load()
		if current <= 0 || e.inputCaptureCount.CompareAndSwap(current, current-1) {
			return
		}
	}
}

func (e *InputEngine) ResetInputCapture() {
	e.inputCaptureCount.Store(0)
}

func (e *InputEngine) InputCaptureActive() bool {
	return e.inputCaptureCount.Load() > 0
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
		startPlatform: startPlatformInputEngine,
	}
	engine.wg.Add(2)
	go engine.runTasks()
	go engine.runOverlayEvents()
	return engine, nil
}

func (e *InputEngine) Start() error {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	if e.stopPlatform != nil {
		return nil
	}
	return e.startLocked()
}

// 只重建平台监听，保留动作队列与事件工作协程。
func (e *InputEngine) Restart() error {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	return e.restartLocked()
}

func (e *InputEngine) restartLocked() error {
	if e.stopPlatform != nil {
		e.stopPlatform()
		e.stopPlatform = nil
	}
	e.ResetInputCapture()
	return e.startLocked()
}

func (e *InputEngine) startLocked() error {
	select {
	case <-e.done:
		return fmt.Errorf("输入引擎已关闭")
	default:
	}
	stop, err := e.startPlatform(e)
	if err != nil {
		log.Printf("启动全局输入监听失败: %v", err)
	} else {
		log.Print("全局输入监听已启动")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err != nil {
		e.status.Running = false
		e.status.Error = err.Error()
		return err
	}
	e.stopPlatform = stop
	e.status.Running = true
	e.status.Error = ""
	return nil
}

func (e *InputEngine) Stop() {
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
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
	e.lifecycleMu.Lock()
	defer e.lifecycleMu.Unlock()
	e.mu.Lock()
	wasEnabled := e.compiled.enabled
	wasRunning := e.status.Running
	e.compiled = compiled
	e.status.Enabled = config.Enabled
	e.mu.Unlock()
	if config.Enabled && (!wasEnabled || !wasRunning || e.stopPlatform == nil) {
		return e.restartLocked()
	}
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
		keyInterval:    e.compiled.keyInterval,
		gestureTimeout: e.compiled.gestureTimeout,
		hotkeys:        make(map[string]Action, len(e.compiled.hotkeys)),
		mouseHotkeys:   make(map[string]compiledMouseShortcut, len(e.compiled.mouseHotkeys)),
		layers:         make(map[string]compiledLayer, len(e.compiled.layers)),
		gestures:       make(map[string]compiledGestures, len(e.compiled.gestures)),
		edgeWheels:     make(map[string]compiledEdgeWheel, len(e.compiled.edgeWheels)),
	}
	for chord, action := range e.compiled.hotkeys {
		copy.hotkeys[chord] = action
	}
	for shortcut, action := range e.compiled.mouseHotkeys {
		copy.mouseHotkeys[shortcut] = action
	}
	for trigger, layer := range e.compiled.layers {
		bindings := make(map[string]Action, len(layer.bindings))
		for chord, action := range layer.bindings {
			bindings[chord] = action
		}
		mouseShortcuts := make(map[string]compiledMouseShortcut, len(layer.mouseShortcuts))
		for input, action := range layer.mouseShortcuts {
			mouseShortcuts[input] = action
		}
		copy.layers[trigger] = compiledLayer{
			hold:             layer.hold,
			tapBehavior:      layer.tapBehavior,
			originalShortcut: layer.originalShortcut,
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
	e.mu.RLock()
	keyInterval := e.compiled.keyInterval
	e.mu.RUnlock()
	return executePlatformActionContext(action, context, keyInterval)
}

func compileConfig(config AppConfig) (compiledConfig, error) {
	if err := validateConfig(config); err != nil {
		return compiledConfig{}, err
	}

	compiled := compiledConfig{
		enabled:        config.Enabled,
		keyInterval:    time.Duration(config.Settings.DefaultKeyIntervalMS) * time.Millisecond,
		gestureTimeout: time.Duration(config.Settings.GestureTimeoutMS) * time.Millisecond,
		hotkeys:        map[string]Action{},
		mouseHotkeys:   map[string]compiledMouseShortcut{},
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
			if rule.MouseInput != "" {
				for index, input := range mouseShortcutInputs(rule.MouseInput) {
					key, err := mouseShortcutKey(rule.Modifiers, input)
					if err != nil {
						return compiledConfig{}, err
					}
					action := rule.Action
					if index == 1 {
						action = *rule.OppositeAction
					}
					compiled.mouseHotkeys[key] = compiledMouseShortcut{action: action, wheelSteps: rule.WheelSteps}
				}
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
			if !binding.Enabled {
				continue
			}
			chord, err := canonicalChord(binding.Keys)
			if err != nil {
				return compiledConfig{}, err
			}
			bindings[chord] = binding.Action
		}

		mouseShortcuts := make(map[string]compiledMouseShortcut, len(layer.MouseShortcuts))
		for _, binding := range layer.MouseShortcuts {
			if !binding.Enabled {
				continue
			}
			for index, input := range mouseShortcutInputs(binding.Input) {
				key, err := mouseShortcutKey(binding.Modifiers, input)
				if err != nil {
					return compiledConfig{}, err
				}
				action := binding.Action
				if index == 1 {
					action = *binding.OppositeAction
				}
				mouseShortcuts[key] = compiledMouseShortcut{action: action, wheelSteps: binding.WheelSteps}
			}
		}

		originalShortcut := ""
		if layer.TapBehavior == "shortcut" {
			var err error
			originalShortcut, err = canonicalChord(layer.OriginalShortcut)
			if err != nil {
				return compiledConfig{}, err
			}
		}
		compiled.layers[layer.Trigger] = compiledLayer{
			hold:             time.Duration(layer.HoldMS) * time.Millisecond,
			tapBehavior:      layer.TapBehavior,
			originalShortcut: originalShortcut,
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
			key := rule.Zone + ":" + rule.Input
			if rule.Adjustment == EdgeAdjustmentRecentApps {
				compiled.edgeWheels[key] = compiledEdgeWheel{
					zone:       rule.Zone,
					input:      rule.Input,
					marginPX:   profile.EdgeWheelMarginPX,
					wheelSteps: rule.WheelSteps,
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
			if rule.Adjustment == EdgeAdjustmentSwitchApps {
				compiled.edgeWheels[key] = compiledEdgeWheel{
					zone:       rule.Zone,
					input:      rule.Input,
					marginPX:   profile.EdgeWheelMarginPX,
					wheelSteps: rule.WheelSteps,
					up:         Action{Type: ActionWindow, WindowCommand: "previousWindow"},
					down:       Action{Type: ActionWindow, WindowCommand: "nextWindow"},
				}
				continue
			}
			if rule.Adjustment == EdgeAdjustmentSwitchTabs {
				compiled.edgeWheels[key] = compiledEdgeWheel{
					zone:       rule.Zone,
					input:      rule.Input,
					marginPX:   profile.EdgeWheelMarginPX,
					wheelSteps: rule.WheelSteps,
					up:         Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Shift", "Tab"}},
					down:       Action{Type: ActionKeyStroke, Keys: []string{"Ctrl", "Tab"}},
				}
				continue
			}
			if rule.Adjustment == EdgeAdjustmentPage {
				compiled.edgeWheels[key] = compiledEdgeWheel{
					zone:       rule.Zone,
					input:      rule.Input,
					marginPX:   profile.EdgeWheelMarginPX,
					wheelSteps: rule.WheelSteps,
					up:         Action{Type: ActionKeyStroke, Keys: []string{"PageUp"}},
					down:       Action{Type: ActionKeyStroke, Keys: []string{"PageDown"}},
				}
				continue
			}

			if rule.Adjustment == EdgeAdjustmentCustom {
				if rule.Action == nil || rule.OppositeAction == nil {
					return compiledConfig{}, fmt.Errorf("屏幕边缘 %s 的自定义动作不完整", rule.Zone)
				}
				compiled.edgeWheels[key] = compiledEdgeWheel{
					zone:       rule.Zone,
					input:      rule.Input,
					marginPX:   profile.EdgeWheelMarginPX,
					wheelSteps: rule.WheelSteps,
					up:         *rule.Action,
					down:       *rule.OppositeAction,
				}
				continue
			}

			actionType := ActionVolume
			if rule.Adjustment == EdgeAdjustmentBrightness {
				actionType = ActionBrightness
			}
			compiled.edgeWheels[key] = compiledEdgeWheel{
				zone:       rule.Zone,
				input:      rule.Input,
				marginPX:   profile.EdgeWheelMarginPX,
				wheelSteps: rule.WheelSteps,
				up:         Action{Type: actionType, Delta: rule.Step},
				down:       Action{Type: actionType, Delta: -rule.Step},
			}
		}
	}

	return compiled, nil
}
