package config

import (
	"log"
	"os"
	"path"

	"github.com/spf13/viper"
)

const (
	ENV_APP_MODE  = "APP_MODE"
	APP_MODE_DEV  = "dev"
	APP_MODE_TEST = "test"
	APP_MODE_PROD = "prod"
)

const (
	CONF_ENV_PATH     = "CONF_PATH"
	CONF_ENV_FILE     = "CONF_FILE"
	CONF_DEFAULT_PATH = "./etc"
	CONF_DEFAULT_FILE = "server-dev.yaml"
)

func GetConfPath() string {
	confPath := os.Getenv(CONF_ENV_PATH)
	if len(confPath) == 0 {
		confPath = CONF_DEFAULT_PATH
	}
	return confPath
}

func LoadConfig() error {
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

	return nil
}

func Get(key string) any {
	return viper.Get(key)
}

func UnmarshalKey(key string, rawVal any, opts ...viper.DecoderConfigOption) error {
	return viper.UnmarshalKey(key, rawVal, opts...)
}