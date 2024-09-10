package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	status  bool
	archive bool

	rootCmd = &cobra.Command{
		Use:              "cobra-cli",
		TraverseChildren: true,
		Short:            "A generator for Cobra based Applications",
		Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			// runClient(os.Args[1:])
		},
	}

	emailCmd = &cobra.Command{
		Use:   "email",
		Short: "Print the version number of Hugo",
		Long:  `All software has versions. This is Hugo's`,
		Run: func(cmd *cobra.Command, args []string) {
			runClient(os.Args[1:])
		},
	}

	daemonCmd = &cobra.Command{
		Use:   "daemon",
		Short: "Print the version number of Hugo",
		Long:  `All software has versions. This is Hugo's`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runDaemon(cfgFile)
			if err != nil {
				os.Exit(1)
			}
		},
	}
)

func Init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	// rootCmd.PersistentFlags().BoolVarP(&status, "status", "s", false, "just exit with status code")

	emailCmd.Flags().BoolVarP(&status, "status", "s", false, "just exit with status code")
	emailCmd.Flags().BoolVarP(&archive, "archive", "a", false, "archive the affected email(s)")

	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(emailCmd)
}
