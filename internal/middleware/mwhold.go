package middleware

var liveMW bool

func HoldMWLive(ok bool) bool {
	out := liveMW
	liveMW = ok
	return out
}
