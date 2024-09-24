package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	from    string
	status  bool
	seqs    bool
	archive bool
	read    bool

	rootCmd = &cobra.Command{
		Use:              "wmb",
		TraverseChildren: true,
		Short:            "A tool to automate some of my tasks",
		Long:             `By default, it will run the daemon command`,
		Run: func(cmd *cobra.Command, args []string) {
			err := runDaemon(cfgFile)
			if err != nil {
				log.Error("failed to run daemon", "error", err)
				os.Exit(1)
			}
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

	slackCmd = &cobra.Command{
		Use:   "slack",
		Short: "Interact with slack api",
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
	emailCmd.Flags().BoolVar(&seqs, "seqs", false, "print sequence numbers")
	emailCmd.Flags().BoolVarP(&archive, "archive", "a", false, "archive the affected email(s)")
	emailCmd.Flags().StringVarP(&from, "from", "f", "", "from email address")
	emailCmd.Flags().BoolVarP(&read, "read", "r", false, "read from stdin")

	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(slackCmd)
	rootCmd.AddCommand(emailCmd)
}
