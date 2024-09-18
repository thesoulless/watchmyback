package main

import (
	"context"

	"github.com/spf13/pflag"
	"github.com/thesoulless/watchmyback/services/slack"
)

func runSlack(args []string) string {
	log.Info("slack command (runSlack)", "args", args)
	flags := pflag.NewFlagSet("slack", pflag.ExitOnError)
	// flags.ParseErrorsWhitelist.UnknownFlags = true
	// flags.BoolVarP(&status, "status", "s", false, "just exit with status code")
	flags.Parse(args)

	args = flags.Args()
	log.Info("args", "args", args)

	command := args[0]
	uri := args[1]
	msg := args[2]

	switch command {
	case "webhook":
		log.Info("sending message to slack", "uri", uri, "msg", msg)
		err := slack.SendToChanel(context.Background(), uri, msg)
		if err != nil {
			return err.Error()
		}
	default:
		log.Error("unknown command", "command", command)
		return "unknown command"
	}

	return "ok"
}
