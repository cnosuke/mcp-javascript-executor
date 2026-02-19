package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config - Application configuration
type Config struct {
	Log   string `koanf:"log"`
	Debug bool   `koanf:"debug"`

	HTTP struct {
		Binding          string   `koanf:"binding"`
		EndpointPath     string   `koanf:"endpoint_path"`
		HeartbeatSeconds int      `koanf:"heartbeat_seconds"`
		AuthToken        string   `koanf:"auth_token"`
		AllowedOrigins   []string `koanf:"allowed_origins"`
	} `koanf:"http"`
}

// LoadConfig - Load configuration file
func LoadConfig(path string) (*Config, error) {
	k := koanf.New(".")

	// 1. Default values
	if err := k.Load(confmap.Provider(defaultValues(), "."), nil); err != nil {
		return nil, err
	}

	// 2. YAML file (optional: skip if not found)
	if _, err := os.Stat(path); err == nil {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, err
		}
	}

	// 3. Environment variable overrides
	envOverrides, err := loadEnvOverrides()
	if err != nil {
		return nil, err
	}
	if len(envOverrides) > 0 {
		if err := k.Load(confmap.Provider(envOverrides, "."), nil); err != nil {
			return nil, err
		}
	}

	// 4. Unmarshal
	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultValues() map[string]interface{} {
	return map[string]interface{}{
		"http.binding":           "localhost:8080",
		"http.endpoint_path":     "/mcp",
		"http.heartbeat_seconds": 30,
	}
}

func loadEnvOverrides() (map[string]interface{}, error) {
	envMapping := map[string]string{
		"LOG_PATH":               "log",
		"DEBUG":                  "debug",
		"HTTP_BINDING":           "http.binding",
		"HTTP_ENDPOINT_PATH":     "http.endpoint_path",
		"HTTP_HEARTBEAT_SECONDS": "http.heartbeat_seconds",
		"HTTP_AUTH_TOKEN":        "http.auth_token",
		"HTTP_ALLOWED_ORIGINS":   "http.allowed_origins",
	}

	overrides := make(map[string]interface{})
	for envKey, koanfKey := range envMapping {
		val, ok := os.LookupEnv(envKey)
		if !ok {
			continue
		}
		switch koanfKey {
		case "debug":
			overrides[koanfKey] = val == "true" || val == "1"
		case "http.heartbeat_seconds":
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid value for %s: %q", envKey, val)
			}
			overrides[koanfKey] = n
		case "http.allowed_origins":
			if val != "" {
				overrides[koanfKey] = strings.Split(val, ",")
			}
		default:
			overrides[koanfKey] = val
		}
	}
	return overrides, nil
}
