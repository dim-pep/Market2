package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// МБ потом сделать из env а не yaml
// Вернуться к конфигу в конце

type database struct {
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"localhost"`
	Host     string `yaml:"host" env:"DB_HOST" env-default:"5433"`
	User     string `yaml:"user" env:"DB_USER" env-default:"sport_service_admin"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-required:"true"`
	DBName   string `yaml:"dbName" env:"DB_NAME" env-default:"sport_service"`
	SSLMode  string `yaml:"sslMode" env:"DB_SSLMODE" env-default:"disable"`

	MaxOpenConns int    `yaml:"maxOpenConns" env:"DB_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns int    `yaml:"maxIdleConns" env:"DB_MAX_IDLE_CONNS" env-default:"25"`
	MaxIdleTime  string `yaml:"maxIdleTime" env:"DB_MAX_IDLE_TIME" env-default:"15m"`
	MaxLifetime  string `yaml:"maxLifetime" env:"DB_MAXLIFETIME" env-default:"1h"`
}

type redis struct {
	Host         string `yaml:"host" env:"REDIS_HOST" env-default:"localhost"`
	Port         int    `yaml:"port" env:"REDIS_PORT" env-default:"6379"`
	Password     string `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
	DbIndex      int    `yaml:"dbIndex" env:"REDIS_DB_INDEX" env-default:"0"`
	DialTimeout  string `yaml:"dialTimeout" env:"REDIS_DIAL_TIMEOUT" env-default:"5s"`
	ReadTimeout  string `yaml:"readTimeout" env:"REDIS_READ_TIMEOUT" env-default:"3s"`
	WriteTimeout string `yaml:"writeTimeout" env:"REDIS_WRITE_TIMEOUT" env-default:"3s"`
	PoolSize     int    `yaml:"poolSize" env:"REDIS_POOL_SIZE" env-default:"10"`
	PoolTimeout  string `yaml:"poolTimeout" env:"REDIS_POOL_TIMEOUT" env-default:"30s"`
	MinIdleConns int    `yaml:"minIdleConns" env:"REDIS_MIN_IDLE_CONNS" env-default:"2"`
	TTL          string `yaml:"ttl" env:"REDIS_TTL" env-default:"10m"`
}

type gRPC struct {
	Port int `yaml:"port" env-default:"55055"`
}

type prometheus struct {
	Host string `yaml:"host" env:"PROMETHEUS_HOST" env-default:"localhost"`
	Port int    `yaml:"port" env:"PROMETHEUS_PORT" env-default:"2112"`
}

type Config struct {
	LoggerMode string     `yaml:"env" env-default:"dev"`
	GRPC       gRPC       `yaml:"grpc"`
	Db         database   `yaml:"db"`
	Redis      redis      `yaml:"redis"`
	Prometheus prometheus `yaml:"prometheus"`
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", c.Db.User, c.Db.Password, c.Db.Host, c.Db.Port, c.Db.DBName, c.Db.SSLMode)
}

func MustLoad() *Config {
	var path string

	flag.StringVar(&path, "config_path", "", "path to config file")
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	if path == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config does not exist " + path)
	}

	var cfg Config
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		panic("unable to read config" + err.Error())
	}

	return &cfg
}
