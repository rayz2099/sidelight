package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sidelight [command]",
		Short: "SideLight is an AI-powered color grading tool for RAW photos",
		Long: `SideLight helps you speed up your photography workflow.

It offers three main tools:
1. grade: Extracts preview, analyzes it with AI, and generates an XMP sidecar for editing.
2. frame: Generates a beautiful preview image with a border and EXIF data.
3. export: Quickly extracts the embedded full-size JPEG preview from RAW files.`,
	}

	cobra.OnInitialize(initConfig)

	// 支持 -c 和 --config 两种方式指定配置文件
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default searches: ./config.json, ~/.config/sidelight/config.json)")

	// Register Subcommands
	rootCmd.AddCommand(gradeCmd)
	rootCmd.AddCommand(frameCmd)
	rootCmd.AddCommand(exportCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		// 使用命令行指定的配置文件
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read config file '%s': %v\n", cfgFile, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		// 按优先级搜索配置文件
		viper.SetConfigType("json")
		viper.SetConfigName("config")

		// 1. 当前工作目录
		viper.AddConfigPath(".")

		// 2. 可执行文件所在目录
		if exePath, err := os.Executable(); err == nil {
			viper.AddConfigPath(fmt.Sprintf("%s", exePath[:len(exePath)-len("/sidelight")]))
		}

		// 3. 用户配置目录 ~/.config/sidelight
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(home + "/.config/sidelight")
		}

		// 尝试读取配置文件（不是强制的）
		if err := viper.ReadInConfig(); err == nil {
			fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
		}
		// 如果没找到配置文件，不报错，允许使用环境变量和命令行参数
	}

	// 绑定环境变量（优先级最低）
	viper.AutomaticEnv()
}
