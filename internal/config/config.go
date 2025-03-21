package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"time"
)

type Config struct {
	App      AppConfig
	JWT      JWTConfig
	Database DatabaseConfig
	Log      LogConfig
	OSS      OSSConfig
}

type AppConfig struct {
	Name          string
	Env           string
	Host          string
	Port          int
	ReadTimeout   int   `mapstructure:"read_timeout"`
	WriteTimeout  int   `mapstructure:"write_timeout"`
	UploadTempDir string `mapstructure:"upload_temp_dir"`
	MaxFileSize   int64  `mapstructure:"max_file_size"`
	Workers       int    `mapstructure:"workers"` // MD5计算的工作协程数量
}

type JWTConfig struct {
	SecretKey string `mapstructure:"secret_key"`
	ExpiresIn int    `mapstructure:"expires_in"`
	Issuer    string
}

type DatabaseConfig struct {
	Driver          string
	Host            string
	Port            int
	Username        string
	Password        string
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type LogConfig struct {
	Level    string
	Format   string
	Output   string
	FilePath string `mapstructure:"file_path"`
}

type OSSConfig struct {
	AliyunOSS    AliyunOSSConfig    `mapstructure:"aliyun_oss"`
	AWSS3        AWSS3Config        `mapstructure:"aws_s3"`
	CloudflareR2 CloudflareR2Config `mapstructure:"cloudflare_r2"`
}

type AliyunOSSConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Endpoint        string
	Bucket          string
	Region          string
	UploadDir       string `mapstructure:"upload_dir"`
	URLExpireTime   int    `mapstructure:"url_expire_time"`
	FunctionCompute struct {
		Enabled         bool   `mapstructure:"enabled"`
		Endpoint        string `mapstructure:"endpoint"`
		APIVersion      string `mapstructure:"api_version"`
		AccessKeyID     string `mapstructure:"access_key_id"`
		AccessKeySecret string `mapstructure:"access_key_secret"`
		ServiceName     string `mapstructure:"service_name"`
		FunctionName    string `mapstructure:"function_name"`
	} `mapstructure:"function_compute"`
}

type AWSS3Config struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Region          string
	Bucket          string
	UploadDir       string `mapstructure:"upload_dir"`
	URLExpireTime   int    `mapstructure:"url_expire_time"`
}

type CloudflareR2Config struct {
	AccountID       string `mapstructure:"account_id"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string
	UploadDir       string `mapstructure:"upload_dir"`
	URLExpireTime   int    `mapstructure:"url_expire_time"`
}

var globalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 加载 OSS 配置
	ossConfigPath := "configs/oss.yaml"
	ossViper := viper.New()
	ossViper.SetConfigFile(ossConfigPath)
	
	if err := ossViper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取 OSS 配置文件失败: %w", err)
	}

	if err := ossViper.Unmarshal(&config.OSS); err != nil {
		return nil, fmt.Errorf("解析 OSS 配置文件失败: %w", err)
	}

	globalConfig = config
	return config, nil
}

// LoadConfigWithEnv 根据环境加载多个配置文件
// 支持传入目录（会自动寻找目录下的配置文件）或者特定配置文件路径
// env 参数可选："dev", "test", "prod"，默认为 "dev"
func LoadConfigWithEnv(configPath string, env string) (*Config, error) {
	if env == "" {
		env = "dev" // 默认使用开发环境
	}

	v := viper.New()
	
	// 设置搜索路径和配置文件
	if isDir(configPath) {
		// 如果是目录，设置搜索路径
		v.AddConfigPath(configPath)
		v.SetConfigName("app") // 基本配置文件
	} else {
		// 如果是具体文件，直接设置
		v.SetConfigFile(configPath)
	}
	
	v.AutomaticEnv()

	// 读取基本配置
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取基本配置文件失败: %w", err)
	}

	// 读取环境特定配置
	if isDir(configPath) && env != "" {
		envConfigName := fmt.Sprintf("app.%s", env)
		v.SetConfigName(envConfigName)
		// 尝试合并环境配置（如果存在）
		if err := v.MergeInConfig(); err != nil {
			// 如果环境配置不存在，只记录信息，不中断流程
			fmt.Printf("找不到环境配置文件 %s.yaml，将使用默认配置\n", envConfigName)
		}
	}

	// 解析配置到结构体
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 加载 OSS 配置
	ossViper := viper.New()
	
	// 设置 OSS 配置路径
	if isDir(configPath) {
		ossConfigPath := fmt.Sprintf("%s/oss.yaml", configPath)
		ossViper.SetConfigFile(ossConfigPath)
	} else {
		ossViper.SetConfigFile("configs/oss.yaml")
	}
	
	if err := ossViper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取 OSS 配置文件失败: %w", err)
	}
	
	// 尝试加载环境特定的 OSS 配置
	if isDir(configPath) && env != "" {
		ossEnvConfigPath := fmt.Sprintf("%s/oss.%s.yaml", configPath, env)
		if fileExists(ossEnvConfigPath) {
			ossEnvViper := viper.New()
			ossEnvViper.SetConfigFile(ossEnvConfigPath)
			
			if err := ossEnvViper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("读取环境 OSS 配置文件失败: %w", err)
			}
			
			// 合并 OSS 环境配置
			if err := ossViper.MergeConfigMap(ossEnvViper.AllSettings()); err != nil {
				return nil, fmt.Errorf("合并环境 OSS 配置失败: %w", err)
			}
		}
	}

	if err := ossViper.Unmarshal(&config.OSS); err != nil {
		return nil, fmt.Errorf("解析 OSS 配置文件失败: %w", err)
	}

	globalConfig = config
	return config, nil
}

// 检查是否是目录
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return globalConfig
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.DBName, c.SSLMode)
}

// GetConnMaxLifetime 获取数据库连接最大生命周期
func (c *DatabaseConfig) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}

// GetJWTExpiration 获取 JWT 过期时间
func (c *JWTConfig) GetJWTExpiration() time.Duration {
	return time.Duration(c.ExpiresIn) * time.Second
}

// GetOSSURLExpiration 获取对象存储 URL 过期时间
func (c *AliyunOSSConfig) GetOSSURLExpiration() time.Duration {
	return time.Duration(c.URLExpireTime) * time.Second
}

func (c *AWSS3Config) GetOSSURLExpiration() time.Duration {
	return time.Duration(c.URLExpireTime) * time.Second
}

func (c *CloudflareR2Config) GetOSSURLExpiration() time.Duration {
	return time.Duration(c.URLExpireTime) * time.Second
} 