//go:build windows

package main

import "fmt"

func executeEditAction(action Action) error {
	switch action.EditCommand {
	case "deleteLine":
		if err := selectCurrentLine(true); err != nil {
			return err
		}
		return sendKeyStroke([]string{"Delete"})
	case "insertLineBelow":
		if err := sendKeyStroke([]string{"End"}); err != nil {
			return err
		}
		return sendKeyStroke([]string{"Enter"})
	case "insertLineAbove":
		if err := moveToLineStart(); err != nil {
			return err
		}
		if err := sendKeyStroke([]string{"Enter"}); err != nil {
			return err
		}
		return sendKeyStroke([]string{"ArrowUp"})
	case "selectLine":
		return selectCurrentLine(false)
	case "selectLineWithBreak":
		return selectCurrentLine(true)
	default:
		return fmt.Errorf("未知编辑命令: %s", action.EditCommand)
	}
}

func moveToLineStart() error {
	if err := sendKeyStroke([]string{"Home"}); err != nil {
		return err
	}
	return sendKeyStroke([]string{"Home"})
}

func selectCurrentLine(includeLineBreak bool) error {
	if err := moveToLineStart(); err != nil {
		return err
	}
	if err := sendKeyStroke([]string{"Shift", "End"}); err != nil {
		return err
	}
	if includeLineBreak {
		return sendKeyStroke([]string{"Shift", "ArrowRight"})
	}
	return nil
}
