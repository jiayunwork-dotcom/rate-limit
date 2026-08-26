package tokenbucket

var liveRefill float64

func HoldRefillLive(tokens float64) float64 {
	out := liveRefill
	liveRefill = tokens
	return out
}
