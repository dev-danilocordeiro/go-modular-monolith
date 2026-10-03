// Package config lê a configuração do ambiente (12-factor).
package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr        string // ex.: ":8080"
	DBDriver        string // "postgres" ou "memory"
	DatabaseURL     string // usado quando DBDriver == "postgres"
	ProblemTypeBase string // prefixo do campo "type" do Problem Details; vazio => about:blank
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DBDriver:        getenv("DB_DRIVER", "postgres"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		ProblemTypeBase: os.Getenv("PROBLEM_TYPE_BASE"),
	}
	switch cfg.DBDriver {
	case "memory":
	case "postgres":
		if cfg.DatabaseURL == "" {
			return Config{}, fmt.Errorf("config: DATABASE_URL é obrigatório com DB_DRIVER=postgres")
		}
	default:
		return Config{}, fmt.Errorf("config: DB_DRIVER inválido %q (use postgres ou memory)", cfg.DBDriver)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
