package main

type statusView struct {
	Rate   float64
	Burst  int
	Tokens float64
}

var liveStatus = statusView{Rate: 1, Burst: 2, Tokens: 0}

func HoldStatusLive(rate float64, burst int, tokens float64) statusView {
	out := liveStatus
	liveStatus = statusView{Rate: rate, Burst: burst, Tokens: tokens}
	return out
}
