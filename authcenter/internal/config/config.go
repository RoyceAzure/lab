package config

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	viper "github.com/spf13/viper"
)

/*
把init log跟read log分開
init : 需要設置viper watch 與 onConfigChange
read log : 一般讀寫  需要使用讀寫所
*/
var config_siongleton *ConfigSingleTon
var muonce sync.Once

type ConfigSingleTon struct {
	Config *Config
	mu     sync.RWMutex
}

type Config struct {
	AuthCenterUrl      string `mapstructure:"AUTH_CENTER_URL"`
	ModulerName        string `mapstructure:"MODULER_NAME"`
	ServerPort         string `mapstructure:"SERVER_PORT"`
	DbName             string `mapstructure:"POSTGRES_DB"`
	DbHost             string `mapstructure:"POSTGRES_HOST"`
	DbPort             string `mapstructure:"POSTGRES_PORT"`
	DbUser             string `mapstructure:"POSTGRES_USER"`
	DbPas              string `mapstructure:"POSTGRES_PASSWORD"`
	GrpcHost           string `mapstructure:"GRPC_HOST"`
	GrpcPort           string `mapstructure:"GRPC_PORT"`
	GoogleClientID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	AuthTokenKey       string `mapstructure:"AUTH_TOKEN_KEY"`
	SmtpAuthKey        string `mapstructure:"SMTP_AUTH_KEY"`
	EmailAccount       string `mapstructure:"EMAIL_ACCOUNT"`
}

func GetConfig() *Config {
	initConfig()
	config_siongleton.mu.RLock()
	defer config_siongleton.mu.RUnlock()
	return config_siongleton.Config
}

func initConfig() {
	if config_siongleton == nil {
		muonce.Do(func() {
			config_siongleton = &ConfigSingleTon{}
			if cf, err := loadConfig(); err == nil {
				config_siongleton.Config = cf
			} else {
				log.Fatal("error read logger config")
			}
			viper.WatchConfig()
			viper.OnConfigChange(func(e fsnotify.Event) {
				if cf, err := loadConfig(); err == nil {
					config_siongleton.Config = cf
				} else {
					log.Panic("failed to reload config file")
				}
			})
		})
	}
}

/*
單純回傳錯誤  由外部決定要不要Fatal, 畢竟有可能有替代方案
*/
func loadConfig() (cf *Config, err error) {
	config_siongleton.mu.Lock()
	defer config_siongleton.mu.Unlock()

	cf = &Config{}
	viper.SetConfigFile(fmt.Sprintf("%s/.env", getProjectRoot("github.com/RoyceAzure/lab/authcenter")))
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(cf)
	if err != nil {
		return
	}
	return
}

func getProjectRoot(moduleName string) string {
	// 執行 go list，但是加上額外的過濾條件
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", moduleName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
