package main

import (
	"Echo/internel/config"
	"Echo/internel/provider"
	"Echo/internel/tui"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

/*
echoai echoai
echoai config
*/

var rootCmd = &cobra.Command{
	Use:   "echoai",
	Short: `Echo Ai CLI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		if err := c.Validate(); err != nil {
			return err
		}
		client, err := provider.New(provider.Options{
			Provider:   c.Provider,
			APIKey:     c.APIKey,
			BaseURL:    c.BaseURL,
			Model:      c.Model,
			APIVersion: c.APIVersion,
		})
		if err != nil {
			return err
		}
		return tui.Run(client)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置 API Key、BaseURL 和模型",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)
		prompt := func(label, old string) string {
			if old != "" {
				fmt.Printf("%s [回车保留 %s]: ", label, old)
			} else {
				fmt.Printf("%s: ", label)
			}
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			if text == "" {
				return old
			}
			return text
		}

		c.APIKey = prompt("API Key", c.APIKey)
		c.BaseURL = prompt("Base URL (如 https://api.deepseek.com/v1)", c.BaseURL)
		c.Model = prompt("Model (如 deepseek-chat)", c.Model)

		if err := c.Save(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
