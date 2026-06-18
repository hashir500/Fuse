package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/money"
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
		fmt.Fprintf(cmd.OutOrStdout(), "   Budgets: %s/%s daily | %s/%s weekly | %s/%s monthly\n",
			money.Dollars(cfg.Budgets.Daily.Soft), money.Dollars(cfg.Budgets.Daily.Hard),
			money.Dollars(cfg.Budgets.Weekly.Soft), money.Dollars(cfg.Budgets.Weekly.Hard),
			money.Dollars(cfg.Budgets.Monthly.Soft), money.Dollars(cfg.Budgets.Monthly.Hard))
		fmt.Fprintf(cmd.OutOrStdout(), "   Today: %s / %s\n", money.Dollars(spend.Daily), money.Dollars(cfg.Budgets.Daily.Hard))

		server := &proxy.Server{Config: cfg, Store: db, Stderr: os.Stderr}
		return server.ListenAndServe(proxyAddr)
	},
}

func init() {
	proxyCmd.Flags().StringVar(&proxyAddr, "addr", "localhost:8787", "proxy listen address")
	rootCmd.AddCommand(proxyCmd)
}
