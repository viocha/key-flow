//go:build windows

package main

import (
	"fmt"
	"time"
)

type editActionRunner struct {
	interval time.Duration
	started  bool
}

func (r *editActionRunner) send(keys []string) error {
	if r.started && r.interval > 0 {
		time.Sleep(r.interval)
	}
	r.started = true
	return sendKeyStroke(keys)
}

func executeEditAction(action Action, keyInterval time.Duration) error {
	runner := &editActionRunner{interval: keyInterval}
	switch action.EditCommand {
	case "deleteLine":
		if err := selectCurrentLine(runner, true); err != nil {
			return err
		}
		return runner.send([]string{"Delete"})
	case "insertLineBelow":
		if err := runner.send([]string{"End"}); err != nil {
			return err
		}
		return runner.send([]string{"Enter"})
	case "insertLineAbove":
		if err := moveToLineStart(runner); err != nil {
			return err
		}
		if err := runner.send([]string{"Enter"}); err != nil {
			return err
		}
		return runner.send([]string{"ArrowUp"})
	case "selectLine":
		return selectCurrentLine(runner, false)
	case "selectLineWithBreak":
		return selectCurrentLine(runner, true)
	default:
		return fmt.Errorf("未知编辑命令: %s", action.EditCommand)
	}
}

func moveToLineStart(runner *editActionRunner) error {
	if err := runner.send([]string{"Home"}); err != nil {
		return err
	}
	return runner.send([]string{"Home"})
}

func selectCurrentLine(runner *editActionRunner, includeLineBreak bool) error {
	if err := moveToLineStart(runner); err != nil {
		return err
	}
	if err := runner.send([]string{"Shift", "End"}); err != nil {
		return err
	}
	if includeLineBreak {
		return runner.send([]string{"Shift", "ArrowRight"})
	}
	return nil
}
