package cmd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/money"
	"github.com/hashir500/Fuse/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current spend vs. limits",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.DefaultPath)
		if err != nil {
			return err
		}
		db, err := store.Open(store.DefaultDBPath)
		if err != nil {
			return err
		}
		defer db.Close()

		spend, err := db.PeriodSpend(cmd.Context(), time.Now())
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Today:   %s / %s  %s %s\n", money.Dollars(spend.Daily), money.Dollars(cfg.Budgets.Daily.Hard), bar(spend.Daily, cfg.Budgets.Daily.Hard), percent(spend.Daily, cfg.Budgets.Daily.Hard))
		fmt.Fprintf(out, "Week:    %s / %s  %s %s\n", money.Dollars(spend.Weekly), money.Dollars(cfg.Budgets.Weekly.Hard), bar(spend.Weekly, cfg.Budgets.Weekly.Hard), percent(spend.Weekly, cfg.Budgets.Weekly.Hard))
		fmt.Fprintf(out, "Month:   %s / %s  %s %s\n", money.Dollars(spend.Monthly), money.Dollars(cfg.Budgets.Monthly.Hard), bar(spend.Monthly, cfg.Budgets.Monthly.Hard), percent(spend.Monthly, cfg.Budgets.Monthly.Hard))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func percent(value, max float64) string {
	if max <= 0 {
		return "0%"
	}
	pct := (value / max) * 100
	if pct > 0 && pct < 10 {
		return fmt.Sprintf("%.1f%%", pct)
	}
	return fmt.Sprintf("%.0f%%", math.Round(pct))
}

func bar(value, max float64) string {
	const width = 10
	if max <= 0 {
		return strings.Repeat(".", width)
	}
	filled := int(math.Round((value / max) * width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat(".", width-filled)
}
