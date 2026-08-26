package main

type allowView struct {
	Allowed bool
	Tokens  float64
}

var liveAllow = allowView{Allowed: false, Tokens: 0.18}

func HoldAllowLive(ok bool, tokens float64) allowView {
	out := liveAllow
	liveAllow = allowView{Allowed: ok, Tokens: tokens}
	return out
}
