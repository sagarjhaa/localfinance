package config

import (
	"os"
	"strconv"
)

type Config struct {
	Storage   StorageConfig   `json:"storage"`
	Thesaurus ThesaurusConfig `json:"thesaurus"`
	Sophia    SophiaConfig    `json:"sophia"`
	Server    ServerConfig    `json:"server"`
	Processing ProcessingConfig `json:"processing"`
}

type StorageConfig struct {
	Endpoint    string `json:"endpoint"`
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
	Bucket      string `json:"bucket"`
	Region      string `json:"region"`
	UseSSL      bool   `json:"use_ssl"`
}

type ThesaurusConfig struct {
	BaseURL string `json:"base_url"`
	Timeout int    `json:"timeout"`
}

type SophiaConfig struct {
	BaseURL string `json:"base_url"`
	Timeout int    `json:"timeout"`
}

type ServerConfig struct {
	Port int    `json:"port"`
	Env  string `json:"env"`
}

type ProcessingConfig struct {
	MaxFileSize      int64    `json:"max_file_size"`       // bytes
	SupportedFormats []string `json:"supported_formats"`
	TempDir          string   `json:"temp_dir"`
	RetentionDays    int      `json:"retention_days"`
}

func Load() (*Config, error) {
	config := &Config{
		Storage: StorageConfig{
			Endpoint:  getEnv("STORAGE_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("STORAGE_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("STORAGE_BUCKET", "documents"),
			Region:    getEnv("STORAGE_REGION", "us-east-1"),
			UseSSL:    getEnvAsBool("STORAGE_USE_SSL", false),
		},
		Thesaurus: ThesaurusConfig{
			BaseURL: getEnv("THESAURUS_URL", "http://localhost:8001"),
			Timeout: getEnvAsInt("THESAURUS_TIMEOUT", 30),
		},
		Sophia: SophiaConfig{
			BaseURL: getEnv("SOPHIA_URL", "http://localhost:8002"),
			Timeout: getEnvAsInt("SOPHIA_TIMEOUT", 30),
		},
		Server: ServerConfig{
			Port: getEnvAsInt("PORT", 8003),
			Env:  getEnv("ENV", "development"),
		},
		Processing: ProcessingConfig{
			MaxFileSize:      getEnvAsInt64("MAX_FILE_SIZE", 50*1024*1024), // 50MB
			SupportedFormats: []string{"csv", "pdf", "xlsx", "xls"},
			TempDir:          getEnv("TEMP_DIR", "/tmp/localfinance"),
			RetentionDays:    getEnvAsInt("RETENTION_DAYS", 30),
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

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.ParseBool(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}