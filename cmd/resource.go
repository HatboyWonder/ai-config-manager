package cmd

import "github.com/spf13/cobra"

// resourceCmd represents the resource command group.
var resourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "Validate and inspect individual resources",
	Long: `Validate and inspect individual resources.

The resource command group provides resource-scoped workflows that do not
require mutating repository state.`,
}

func init() {
	rootCmd.AddCommand(resourceCmd)
}
