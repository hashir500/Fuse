package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/proxy"
	"github.com/hashir500/Fuse/internal/store"
	"github.com/spf13/cobra"
)

var proxyAddr string

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start the local Fuse proxy",
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
		fmt.Fprintf(cmd.OutOrStdout(), "Fuse proxy running on %s\n", proxyAddr)
		fmt.Fprintf(cmd.OutOrStdout(), "   Budgets: $%.2f/$%.2f daily | $%.2f/$%.2f weekly | $%.2f/$%.2f monthly\n",
			cfg.Budgets.Daily.Soft, cfg.Budgets.Daily.Hard,
			cfg.Budgets.Weekly.Soft, cfg.Budgets.Weekly.Hard,
			cfg.Budgets.Monthly.Soft, cfg.Budgets.Monthly.Hard)
		fmt.Fprintf(cmd.OutOrStdout(), "   Today: $%.2f / $%.2f\n", spend.Daily, cfg.Budgets.Daily.Hard)

		server := &proxy.Server{Config: cfg, Store: db, Stderr: os.Stderr}
		return server.ListenAndServe(proxyAddr)
	},
}

func init() {
	proxyCmd.Flags().StringVar(&proxyAddr, "addr", "localhost:8787", "proxy listen address")
	rootCmd.AddCommand(proxyCmd)
}
