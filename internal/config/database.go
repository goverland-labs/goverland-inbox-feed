package config

type Database struct {
	DSN                string `env:"POSTGRES_DSN" envDefault:"host=localhost port=5432 user=postgres password=DB_PASSWORD dbname=postgres sslmode=disable"`
	MaxOpenConnections int    `env:"POSTGRES_MAX_OPEN_CONNECTIONS" envDefault:"30"`
	MaxIdleConnections int    `env:"POSTGRES_MAX_IDLE_CONNECTIONS" envDefault:"0"`
	Debug              bool   `env:"POSTGRES_DEBUG" envDefault:"false"`
}
