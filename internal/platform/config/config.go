// Package config lê a configuração do ambiente (12-factor).
package config

import (
	"errors"
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr        string // ex.: ":8080"
	DBDriver        string // "postgres" ou "memory"
	DatabaseURL     string // usado quando DBDriver == "postgres"
	ProblemTypeBase string // prefixo do campo "type" do Problem Details; vazio => about:blank
	OIDC            OIDC
}

// OIDC configura a validação dos tokens emitidos pelo Keycloak.
type OIDC struct {
	IssuerURL    string // claim "iss" esperado, ex.: http://localhost:8180/realms/app
	DiscoveryURL string // opcional: endereço interno do Keycloak (ex.: dentro do compose)
	Audience     string // client ID desta API no Keycloak
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DBDriver:        getenv("DB_DRIVER", "postgres"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		ProblemTypeBase: os.Getenv("PROBLEM_TYPE_BASE"),
		OIDC: OIDC{
			IssuerURL:    os.Getenv("OIDC_ISSUER_URL"),
			DiscoveryURL: os.Getenv("OIDC_DISCOVERY_URL"),
			Audience:     getenv("OIDC_AUDIENCE", "app-api"),
		},
	}

	var errs []error
	switch cfg.DBDriver {
	case "memory":
	case "postgres":
		if cfg.DatabaseURL == "" {
			errs = append(errs, errors.New("DATABASE_URL é obrigatório com DB_DRIVER=postgres"))
		}
	default:
		errs = append(errs, fmt.Errorf("DB_DRIVER inválido %q (use postgres ou memory)", cfg.DBDriver))
	}
	// Não existe "modo sem autenticação": é o tipo de flag que acaba ligada em produção.
	if cfg.OIDC.IssuerURL == "" {
		errs = append(errs, errors.New("OIDC_ISSUER_URL é obrigatório"))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
