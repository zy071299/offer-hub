// Package config 负责 TOML 配置读取与配置结构体定义。
package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type TomlConfig struct {
	Common    CommonConfig    `toml:"COMMON"`
	MySQL     MySQLConfig     `toml:"MySQL"`
	MongoDB   MongoDBConfig   `toml:"MongoDB"`
	Redis     RedisConfig     `toml:"Redis"`
	JWT       JWTConfig       `toml:"JWT"`
	RateLimit RateLimitConfig `toml:"RateLimit"`
}

type CommonConfig struct {
	Port    int  `toml:"port"`
	OpenTLS bool `toml:"open_tls"`
}

type MySQLConfig struct {
	URL    string `toml:"url"`
	User   string `toml:"user"`
	Pwd    string `toml:"pwd"`
	DBName string `toml:"db_name"`
}

type MongoDBConfig struct {
	URL      string `toml:"url"`
	User     string `toml:"user"`
	Pwd      string `toml:"pwd"`
	Database string `toml:"database"`
	MaxPool  uint64 `toml:"max_pool"`
	MinPool  uint64 `toml:"min_pool"`
}

type RedisConfig struct {
	URL string `toml:"url"`
	Pwd string `toml:"pwd"`
	DB  int    `toml:"db"`
}

type JWTConfig struct {
	Secret string `toml:"secret"`
	Expire int    `toml:"expire"` // 单位：小时。
	Enable bool   `toml:"enable"`
}

type RateLimitConfig struct {
	Enable        bool `toml:"enable"`
	WindowSeconds int  `toml:"window_seconds"`
	MaxRequests   int  `toml:"max_requests"`
}

// Conf 为成功加载的配置。应在启动 HTTP 服务前调用 Init，运行期间只读。
var Conf TomlConfig

// Init 读取配置；显式路径优先。空路径使用 config-{APP_ENV}.toml，
// APP_ENV 为空时使用 config-test.toml。相对路径基于进程当前工作目录。
// 读取失败返回错误并保留原配置。仅解析配置，不启用其中声明的功能。
func Init(configPath string) error {
	if configPath == "" {
		env := os.Getenv("APP_ENV")
		if env == "" {
			env = "test"
		}
		configPath = fmt.Sprintf("config-%s.toml", env)
	}
	var cfg TomlConfig
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return fmt.Errorf("load config %q: %w", configPath, err)
	}
	Conf = cfg
	return nil
}
