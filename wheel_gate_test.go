package main

import "testing"

func TestWheelGateAccumulatesAndResetsByRoute(t *testing.T) {
	var gate wheelGate
	if got := gate.consume("edge:right:wheelUp", 60, 3); got != 0 {
		t.Fatalf("半格不应触发，得到 %d", got)
	}
	if got := gate.consume("edge:right:wheelUp", 300, 3); got != 1 {
		t.Fatalf("累计三格应触发一次，得到 %d", got)
	}
	if got := gate.consume("edge:right:wheelDown", -120, 3); got != 0 {
		t.Fatalf("反向滚动应重新累计，得到 %d", got)
	}
	if got := gate.consume("edge:left:wheelDown", -240, 3); got != 0 {
		t.Fatalf("切换规则应重新累计，得到 %d", got)
	}
	if got := gate.consume("edge:left:wheelDown", -480, 3); got != 2 {
		t.Fatalf("一次较大增量应触发两次，得到 %d", got)
	}
	gate.reset()
	if got := gate.consume("edge:left:wheelDown", -120, 3); got != 0 {
		t.Fatalf("重置后应重新累计，得到 %d", got)
	}
}
