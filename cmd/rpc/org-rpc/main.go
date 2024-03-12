package main

import (
	"flag"
	"github.com/OpenIMSDK/chat/internal/rpc/organization"
	"github.com/OpenIMSDK/chat/pkg/common/chatrpcstart"
	"github.com/OpenIMSDK/chat/pkg/common/config"
	"github.com/OpenIMSDK/tools/log"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config_folder_path", "../config/config.yaml", "Config full path")

	var rpcPort int

	flag.IntVar(&rpcPort, "port", 30500, "get rpc ServerPort from cmd")

	flag.Parse()
	if err := config.InitConfig(configFile); err != nil {
		panic(err)
	}
	if err := log.InitFromConfig("chat.log", "organization-rpc", *config.Config.Log.RemainLogLevel, *config.Config.Log.IsStdout, *config.Config.Log.IsJson, *config.Config.Log.StorageLocation, *config.Config.Log.RemainRotationCount, *config.Config.Log.RotationTime); err != nil {
		panic(err)
	}
	err := chatrpcstart.Start(rpcPort, config.Config.RpcRegisterName.OpenImOrganizationName, 0, organization.Start)
	if err != nil {
		panic(err)
	}
}
