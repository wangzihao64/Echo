package main

import (
	"Echo/internel/tui"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "echoai",
	Short: `Echo Ai CLI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run() // 启动交互式输入循环
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
