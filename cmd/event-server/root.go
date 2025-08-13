package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"event/conf"
	vulpeslog "github.com/arwoosa/vulpes/log"
)

var (
	cfgFile   string
	appConfig *conf.AppConfig
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "event-server",
	Short: "Event microservice with console and public APIs",
	Long: `Event microservice provides both console (management) and public APIs.
	
Use 'console' command to start the internal management service.
Use 'public' command to start the public-facing API service.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags for all commands
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "conf/config.yaml", "config file path")
}

// initConfig reads in config file and sets up the application configuration
func initConfig() {
	var err error

	// Load configuration using the existing config package
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal into our AppConfig struct
	appConfig = &conf.AppConfig{}
	if err := viper.Unmarshal(appConfig); err != nil {
		fmt.Printf("Error unmarshaling config: %v\n", err)
		os.Exit(1)
	}

	// Set timezone
	loc, err := time.LoadLocation(appConfig.TimeZone)
	if err != nil {
		fmt.Printf("Error loading timezone: %v\n", err)
		os.Exit(1)
	}
	time.Local = loc

	// Initialize vulpes logger
	isDev := appConfig.Mode == "dev"
	vulpeslog.SetConfig(
		vulpeslog.WithDev(isDev),
		vulpeslog.WithLevel(appConfig.LogConfig.Level),
	)

	vulpeslog.Info("Configuration loaded successfully", vulpeslog.String("config_file", viper.ConfigFileUsed()))
}

// GetAppConfig returns the loaded application configuration
func GetAppConfig() *conf.AppConfig {
	return appConfig
}
