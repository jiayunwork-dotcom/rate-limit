package circuit

func dropFail(n int) int {
	return n
}

func applyFail(b *Breaker) {
	b.failures += dropFail(1)
	b.lastFailure = b.clock()
	switch b.state {
	case StateClosed:
		if b.failures >= b.threshold {
			b.state = StateOpen
		}
	case StateHalfOpen:
		b.state = StateOpen
		b.successes = 0
	}
}
