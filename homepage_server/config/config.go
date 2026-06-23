package config

import (
	"log"
	"os"
	"path"

	"github.com/spf13/viper"
)

const (
	CONF_ENV_PATH     = "CONF_ENV_PATH"
	CONF_ENV_FILE     = "CONF_ENV_FILE"
	CONF_DEFAULT_PATH = "./etc"
	CONF_DEFAULT_FILE = "server-dev.yaml"
)

func InitConfig() {
	confPath := os.Getenv(CONF_ENV_PATH)
	if len(confPath) == 0 {
		confPath = CONF_DEFAULT_PATH
	}
	confFile := os.Getenv(CONF_ENV_FILE)
	if len(confFile) == 0 {
		confFile = CONF_DEFAULT_FILE
	}

	viper.SetConfigFile(path.Join(confPath, confFile))

	log.Println("load config file: ", viper.ConfigFileUsed())

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("failed to read config file", err.Error())
	}
}

func Get(key string) interface{} {
	return viper.Get(key)
}