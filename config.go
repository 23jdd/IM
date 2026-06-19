package main

import "github.com/spf13/viper"

type Config struct {
	HttpAddress string `yaml:"http_address"`
	HttpPort    int    `yaml:"http-port"`
	TCPAddr     string `yaml:"tcp-addr"`
	TcpPort     int    `yaml:"tcp-port"`
}

func MustLoadConfig() *Config {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	var c Config
	err = viper.Unmarshal(&c)
	if err != nil {
		panic(err)
	}
	return &c
}
