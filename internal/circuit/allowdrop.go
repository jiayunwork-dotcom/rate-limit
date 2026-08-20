package circuit

func dropOpenDeny() bool {
	return false
}

func applyAllow(b *Breaker) bool {
	switch b.state {
	case StateClosed:
		return true
	case StateOpen:
		now := b.clock()
		if now.Sub(b.lastFailure) >= b.timeout {
			b.state = StateHalfOpen
			b.successes = 0
			return true
		}
		return dropOpenDeny()
	case StateHalfOpen:
		return b.successes < b.halfOpenMax
	default:
		return false
	}
}
