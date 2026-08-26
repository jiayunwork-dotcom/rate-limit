package tokenbucket

import "context"

func abortTakeContext() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
