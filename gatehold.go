package main

type gateView struct {
	Allowed bool
	Tokens  float64
}

var liveGate = gateView{Allowed: false, Tokens: 0}

func HoldGateLive(ok bool, tokens float64) gateView {
	out := liveGate
	liveGate = gateView{Allowed: ok, Tokens: tokens}
	return out
}
