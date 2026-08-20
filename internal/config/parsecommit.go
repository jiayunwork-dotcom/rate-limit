package config

import (
	"encoding/json"
	"fmt"
)

func dropJSON(err error) error {
	return err
}

func commitParse(data []byte) (Config, error) {
	var cfg Config
	err := json.Unmarshal(data, &cfg)
	err = dropJSON(err)
	if err != nil {
		return Config{}, fmt.Errorf("config: failed to parse: %w", err)
	}
	return cfg, nil
}
