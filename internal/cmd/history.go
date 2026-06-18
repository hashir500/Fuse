package cmd

import (
	"fmt"

	"github.com/hashir500/Fuse/internal/store"
	"github.com/spf13/cobra"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show recent requests and costs",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := store.Open(store.DefaultDBPath)
		if err != nil {
			return err
		}
		defer db.Close()

		logs, err := db.Recent(cmd.Context(), historyLimit)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "TIME              PROVIDER   MODEL                         TOKENS    COST      STATUS")
		for _, log := range logs {
			status := "ok"
			if log.WasBlocked {
				status = "blocked"
			}
			fmt.Fprintf(out, "%-16s  %-9s  %-28s  %-8d  $%-7.3f %s\n",
				log.Timestamp.Format("2006-01-02 15:04"),
				log.Provider,
				truncate(log.Model, 28),
				log.TotalTokens,
				log.EstimatedCost,
				status,
			)
		}
		return nil
	},
}

func init() {
	historyCmd.Flags().IntVar(&historyLimit, "limit", 10, "number of requests to show")
	rootCmd.AddCommand(historyCmd)
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}
