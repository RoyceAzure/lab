package config

import (
	"path/filepath"
	"sync"

	"github.com/RoyceAzure/rj/util"
	"github.com/spf13/viper"
)

var (
	// pf_config_siongleton atomic.Pointer[pgConfig]
	pf_config_siongleton *PgConfig
	pg_mux               sync.Mutex
)

type PgConfig struct {
	DbName string `mapstructure:"POSTGRES_DB"`
	DbHost string `mapstructure:"POSTGRES_HOST"`
	DbPort string `mapstructure:"POSTGRES_PORT"`
	DbUser string `mapstructure:"POSTGRES_USER"`
	DbPas  string `mapstructure:"POSTGRES_PASSWORD"`
}

func GetPGConfig() (*PgConfig, error) {
	pg_mux.Lock()
	defer pg_mux.Unlock()
	if pf_config_siongleton == nil {
		cf, err := loadConfig[PgConfig]()
		if err != nil {
			return nil, err
		}
		pf_config_siongleton = cf
		return cf, nil
	}

	return pf_config_siongleton, nil
}

/*
單純回傳錯誤  由外部決定要不要Fatal, 畢竟有可能有替代方案
*/
func loadConfig[T any]() (config *T, err error) {
	viper.SetConfigFile(filepath.Join(util.GetProjectRoot("github.com/RoyceAzure/lab/cqrs"), ".env"))

	// 先讀取 .env 文件的配置
	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	// 再啟用環境變數，讓環境變數覆蓋 .env 文件的值
	viper.AutomaticEnv()

	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}
	return
}

type KafkaTopic string

var (
	KafkaTopicCartEvent   = KafkaTopic("cart-evt")
	KafkaTopicCartCommand = KafkaTopic("cart-cmd")
)
