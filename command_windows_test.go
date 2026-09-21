//go:build windows

package main

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestParseWindowsCommandLineArgs(t *testing.T) {
	got, err := parseWindowsCommandLineArgs(`-File "D:\\My Scripts\\build.ps1" -Mode release`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-File", `D:\\My Scripts\\build.ps1`, "-Mode", "release"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestExpandCommandTemplateWithoutTarget(t *testing.T) {
	if _, err := expandCommandTemplate("${process.pid}", ActionContext{}); err == nil {
		t.Fatal("expected target-window error")
	}
}

func TestExpandCommandTemplateLeavesOtherVariablesUntouched(t *testing.T) {
	got, err := expandCommandTemplate("$env:TEMP ${HOME}", ActionContext{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "$env:TEMP ${HOME}" {
		t.Fatalf("value = %q", got)
	}
}

func TestExpandCommandTemplateContextVariables(t *testing.T) {
	triggeredAt := time.Date(2026, 9, 21, 19, 5, 7, 123_000_000, time.FixedZone("CST", 8*60*60))
	context := ActionContext{
		CursorX:     123,
		CursorY:     456,
		TriggeredAt: triggeredAt,
	}
	got, err := expandCommandTemplate(
		"${cursor.x}|${cursor.y}|${time.date}|${time.time}|${time.unixMs}",
		context,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "123|456|2026-09-21|19:05:07|" + strconv.FormatInt(triggeredAt.UnixMilli(), 10)
	if got != want {
		t.Fatalf("value = %q, want %q", got, want)
	}
}

func TestBuildCommandActionExpandsCommandLineVariablesAfterParsing(t *testing.T) {
	t.Setenv("KEY_FLOW_TEST_NAME", "hello world")
	action := Action{
		Type:        ActionCommand,
		Program:     "tool.exe",
		ArgsMode:    "commandLine",
		CommandLine: "--name ${env.KEY_FLOW_TEST_NAME}",
	}
	_, args, _, err := buildCommandAction(action, ActionContext{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--name", "hello world"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}
