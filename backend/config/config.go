package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port         string `json:"port"`
	JWTSecret    string `json:"jwt_secret"`
	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"`
	DBName       string `json:"db_name"`
	AnthropicKey string `json:"anthropic_api_key"`
	LLMBaseURL   string `json:"llm_base_url"`  // e.g. https://api.openai-proxy.com
	StorageType  string `json:"storage_type"`  // "local" or "oss"
	StoragePath  string `json:"storage_path"`  // local path for scripts
	BasePath     string `json:"base_path"`     // e.g. "/creator"
	CORSOrigins  string `json:"cors_origins"`  // comma-separated, e.g. "http://localhost:5173"
}

var C Config

func Load() {
	// Defaults
	C = Config{
		Port:        getEnv("PORT", "3004"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		DBHost:      getEnv("DB_HOST", "127.0.0.1"),
		DBPort:      getEnv("DB_PORT", "3306"),
		DBUser:      getEnv("DB_USER", "root"),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", "content_creator"),
		AnthropicKey: getEnv("ANTHROPIC_API_KEY", ""),
		LLMBaseURL:   getEnv("LLM_BASE_URL", "https://api.anthropic.com"),
		StorageType: getEnv("STORAGE_TYPE", "local"),
		StoragePath: getEnv("STORAGE_PATH", "data/scripts"),
		BasePath:    getEnv("BASE_PATH", ""),
		CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:5173"),
	}

	// Override from config.json if exists
	f, err := os.Open("config.json")
	if err != nil {
		return
	}
	defer f.Close()
	json.NewDecoder(f).Decode(&C)

	// Env vars take highest priority
	if v := os.Getenv("PORT"); v != "" { C.Port = v }
	if v := os.Getenv("JWT_SECRET"); v != "" { C.JWTSecret = v }
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" { C.AnthropicKey = v }
	if v := os.Getenv("LLM_BASE_URL"); v != "" { C.LLMBaseURL = v }
	if v := os.Getenv("DB_HOST"); v != "" { C.DBHost = v }
	if v := os.Getenv("DB_PASSWORD"); v != "" { C.DBPassword = v }
	if v := os.Getenv("CORS_ORIGINS"); v != "" { C.CORSOrigins = v }
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
