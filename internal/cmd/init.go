package cmd

import (
	"fmt"
	"os"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create fuse.yml and .fuse/spend.db",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.WriteDefault(config.DefaultPath); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Created fuse.yml")
		if err := os.MkdirAll(".fuse", 0o755); err != nil {
			return err
		}
		db, err := store.Open(store.DefaultDBPath)
		if err != nil {
			return err
		}
		defer db.Close()
		fmt.Fprintln(cmd.OutOrStdout(), "Created .fuse/spend.db")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
