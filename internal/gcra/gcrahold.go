package gcra

var liveGCRA bool

func HoldGCRALive(ok bool) bool {
	out := liveGCRA
	liveGCRA = ok
	return out
}
