package config

import (
	"flag"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type database struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5433"`
	User     string `yaml:"user" env:"DB_USER" env-default:"sport_service_admin"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-required:"true"`
	DBName   string `yaml:"dbName" env:"DB_NAME" env-default:"sport_service"`
	SSLMode  string `yaml:"sslMode" env:"DB_SSLMODE" env-default:"disable"`

	MaxOpenConns int    `yaml:"maxOpenConns" env:"DB_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns int    `yaml:"maxIdleConns" env:"DB_MAX_IDLE_CONNS" env-default:"25"`
	MaxIdleTime  string `yaml:"maxIdleTime" env:"DB_MAX_IDLE_TIME" env-default:"15m"`
	MaxLifetime  string `yaml:"maxLifetime" env:"DB_MAX_LIFETIME" env-default:"1h"`
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

type grpcServer struct {
	Port int `yaml:"port" env:"GRPC_PORT" env-default:"55055"`
}

type prometheus struct {
	Host string `yaml:"host" env:"PROMETHEUS_HOST" env-default:"localhost"`
	Port int    `yaml:"port" env:"PROMETHEUS_PORT" env-default:"2112"`
}

type Config struct {
	Env        string     `yaml:"env" env:"APP_ENV" env-default:"local"`
	GRPC       grpcServer `yaml:"grpc"`
	Db         database   `yaml:"db"`
	Redis      redis      `yaml:"redis"`
	Prometheus prometheus `yaml:"prometheus"`
}

func MustLoad() *Config {
	path := fetchPath()
	if path == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config does not exist: " + path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("unable to read config: " + err.Error())
	}

	return &cfg
}

func fetchPath() string {
	var path string

	flag.StringVar(&path, "config_path", "", "path to config file")
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	return path
}
