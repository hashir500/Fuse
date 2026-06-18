package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fuse",
	Short: "Set a fuse on your AI spending",
	Long:  "Fuse is a lightweight local proxy that tracks AI API spend and enforces hard budget caps.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
