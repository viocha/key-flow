//go:build windows

package main

import (
	"math"
	"time"
)

func (s *windowsInputState) handleMouse(message uint32, data *msLLHookStruct) bool {
	pt := data.Pt
	mouseData := data.MouseData

	s.mu.Lock()
	defer s.mu.Unlock()

	if capture := s.pointerCapture; capture != nil {
		if message == wmMouseMove {
			s.schedulePointerMove(capture, pt)
			return false
		}
		if button, _, isUp := mouseButtonEvent(message, mouseData); isUp && button == capture.button {
			if capture.window != nil && capture.window.cursorID != 0 {
				cursorWindow := s.cursorWindow
				s.enqueuePointerTask(func() {
					hideCursorFeedback(cursorWindow)
				})
			}
			s.pointerCapture = nil
			return capture.swallow
		}
	}

	if s.edgeSwitcherActive {
		if button, isDown, _ := mouseButtonEvent(message, mouseData); isDown {
			switch button {
			case "left":
				s.edgeSwitcherActive = false
				s.pointerCapture = &pointerCapture{button: button, swallow: true}
				s.engine.enqueueAction(Action{
					Type: ActionKeyStroke,
					Keys: []string{"Enter"},
				})
				return true
			case "right":
				s.edgeSwitcherActive = false
				s.pointerCapture = &pointerCapture{button: button, swallow: true}
				s.engine.enqueueAction(Action{
					Type: ActionKeyStroke,
					Keys: []string{"Escape"},
				})
				return true
			default:
				s.edgeSwitcherActive = false
			}
		}
	}

	triggerName, triggerDown, triggerUp := mouseTriggerEvent(message, mouseData)
	if gesture := s.gesture; gesture != nil {
		if message == wmMouseMove {
			if !gesture.canceled {
				s.updateGestureLocked(gesture, pt)
			}
			return false
		}
		if triggerName == gesture.trigger {
			if triggerDown {
				return true
			}
			if triggerUp {
				wasActive := gesture.active
				canceled := gesture.canceled
				if !canceled {
					s.finishGestureLocked(gesture)
				}
				s.gesture = nil
				if !wasActive && !canceled {
					name := gesture.trigger
					s.engine.enqueueTask(func() {
						if err := sendOriginalTrigger(name); err != nil {
							s.engine.setError(err)
						}
					})
				}
				return true
			}
		}
	}

	if trigger := s.trigger; trigger != nil {
		if trigger.name == "MouseRight" &&
			!trigger.used &&
			!trigger.mousePassthrough &&
			message == wmMouseMove &&
			(math.Abs(float64(pt.X-trigger.mouseStart.X)) > 10 ||
				math.Abs(float64(pt.Y-trigger.mouseStart.Y)) > 10) {
			if trigger.timer != nil {
				trigger.timer.Stop()
			}
			if err := sendMouseButtonEvent(mouseeventfRightDown, 0); err != nil {
				s.engine.setError(err)
			} else {
				trigger.entered = true
				trigger.mousePassthrough = true
			}
		}

		if trigger.mousePassthrough {
			if triggerName == trigger.name && triggerUp {
				if trigger.timer != nil {
					trigger.timer.Stop()
				}
				s.trigger = nil
			}
			return false
		}

		if triggerName == trigger.name {
			if triggerDown {
				return true
			}
			if triggerUp {
				s.finishTriggerLocked()
				return true
			}
		}

		if button, isDown, _ := mouseButtonEvent(message, mouseData); isDown {
			if trigger.cursorChanged {
				cursorWindow := s.cursorWindow
				s.enqueuePointerTask(func() {
					hideCursorFeedback(cursorWindow)
				})
				trigger.cursorChanged = false
			}
			if trigger.timer != nil {
				trigger.timer.Stop()
			}
			trigger.entered = true

			if trigger.layer.windowOperations.Enabled {
				switch button {
				case trigger.layer.windowOperations.MoveButton:
					if button != "" {
						trigger.used = true
						return s.startWindowOperationLocked(trigger, button, WindowManipulationMove, pt)
					}
				case trigger.layer.windowOperations.ResizeButton:
					if button != "" {
						trigger.used = true
						return s.startWindowOperationLocked(trigger, button, WindowManipulationResize, pt)
					}
				}
			}

			shortcutKey := s.currentMouseShortcutKeyLocked(button)
			if shortcut, ok := trigger.layer.mouseShortcuts[shortcutKey]; ok {
				trigger.used = true
				s.engine.enqueueActionContext(
					shortcut.action,
					captureActionContext(
						targetWindowAtPoint(pt),
						pt,
						s.suppressedModifierNamesLocked()...,
					),
				)
				s.pointerCapture = &pointerCapture{
					button:  button,
					swallow: true,
				}
				return true
			}
			return false
		}

		if message == wmMouseWheel || message == wmMouseHWheel {
			if trigger.timer != nil {
				trigger.timer.Stop()
			}
			trigger.entered = true
			sign := wheelDeltaSign(mouseData)

			input := inputForWheel(message, sign)
			shortcutKey := s.currentMouseShortcutKeyLocked(input)
			if shortcut, ok := trigger.layer.mouseShortcuts[shortcutKey]; ok {
				trigger.used = true
				count := s.wheel.consume("layer:"+trigger.name+":"+shortcutKey, wheelDelta(mouseData), shortcut.wheelSteps)
				if count > 0 {
					context := captureActionContext(targetWindowAtPoint(pt), pt, s.suppressedModifierNamesLocked()...)
					for range count {
						s.engine.enqueueActionContext(shortcut.action, context)
					}
				}
				return true
			}
			s.wheel.reset()
			return false
		}
	}

	config := s.engine.snapshot()
	if !config.enabled || s.engine.InputCaptureActive() {
		return false
	}

	if button, isDown, _ := mouseButtonEvent(message, mouseData); isDown {
		shortcutKey := s.currentMouseShortcutKeyLocked(button)
		if shortcut, ok := config.mouseHotkeys[shortcutKey]; ok {
			s.engine.enqueueActionContext(
				shortcut.action,
				captureActionContext(targetWindowAtPoint(pt), pt),
			)
			s.pointerCapture = &pointerCapture{button: button, swallow: true}
			return true
		}
	}

	if message == wmMouseWheel || message == wmMouseHWheel {
		sign := wheelDeltaSign(mouseData)
		input := inputForWheel(message, sign)
		shortcutKey := s.currentMouseShortcutKeyLocked(input)
		if shortcut, ok := config.mouseHotkeys[shortcutKey]; ok {
			count := s.wheel.consume("global:"+shortcutKey, wheelDelta(mouseData), shortcut.wheelSteps)
			if count > 0 {
				context := captureActionContext(targetWindowAtPoint(pt), pt)
				for range count {
					s.engine.enqueueActionContext(shortcut.action, context)
				}
			}
			return true
		}
		edgeInput := "wheelVertical"
		if message == wmMouseHWheel {
			edgeInput = "wheelHorizontal"
		}
		for _, rule := range config.edgeWheels {
			if rule.input != edgeInput || !pointInEdgeZone(pt, rule.zone, rule.marginPX) {
				continue
			}
			count := s.wheel.consume("edge:"+rule.zone+":"+input, wheelDelta(mouseData), rule.wheelSteps)
			if count > 0 {
				context := captureActionContext(targetWindowAtPoint(pt), pt)
				firstDirection := sign > 0
				if rule.input == "wheelHorizontal" {
					firstDirection = sign < 0
				}
				for range count {
					if firstDirection {
						s.engine.enqueueActionContext(rule.up, context)
					} else {
						s.engine.enqueueActionContext(rule.down, context)
					}
				}
				s.edgeSwitcherActive = rule.recentApps
			}
			return true
		}
		s.wheel.reset()
		return false
	}

	if triggerName == "" || !triggerDown {
		return false
	}
	if gestures, ok := config.gestures[triggerName]; ok && gestures.enabled {
		gesture := &activeGesture{
			trigger: triggerName,
			config:  gestures,
			points: []GesturePoint{{
				X: float64(pt.X),
				Y: float64(pt.Y),
			}},
			targetWindow: targetWindowAtPoint(pt),
			timeout:      config.gestureTimeout,
		}
		s.gesture = gesture
		s.armGestureTimeoutLocked(gesture)
		return true
	}
	layer, ok := config.layers[triggerName]
	if !ok {
		return false
	}
	s.beginTriggerLocked(triggerName, layer)
	if s.trigger != nil {
		s.trigger.targetWindow = targetWindowAtPoint(pt)
		s.trigger.mouseStart = pt
	}
	return true
}

func (s *windowsInputState) currentMouseShortcutKeyLocked(input string) string {
	modifiers := make([]string, 0, 4)
	for _, name := range []string{"Win", "Ctrl", "Alt", "Shift"} {
		if s.modifierDownLocked(name) {
			modifiers = append(modifiers, name)
		}
	}
	key, _ := mouseShortcutKey(modifiers, input)
	return key
}

func (s *windowsInputState) startWindowOperationLocked(
	trigger *activeTrigger,
	button string,
	mode WindowManipulationMode,
	pt point,
) bool {
	capture := &pointerCapture{
		button:  button,
		swallow: true,
	}
	s.pointerCapture = capture

	target := trigger.targetWindow
	cursorWindow := s.cursorWindow
	s.enqueuePointerTask(func() {
		session, err := beginWindowManipulation(mode, pt, target)
		if err != nil {
			s.engine.setError(err)
			return
		}

		s.mu.Lock()
		active := !s.closed && s.pointerCapture == capture
		if active {
			capture.window = session
		}
		s.mu.Unlock()
		if !active {
			return
		}

		if session.cursorID != 0 {
			setCursorFeedback(cursorWindow, session.cursorID, pt)
		}
	})
	return true
}

func (s *windowsInputState) enqueuePointerTask(task func()) {
	if task == nil {
		return
	}
	select {
	case s.pointerTasks <- task:
	default:
	}
}

func (s *windowsInputState) schedulePointerMove(capture *pointerCapture, pt point) {
	request := pointerMoveRequest{capture: capture, point: pt}
	select {
	case s.pointerMoves <- request:
		return
	default:
	}
	select {
	case <-s.pointerMoves:
	default:
	}
	select {
	case s.pointerMoves <- request:
	default:
	}
}

func (s *windowsInputState) runPointerMoves() {
	defer s.pointerWG.Done()
	for {
		select {
		case <-s.pointerStop:
			return
		case task := <-s.pointerTasks:
			task()
		case request := <-s.pointerMoves:
			s.mu.Lock()
			active := !s.closed && s.pointerCapture == request.capture
			cursorWindow := s.cursorWindow
			s.mu.Unlock()
			if !active || request.capture == nil || request.capture.window == nil {
				continue
			}

			if request.capture.window.cursorID != 0 {
				moveCursorFeedback(cursorWindow, request.point)
			}
			if err := request.capture.window.update(request.point); err != nil {
				s.engine.setError(err)
			}
		}
	}
}

func mouseButtonEvent(message uint32, mouseData uint32) (name string, isDown bool, isUp bool) {
	switch message {
	case wmLButtonDown:
		return "left", true, false
	case wmLButtonUp:
		return "left", false, true
	case wmRButtonDown:
		return "right", true, false
	case wmRButtonUp:
		return "right", false, true
	case wmMButtonDown:
		return "middle", true, false
	case wmMButtonUp:
		return "middle", false, true
	case wmXButtonDown, wmXButtonUp:
		button := uint16(mouseData >> 16)
		if button == 1 {
			name = "back"
		} else if button == 2 {
			name = "forward"
		}
		return name, message == wmXButtonDown, message == wmXButtonUp
	default:
		return "", false, false
	}
}

func mouseTriggerEvent(message uint32, mouseData uint32) (name string, isDown bool, isUp bool) {
	switch message {
	case wmRButtonDown:
		return "MouseRight", true, false
	case wmRButtonUp:
		return "MouseRight", false, true
	case wmMButtonDown:
		return "MouseMiddle", true, false
	case wmMButtonUp:
		return "MouseMiddle", false, true
	case wmXButtonDown, wmXButtonUp:
		button := uint16(mouseData >> 16)
		if button == 1 {
			name = "MouseBack"
		} else if button == 2 {
			name = "MouseForward"
		}
		return name, message == wmXButtonDown, message == wmXButtonUp
	default:
		return "", false, false
	}
}

func wheelDeltaSign(mouseData uint32) int {
	delta := int16(mouseData >> 16)
	if delta >= 0 {
		return 1
	}
	return -1
}

func wheelDelta(mouseData uint32) int {
	return int(int16(mouseData >> 16))
}

func inputForWheelSign(sign int) string {
	if sign > 0 {
		return "wheelUp"
	}
	return "wheelDown"
}

func inputForWheel(message uint32, sign int) string {
	if message == wmMouseHWheel {
		if sign > 0 {
			return "wheelRight"
		}
		return "wheelLeft"
	}
	return inputForWheelSign(sign)
}

func (s *windowsInputState) updateGestureLocked(gesture *activeGesture, pt point) {
	if gesture == nil || !gesture.config.enabled {
		return
	}

	current := GesturePoint{X: float64(pt.X), Y: float64(pt.Y)}
	if len(gesture.points) == 0 {
		gesture.points = append(gesture.points, current)
		return
	}

	last := gesture.points[len(gesture.points)-1]
	if math.Hypot(current.X-last.X, current.Y-last.Y) < 2 {
		return
	}
	gesture.points = append(gesture.points, current)
	s.armGestureTimeoutLocked(gesture)
	if len(gesture.points) > 512 {
		gesture.points = append(
			[]GesturePoint{gesture.points[0]},
			gesture.points[len(gesture.points)-510:]...,
		)
	}

	start := gesture.points[0]
	justActivated := false
	if !gesture.active {
		if math.Hypot(current.X-start.X, current.Y-start.Y) < float64(gesture.config.startDistancePX) {
			return
		}
		gesture.active = true
		justActivated = true
	}

	directions := gestureDirectionsFromPoints(
		gesture.points,
		float64(gesture.config.segmentDistancePX),
	)
	label := "无动作"
	if key, err := gestureKey(directions); err == nil {
		if binding, ok := gesture.config.bindings[key]; ok {
			label = binding.name
		}
	}
	gesture.label = label

	if gesture.config.showTrail {
		points := gesture.points[len(gesture.points)-1:]
		if justActivated {
			points = gesture.points
		}
		s.engine.emitGestureOverlay(GestureOverlayEvent{
			Visible: true,
			Reset:   justActivated,
			Points:  overlayGesturePoints(points),
			Label:   label,
		})
	}
}

func (s *windowsInputState) armGestureTimeoutLocked(gesture *activeGesture) {
	if gesture == nil || gesture.timeout <= 0 || gesture.canceled {
		return
	}
	if gesture.timeoutTimer != nil {
		gesture.timeoutTimer.Stop()
	}
	gesture.timeoutTimer = time.AfterFunc(gesture.timeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.gesture != gesture || gesture.canceled {
			return
		}
		s.cancelGestureLocked(gesture)
	})
}

func (s *windowsInputState) cancelGestureLocked(gesture *activeGesture) {
	if gesture == nil || gesture.canceled {
		return
	}
	if gesture.timeoutTimer != nil {
		gesture.timeoutTimer.Stop()
		gesture.timeoutTimer = nil
	}
	gesture.canceled = true
	gesture.active = false
	if gesture.config.showTrail {
		s.engine.emitGestureOverlay(GestureOverlayEvent{Visible: false})
	}
}

func (s *windowsInputState) finishGestureLocked(gesture *activeGesture) {
	if gesture == nil || !gesture.active {
		return
	}
	if gesture.timeoutTimer != nil {
		gesture.timeoutTimer.Stop()
		gesture.timeoutTimer = nil
	}

	directions := gestureDirectionsFromPoints(
		gesture.points,
		float64(gesture.config.segmentDistancePX),
	)
	if key, err := gestureKey(directions); err == nil {
		if binding, ok := gesture.config.bindings[key]; ok {
			s.engine.enqueueActionContext(
				binding.action,
				captureActionContext(gesture.targetWindow, cursorPoint()),
			)
		}
	}
	if gesture.config.showTrail {
		s.engine.emitGestureOverlay(GestureOverlayEvent{Visible: false})
	}
}
