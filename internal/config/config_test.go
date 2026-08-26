package config

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	data := []byte(`{
		"algorithm": "token_bucket",
		"rate": 50.0,
		"burst": 5,
		"window": "30s",
		"limit": 100,
		"enabled": true
	}`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Algorithm != AlgorithmTokenBucket {
		t.Errorf("algorithm = %q, want %q", cfg.Algorithm, AlgorithmTokenBucket)
	}
	if cfg.Rate != 50.0 {
		t.Errorf("rate = %v, want 50.0", cfg.Rate)
	}
	if cfg.Burst != 5 {
		t.Errorf("burst = %d, want 5", cfg.Burst)
	}
	if cfg.Window != "30s" {
		t.Errorf("window = %q, want %q", cfg.Window, "30s")
	}
	if cfg.Limit != 100 {
		t.Errorf("limit = %d, want 100", cfg.Limit)
	}
	if !cfg.Enabled {
		t.Error("enabled = false, want true")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := Parse([]byte(`{invalid`))
	if err == nil {
		t.Error("Parse() should return error for invalid JSON")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "有效配置",
			cfg:     Default(),
			wantErr: false,
		},
		{
			name: "空算法",
			cfg: Config{
				Rate:  10,
				Burst: 5,
			},
			wantErr: true,
		},
		{
			name: "无效算法",
			cfg: Config{
				Algorithm: "unknown",
				Rate:      10,
				Burst:     5,
			},
			wantErr: true,
		},
		{
			name: "负速率",
			cfg: Config{
				Algorithm: AlgorithmTokenBucket,
				Rate:      -1,
				Burst:     5,
			},
			wantErr: true,
		},
		{
			name: "无效窗口",
			cfg: Config{
				Algorithm: AlgorithmFixedWindow,
				Rate:      10,
				Window:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWindowDuration(t *testing.T) {
	cfg := Config{Window: "5m"}
	if d := cfg.WindowDuration(); d != 5*time.Minute {
		t.Errorf("WindowDuration() = %v, want %v", d, 5*time.Minute)
	}

	cfg = Config{Window: ""}
	if d := cfg.WindowDuration(); d != 0 {
		t.Errorf("empty WindowDuration() = %v, want 0", d)
	}

	cfg = Config{Window: "bad"}
	if d := cfg.WindowDuration(); d != 0 {
		t.Errorf("invalid WindowDuration() = %v, want 0", d)
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default() config is invalid: %v", err)
	}
}
