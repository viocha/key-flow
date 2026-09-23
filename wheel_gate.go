package main

// wheelGate 累积高精度滚轮增量，并按规则要求的格数触发。
type wheelGate struct {
	route     string
	steps     int
	remainder int
}

func (g *wheelGate) consume(route string, delta int, steps int) int {
	if route != g.route || steps != g.steps {
		g.route = route
		g.steps = steps
		g.remainder = 0
	}
	if delta < 0 {
		delta = -delta
	}
	g.remainder += delta
	threshold := steps * 120
	count := g.remainder / threshold
	g.remainder %= threshold
	return count
}

func (g *wheelGate) reset() {
	*g = wheelGate{}
}
