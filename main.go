package main

import (
	"context"
	"fmt"
	"github.com/CharellKing/ela-lib/config"
	"github.com/CharellKing/ela-lib/service/gateway"
	"github.com/CharellKing/ela-lib/service/task"
	"github.com/CharellKing/ela-lib/utils"
	goflags "github.com/jessevdk/go-flags"
	"log"

	"github.com/spf13/viper"
)

func main() {
	var err error
	cmd := &Cmd{}

	// parse args
	_, err = goflags.Parse(cmd)
	if err != nil {
		log.Fatal(err)
		return
	}

	if cmd.ConfigFile == "" {
		fmt.Println("Usage: go run main.go --config <config_path> [--gateway <gateway>] [--tasks <tasks>]")
		return
	}

	viper.SetConfigFile(cmd.ConfigFile)
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

	utils.InitLogger(&cfg)

	ctx := context.Background()

	if cmd.Tasks {
		taskMgr, err := task.NewTaskMgr(&cfg)
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

	if cmd.Gateway {
		esProxy, err := gateway.NewESGateway(&cfg)
		if err != nil {
			utils.GetLogger(ctx).Errorf("create task manager %+v", err)
			return
		}
		esProxy.Run()
	}

	return
}
