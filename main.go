package main

import (
	"context"
	"fmt"
	"github.com/CharellKing/ela/config"
	"github.com/CharellKing/ela/service"
	"github.com/CharellKing/ela/utils"
	"github.com/spf13/viper"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <config_path>")
		return
	}

	configPath := os.Args[1]
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Unable reading config file, %v\n", err)
		return
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Printf("Unable to decode into struct, %v\n", err)
		return
	}

	ctx := context.Background()
	taskMgr, err := service.NewTaskMgr(&cfg)
	if err != nil {
		utils.GetLogger(ctx).WithError(err).Error("create task manager")
		return
	}
	if err := taskMgr.Run(ctx); err != nil {
		utils.GetLogger(ctx).WithError(err).Error("run task manager")
		return
	}

	return
}
