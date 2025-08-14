package main

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"event/conf"

	"github.com/arwoosa/vulpes/log"
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

	// Initialize basic vulpes logger first (will be reconfigured later)
	log.SetConfig(log.WithDev(true))

	// Load configuration using the existing config package
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Error("Error reading config file", log.Err(err))
		os.Exit(1)
	}

	// Unmarshal into our AppConfig struct
	appConfig = &conf.AppConfig{}
	if err := viper.Unmarshal(appConfig); err != nil {
		log.Error("Error unmarshaling config", log.Err(err))
		os.Exit(1)
	}

	// Set timezone
	loc, err := time.LoadLocation(appConfig.TimeZone)
	if err != nil {
		log.Error("Error loading timezone", log.Err(err))
		os.Exit(1)
	}
	time.Local = loc

	// Reconfigure log with proper settings
	isDev := appConfig.Mode == "dev"
	log.SetConfig(
		log.WithDev(isDev),
		log.WithLevel(appConfig.LogConfig.Level),
	)

	log.Info("Configuration loaded successfully", log.String("config_file", viper.ConfigFileUsed()))
}

// GetAppConfig returns the loaded application configuration
func GetAppConfig() *conf.AppConfig {
	return appConfig
}
