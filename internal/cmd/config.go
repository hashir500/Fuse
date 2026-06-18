package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit or validate fuse.yml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.DefaultPath)
		if err != nil {
			return err
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprintln(cmd.OutOrStdout(), "Press enter to keep the current value.")
		cfg.Budgets.Daily.Soft = promptFloat(reader, cmd, "Daily soft", cfg.Budgets.Daily.Soft)
		cfg.Budgets.Daily.Hard = promptFloat(reader, cmd, "Daily hard", cfg.Budgets.Daily.Hard)
		cfg.Budgets.Weekly.Soft = promptFloat(reader, cmd, "Weekly soft", cfg.Budgets.Weekly.Soft)
		cfg.Budgets.Weekly.Hard = promptFloat(reader, cmd, "Weekly hard", cfg.Budgets.Weekly.Hard)
		cfg.Budgets.Monthly.Soft = promptFloat(reader, cmd, "Monthly soft", cfg.Budgets.Monthly.Soft)
		cfg.Budgets.Monthly.Hard = promptFloat(reader, cmd, "Monthly hard", cfg.Budgets.Monthly.Hard)
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := os.WriteFile(config.DefaultPath, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Updated fuse.yml")
		return nil
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate fuse.yml",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := config.Load(config.DefaultPath); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "fuse.yml is valid")
		return nil
	},
}

func init() {
	configCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(configCmd)
}

func promptFloat(reader *bufio.Reader, cmd *cobra.Command, label string, current float64) float64 {
	for {
		fmt.Fprintf(cmd.OutOrStdout(), "%s [$%.2f]: ", label, current)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return current
		}
		value, err := strconv.ParseFloat(strings.TrimPrefix(line, "$"), 64)
		if err == nil && value >= 0 {
			return value
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Enter a non-negative number.")
	}
}
