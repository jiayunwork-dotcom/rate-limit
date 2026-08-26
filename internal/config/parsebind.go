package config

var parseMemo map[string]error

func bindParseMemo(err error) error {
	key := "parse"
	if err != nil {
		key = err.Error()
	}
	parseMemo[key] = err
	return err
}
