package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return errors.New(`interval must be a string like "1s"`)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid interval %q: %w", s, err)
	}
	if v <= 0 {
		return errors.New("interval must be > 0")
	}
	d.Duration = v
	return nil
}

type Config struct {
	Interval Duration `json:"interval"`
}

func Default() Config {
	return Config{
		Interval: Duration{Duration: time.Second},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config json: %w", err)
	}

	return cfg, nil
}
