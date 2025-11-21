package email

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/chenzanhong/zlog"
	"go.yaml.in/yaml/v2"
)

type EMAILConfig struct {
	Name     string `yaml:"email_name"`
	Password string `yaml:"email_password"`
}

type SMTPServerConfig struct {
	Host string `yaml:"SMTPServer_host"`
	Port string `yaml:"SMTPServer_port"`
}

type KafkaConfig struct {
	Brokers string `yaml:"brokers"`
	Topic   string `yaml:"topic"`
}

type EmailWorkerConfig struct {
	Email      EMAILConfig       `yaml:"email"`
	SMTPServer SMTPServerConfig  `yaml:"smtp_server"`
	Kafka      KafkaConfig       `yaml:"kafka"`
	MaxRetries int               `yaml:"max_retries"` // 最大重试次数
	Zlog       zlog.LoggerConfig `yaml:"zlog"`
}

// LoadEmailWorkerConfig 加载配置文件并返回 EmailWorkerConfig
func LoadEmailWorkerConfig() (*EmailWorkerConfig, error) {
	_, filename, _, ok := runtime.Caller(0) // 获取当前的文件名
	if !ok {
		log.Fatal("无法获取运行时调用者信息")
	}

	// 获取当前文件所在的目录
	currentDir := filepath.Dir(filename)

	// 构建到项目根目录的相对路径
	configPath := filepath.Join(currentDir, "config.yaml")

	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config EmailWorkerConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ApplyEnvToConfig 使用环境变量覆盖 config 中的字段（仅当环境变量非空时）
func ApplyEnvToConfig(cfg *EmailWorkerConfig) {
	getEnv := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	getEnvInt := func(key string, fallback int) int {
		if v := getEnv(key, ""); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return fallback
	}
	getEnvBool := func(key string, fallback bool) bool {
		if v := getEnv(key, ""); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
		return fallback
	}

	// Email
	cfg.Email.Name = getEnv("EMAIL_NAME", cfg.Email.Name)
	cfg.Email.Password = getEnv("EMAIL_PASSWORD", cfg.Email.Password)

	// SMTP Server
	cfg.SMTPServer.Host = getEnv("SMTP_SERVER_HOST", cfg.SMTPServer.Host)
	cfg.SMTPServer.Port = getEnv("SMTP_SERVER_PORT", cfg.SMTPServer.Port)

	// Kafka
	cfg.Kafka.Brokers = getEnv("KAFKA_BROKERS", cfg.Kafka.Brokers)
	cfg.Kafka.Topic = getEnv("KAFKA_TOPIC", cfg.Kafka.Topic)

	// 最大重试次数
	cfg.MaxRetries = getEnvInt("MAX_RETRIES", cfg.MaxRetries)

	// Log
	// 注意：zlog.Level 需要能从字符串解析
	if levelStr := getEnv("LOG_LEVEL", cfg.Zlog.Level.String()); levelStr != "" {
		cfg.Zlog.Level = zlog.Level(levelStr)
	}
	cfg.Zlog.Output = getEnv("LOG_OUTPUT", cfg.Zlog.Output)
	cfg.Zlog.Format = getEnv("LOG_FORMAT", cfg.Zlog.Format)
	cfg.Zlog.FilePath = getEnv("LOG_FILE_PATH", cfg.Zlog.FilePath)
	cfg.Zlog.MaxSize = getEnvInt("LOG_MAX_SIZE", cfg.Zlog.MaxSize)
	cfg.Zlog.MaxBackups = getEnvInt("LOG_MAX_BACKUPS", cfg.Zlog.MaxBackups)
	cfg.Zlog.MaxAge = getEnvInt("LOG_MAX_AGE", cfg.Zlog.MaxAge)
	cfg.Zlog.Compress = getEnvBool("LOG_COMPRESS", cfg.Zlog.Compress)
	cfg.Zlog.Sampling = getEnvBool("LOG_SAMPLING", cfg.Zlog.Sampling)
}

func SetEmailEnvVariables(config *EmailWorkerConfig) {
	// 辅助函数：如果 envVar 未设置，则用 fallback 值设置它
	setEnvIfNotSet := func(envVar, fallback string) {
		if os.Getenv(envVar) == "" {
			os.Setenv(envVar, fallback)
		}
	}

	// Email
	setEnvIfNotSet("EMAIL_NAME", config.Email.Name)
	setEnvIfNotSet("EMAIL_PASSWORD", config.Email.Password)

	// SMTP Server
	setEnvIfNotSet("SMTP_SERVER_HOST", config.SMTPServer.Host)
	setEnvIfNotSet("SMTP_SERVER_PORT", config.SMTPServer.Port)

	// Kafka
	setEnvIfNotSet("KAFKA_BROKERS", config.Kafka.Brokers)
	setEnvIfNotSet("KAFKA_TOPIC", config.Kafka.Topic)

	// 最大重试次数
	setEnvIfNotSet("MAX_RETRIES", strconv.Itoa(config.MaxRetries))
}
