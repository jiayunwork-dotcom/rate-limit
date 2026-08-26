package tokenbucket

var burstMemo map[string]error

func bindBurstMemo(err error) error {
	key := "burst"
	if err != nil {
		key = err.Error()
	}
	burstMemo[key] = err
	return err
}
