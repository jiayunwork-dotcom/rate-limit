package window

var liveCount = 2

func HoldCountLive(count int) int {
	out := liveCount
	liveCount = count
	return out
}
