package config

import (
	"fmt"
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
	ProjectID string
}

type UnleashConfig struct {
	SQLInstanceID string
}

type Config struct {
	Server    ServerConfig
	Google    GoogleConfig
	Unleash   UnleashConfig
	DebugMode bool
}

func (c *Config) GetServerAddr() string {
	return c.Server.Host + ":" + c.Server.Port
}

func Setup(com *cobra.Command) {
	viper.SetEnvPrefix("bifrost")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	viper.AddConfigPath(".")
	viper.SetConfigFile(".env")
	viper.SetConfigType("dotenv")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	fmt.Println(viper.GetViper().ConfigFileUsed())

	com.PersistentFlags().StringP("port", "p", "8080", "api server port")
	com.PersistentFlags().StringP("host", "H", "127.0.0.1", "api server host")
	com.PersistentFlags().Int("write-timeout", 15, "api server write timeout in seconds")
	com.PersistentFlags().Int("read-timeout", 15, "api server read timeout in seconds")
	com.PersistentFlags().Int("idle-timeout", 60, "api server idle timeout in seconds")
	com.PersistentFlags().Int("graceful-timeout", 15, "duration for which the server gracefully wait for existing connections to finish in seconds")
	com.PersistentFlags().Bool("debug-mode", false, "debug mode status")
	com.PersistentFlags().String("google-project-id", "", "google project id")
	com.PersistentFlags().String("unleash-sql-instance-id", "", "google sql instance id")

	viper.BindPFlag("bifrost_port", com.PersistentFlags().Lookup("port"))
	viper.BindPFlag("bifrost_host", com.PersistentFlags().Lookup("host"))
	viper.BindPFlag("bifrost_write_timeout", com.PersistentFlags().Lookup("write-timeout"))
	viper.BindPFlag("bifrost_read_timeout", com.PersistentFlags().Lookup("read-timeout"))
	viper.BindPFlag("bifrost_idle_timeout", com.PersistentFlags().Lookup("idle-timeout"))
	viper.BindPFlag("bifrost_graceful_timeout", com.PersistentFlags().Lookup("graceful-timeout"))
	viper.BindPFlag("bifrost_debug_mode", com.PersistentFlags().Lookup("debug-mode"))
	viper.BindPFlag("bifrost_google_project_id", com.PersistentFlags().Lookup("google-project-id"))
	viper.BindPFlag("bifrost_unleash_sql_instance_id", com.PersistentFlags().Lookup("unleash-sql-instance-id"))
}

func New() *Config {
	if viper.GetString("bifrost_google_project_id") == "" {
		panic("bifrost_google_project_id is not set")
	}

	if viper.GetString("bifrost_unleash_sql_instance_id") == "" {
		panic("bifrost_unleash_sql_instance_id is not set")
	}

	return &Config{
		Server: ServerConfig{
			Port:            viper.GetString("bifrost_port"),
			Host:            viper.GetString("bifrost_host"),
			WriteTimeout:    viper.GetInt("bifrost_write_timeout"),
			ReadTimeout:     viper.GetInt("bifrost_read_timeout"),
			IdleTimeout:     viper.GetInt("bifrost_idle_timeout"),
			GracefulTimeout: viper.GetInt("bifrost_graceful_timeout"),
		},
		DebugMode: viper.GetBool("bifrost_debug_mode"),
		Google: GoogleConfig{
			ProjectID: viper.GetString("bifrost_google_project_id"),
		},
		Unleash: UnleashConfig{
			SQLInstanceID: viper.GetString("bifrost_unleash_sql_instance_id"),
		},
	}
}
