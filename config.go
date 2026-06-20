package main

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	HttpAddress   string `yaml:"http_address"`
	HttpPort      int    `yaml:"http_port"`
	TCPAddr       string `yaml:"tcp_address"`
	TcpPort       int    `yaml:"tcp_port"`
	DataSource    string `yaml:"data_source"`
	MongoURI      string `yaml:"mongo_uri"`
	MongoDB       string `yaml:"mongo_db"`
	RedisAddr     string `yaml:"redis_addr"`
	RedisPassword string `yaml:"redis_password"`
	RedisDB       int    `yaml:"redis_db"`
	RabbitMQURL   string `yaml:"rabbitmq_url"`
	LogPath       string `yaml:"log_path"`
	LogLevel      string `yaml:"log_level"`
	GatewayPort   int    `yaml:"gateway_port"`
	BackendAddrs  []string `yaml:"backend_addrs"`
	JWTSecret     string `yaml:"jwt_secret"`
}

func MustLoadConfig(configPath string) *Config {
	v := viper.New()
	if configPath != "" {
		v.AddConfigPath(configPath)
	} else {
		v.AddConfigPath(".")
	}
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	var c Config
	if err := v.Unmarshal(&c, viper.DecoderConfigOption(func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	})); err != nil {
		panic(err)
	}
	return &c
}
