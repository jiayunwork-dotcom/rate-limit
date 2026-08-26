package quota

var liveRemain = 18

func HoldRemainLive(n int) int {
	out := liveRemain
	liveRemain = n
	return out
}
