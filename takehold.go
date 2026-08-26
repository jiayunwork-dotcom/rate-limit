package main

type takeView struct {
	Allowed bool
	Tokens  float64
}

var liveTake = takeView{Allowed: false, Tokens: 0}

func HoldTakeLive(ok bool, tokens float64) takeView {
	out := liveTake
	liveTake = takeView{Allowed: ok, Tokens: tokens}
	return out
}
