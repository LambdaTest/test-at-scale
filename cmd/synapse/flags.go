package main

import (
	"github.com/spf13/cobra"
)

// AttachCLIFlags attaches command line flags to command
func AttachCLIFlags(rootCmd *cobra.Command) error {
	rootCmd.PersistentFlags().StringP("config", "c", "", "the config file to use")
	rootCmd.PersistentFlags().BoolP("verbose", "", false, "should every proxy request be logged to stdout")
	return nil
}
