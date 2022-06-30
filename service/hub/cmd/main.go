package main

import (
	"strings"

	"github.com/naturalselectionlabs/pregod/common/logger"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/service/hub/internal/config"
	"github.com/naturalselectionlabs/pregod/service/hub/internal/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var rootCommand = cobra.Command{
	Use:           "hub",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       protocol.Version,
}

func main() {
	viper.SetConfigName("hub")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/pregod/")
	viper.AddConfigPath("$HOME/.pregod/")
	viper.AddConfigPath("./deploy/config/")
	// `opentelemetry.host` -> `CONFIG_ENV_OPENTELEMETRY_HOST`
	viper.SetEnvPrefix("CONFIG_ENV")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatalln(err)
	}

	configHub := config.Config{}

	if err := viper.Unmarshal(&configHub); err != nil {
		logrus.Fatalln(err)
	}

	if err := logger.Initialize(string(configHub.Mode)); err != nil {
		logrus.Fatalln(err)
	}

	srv := server.New(&configHub)

	rootCommand.RunE = func(cmd *cobra.Command, args []string) error {
		return srv.Run()
	}

	if err := rootCommand.Execute(); err != nil {
		logger.Global().Fatal("hub execution failed", zap.Error(err))
	}
}
