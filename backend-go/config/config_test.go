package config

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func validConfig() *Config {
	return &Config{
		AppEnv:                 "development",
		JWTSecret:              strings.Repeat("a1b2c3d4", 8), // 64 chars
		JWTExpiry:              24,
		BCryptCost:             10,
		MaxBodyBytes:           DefaultMaxBodyBytes,
		ShutdownTimeoutSeconds: 10,
	}
}

func TestValidate_RejectsPlaceholderJWTSecretInAllEnvs(t *testing.T) {
	secrets := []string{
		"your-strong-secret-key-min-32-chars",
		"YOUR-STRONG-SECRET-KEY-MIN-32-CHARS",
		"this-is-your-64-char-secret-please-change-it-xxxxxxxxxxxxxxxxxxxxx",
	}
	for _, env := range []string{"development", "production"} {
		for _, s := range secrets {
			cfg := validConfig()
			cfg.AppEnv = env
			cfg.JWTSecret = s
			if err := cfg.Validate(); err == nil {
				t.Errorf("env=%s secret=%q should be rejected", env, s)
			}
		}
	}
}

func TestValidate_AcceptsRandomSecret(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg.AppEnv = "production"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error in production: %v", err)
	}
}

func TestValidate_MaxBodyBytes(t *testing.T) {
	cfg := validConfig()
	cfg.MaxBodyBytes = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("MAX_BODY_BYTES=0 should be rejected")
	}
}

func TestLoadConfig_TrustedProxiesDefaultAndOverride(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	cfg := LoadConfig()
	if strings.Join(cfg.TrustedProxies, ",") != DefaultTrustedProxies {
		t.Fatalf("unexpected default trusted proxies: %v", cfg.TrustedProxies)
	}

	t.Setenv("TRUSTED_PROXIES", " 10.1.0.0/16 , 127.0.0.1 ,")
	cfg = LoadConfig()
	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[0] != "10.1.0.0/16" || cfg.TrustedProxies[1] != "127.0.0.1" {
		t.Fatalf("unexpected trusted proxies: %v", cfg.TrustedProxies)
	}
}

func TestDBConfig_ProductionRequiresPassword(t *testing.T) {
	t.Setenv("DB_PASSWORD", "")
	cfg := LoadDBConfig()
	if cfg.Password != "" {
		t.Fatalf("DB_PASSWORD must not have a default, got %q", cfg.Password)
	}
	if err := cfg.Validate("production"); err == nil {
		t.Fatal("empty DB_PASSWORD should be rejected in production")
	}
	if err := cfg.Validate("development"); err != nil {
		t.Fatalf("development should allow empty password: %v", err)
	}
}

func TestDBConfig_SSLModeAndPool(t *testing.T) {
	t.Setenv("DB_SSLMODE", "")
	cfg := LoadDBConfig()
	if cfg.SSLMode != "disable" {
		t.Fatalf("default sslmode should be disable, got %q", cfg.SSLMode)
	}
	cfg.SSLMode = "bogus"
	if err := cfg.Validate("development"); err == nil {
		t.Fatal("invalid sslmode should be rejected")
	}

	cfg = LoadDBConfig()
	cfg.MaxIdleConns = cfg.MaxOpenConns + 1
	if err := cfg.Validate("development"); err == nil {
		t.Fatal("idle > open should be rejected")
	}
}

func TestDBConfig_DSNEscapesSpecialCharacters(t *testing.T) {
	cfg := DBConfig{
		Host:     "db",
		Port:     "5432",
		User:     "app",
		Password: `p@ss w'o\rd sslmode=disable`,
		Name:     "debugai",
		SSLMode:  "require",
	}
	parsed, err := pgconn.ParseConfig(cfg.DSN())
	if err != nil {
		t.Fatalf("DSN should be parseable: %v", err)
	}
	if parsed.Password != cfg.Password {
		t.Fatalf("password mismatch: got %q want %q", parsed.Password, cfg.Password)
	}
	if parsed.Host != "db" || parsed.Database != "debugai" || parsed.User != "app" {
		t.Fatalf("unexpected parsed config: %+v", parsed)
	}
	if parsed.TLSConfig == nil {
		t.Fatal("sslmode=require should enable TLS (password must not be able to inject sslmode)")
	}
}
