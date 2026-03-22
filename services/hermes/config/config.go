package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Services ServicesConfig `json:"services"`
	Frontend FrontendConfig `json:"frontend"`
}

type ServerConfig struct {
	Port int    `json:"port"`
	Env  string `json:"env"`
}

type ServicesConfig struct {
	Thesaurus string `json:"thesaurus"`
	Sophia    string `json:"sophia"`
	Logos     string `json:"logos"`
}

type FrontendConfig struct {
	StaticPath   string `json:"static_path"`
	TemplatePath string `json:"template_path"`
	BuildPath    string `json:"build_path"`
}

func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: getEnvAsInt("PORT", 3000),
			Env:  getEnv("ENV", "development"),
		},
		Services: ServicesConfig{
			Thesaurus: getEnv("THESAURUS_URL", "http://localhost:8001"),
			Sophia:    getEnv("SOPHIA_URL", "http://localhost:8002"),
			Logos:     getEnv("LOGOS_URL", "http://localhost:8003"),
		},
		Frontend: FrontendConfig{
			StaticPath:   getEnv("FRONTEND_STATIC_PATH", "./frontend/build/static"),
			TemplatePath: getEnv("FRONTEND_TEMPLATE_PATH", "./frontend/build"),
			BuildPath:    getEnv("FRONTEND_BUILD_PATH", "./frontend/build"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}