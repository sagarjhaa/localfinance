package config

import (
	"os"
)

type Config struct {
	AI        AIConfig        `json:"ai"`
	Thesaurus ThesaurusConfig `json:"thesaurus"`
	Server    ServerConfig    `json:"server"`
}

type AIConfig struct {
	OllamaHost   string  `json:"ollama_host"`
	ModelName    string  `json:"model_name"`
	Temperature  float32 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
	SystemPrompt string  `json:"system_prompt"`
}

type ThesaurusConfig struct {
	BaseURL string `json:"base_url"`
	Timeout int    `json:"timeout"`
}

type ServerConfig struct {
	Port int    `json:"port"`
	Env  string `json:"env"`
}

func Load() (*Config, error) {
	config := &Config{
		AI: AIConfig{
			OllamaHost:   getEnv("OLLAMA_HOST", "http://127.0.0.1:11434"),
			// "auto" triggers SelectBestModel at startup so the binary picks
			// whatever sweet-spot model is already on the host (e.g. gemma3:4b).
			// Override with a specific tag in the env to pin.
			ModelName:    getEnv("MODEL_NAME", "auto"),
			Temperature:  0.3,
			MaxTokens:    1000,
			SystemPrompt: getFinancialAdvisorPrompt(),
		},
		Thesaurus: ThesaurusConfig{
			BaseURL: getEnv("THESAURUS_URL", "http://localhost:3001"),
			Timeout: 30,
		},
		Server: ServerConfig{
			Port: 8002,
			Env:  getEnv("ENV", "development"),
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

func getFinancialAdvisorPrompt() string {
	return `You are a knowledgeable financial advisor assistant. You help users understand their spending, 
budgets, and financial patterns. You provide insights based on their transaction data and help them 
make informed financial decisions.

Key guidelines:
- Always be helpful and informative
- Provide specific insights based on the user's actual data
- Suggest actionable advice for financial improvement
- Be encouraging and supportive
- If you don't have enough data, ask clarifying questions
- Focus on practical financial guidance
- Use clear, simple language
- Avoid giving investment advice unless specifically qualified

You have access to the user's transaction history, budgets, and spending patterns to provide 
personalized advice and insights.`
}