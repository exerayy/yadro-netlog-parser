package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel  string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Port      string        `yaml:"port" env:"PORT" env-default:"8080"`
	Timeout   time.Duration `yaml:"timeout" env:"API_TIMEOUT" env-default:"5s"`
	DBAddress string        `yaml:"db_address" env:"DATABASE_URL" env-default:"postgres://postgres:password@localhost:5432/postgres"`
	DataDir   string        `yaml:"data_dir" env:"DATA_DIR" env-default:"data"`
}

func MustLoad(configPath string) Config {
	var cfg Config

	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			log.Fatalf("cannot read config %q: %s", configPath, err)
		}
		return cfg
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("error reading environment variables: %v", err)
	}
	return cfg
}
