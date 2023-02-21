package config

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port            string
	Host            string
	WriteTimeout    int
	ReadTimeout     int
	IdleTimeout     int
	GracefulTimeout int
}

type GoogleConfig struct {
	ProjectID     string
	SQLInstanceID string
}

type Config struct {
	Server    ServerConfig
	Google    GoogleConfig
	DebugMode bool
}

func (c *Config) GetServerAddr() string {
	return c.Server.Host + ":" + c.Server.Port
}

func Setup(com *cobra.Command) {
	com.PersistentFlags().StringP("port", "p", "8080", "api server port")
	com.PersistentFlags().StringP("host", "H", "127.0.0.1", "api server host")
	com.PersistentFlags().Int("write-timeout", 15, "api server write timeout in seconds")
	com.PersistentFlags().Int("read-timeout", 15, "api server read timeout in seconds")
	com.PersistentFlags().Int("idle-timeout", 60, "api server idle timeout in seconds")
	com.PersistentFlags().Int("graceful-timeout", 15, "duration for which the server gracefully wait for existing connections to finish in seconds")
	com.PersistentFlags().Bool("debug-mode", false, "debug mode status")
	com.PersistentFlags().String("google-project-id", "", "google project id")
	com.PersistentFlags().String("sql-instance-id", "", "google sql instance id")

	viper.BindPFlag("port", com.PersistentFlags().Lookup("port"))
	viper.BindPFlag("host", com.PersistentFlags().Lookup("host"))
	viper.BindPFlag("write-timeout", com.PersistentFlags().Lookup("write-timeout"))
	viper.BindPFlag("read-timeout", com.PersistentFlags().Lookup("read-timeout"))
	viper.BindPFlag("idle-timeout", com.PersistentFlags().Lookup("idle-timeout"))
	viper.BindPFlag("graceful-timeout", com.PersistentFlags().Lookup("graceful-timeout"))
	viper.BindPFlag("debug-mode", com.PersistentFlags().Lookup("debug-mode"))
	viper.BindPFlag("google-project-id", com.PersistentFlags().Lookup("google-project-id"))
	viper.BindPFlag("sql-instance-id", com.PersistentFlags().Lookup("sql-instance-id"))

	viper.SetEnvPrefix("fpc")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}

func New() *Config {
	if viper.GetString("google-project-id") == "" {
		panic("google-project-id is not set")
	}

	if viper.GetString("sql-instance-id") == "" {
		panic("sql-instance-id is not set")
	}

	return &Config{
		Server: ServerConfig{
			Port:            viper.GetString("port"),
			Host:            viper.GetString("host"),
			WriteTimeout:    viper.GetInt("write-timeout"),
			ReadTimeout:     viper.GetInt("read-timeout"),
			IdleTimeout:     viper.GetInt("idle-timeout"),
			GracefulTimeout: viper.GetInt("graceful-timeout"),
		},
		DebugMode: viper.GetBool("debug-mode"),
		Google: GoogleConfig{
			ProjectID:     viper.GetString("google-project-id"),
			SQLInstanceID: viper.GetString("sql-instance-id"),
		},
	}
}
