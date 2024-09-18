package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/thesoulless/watchmyback/services/email"
)

type EmailStatus int

const (
	Unknown EmailStatus = iota + 100
	ClientError
	SearchFound
	SearchNotFound
	ArchiveError
)

func emailCommand(args []string) (string, int) {
	flags := pflag.NewFlagSet("email", pflag.ExitOnError)
	// flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.BoolVarP(&status, "status", "s", false, "just exit with status code")
	flags.BoolVarP(&archive, "archive", "a", false, "archive the affected email(s)")
	flags.Parse(args)

	args = flags.Args()

	service := args[0]
	command := args[1]
	query := args[2]

	srv, ok := emailSrvs.Get(service)
	if !ok {
		log.Error("email service not found", "service", service)
		return "error: email service not found", int(Unknown)
	}

	switch command {
	case "search":
		res, seqnums, err := srv.Search(query)
		if err != nil {
			if errors.Is(err, email.ErrNotFound) {
				info := fmt.Sprintf("%s: %s\n", "not found", err.Error())
				return info, int(SearchNotFound)
			}

			info := fmt.Sprintf("%s: %s\n", "failed to search", err.Error())
			return info, int(ClientError)
		}

		if archive {
			err = srv.Archive(seqnums)
			if err != nil {
				info := fmt.Sprintf("%s: %s\n", "failed to archive", err.Error())
				return info, int(ArchiveError)
			}
		}

		fmt.Println(args)
		if status {
			info := fmt.Sprintf("status%v\n", res)
			return info, int(SearchFound)
		}

		fmt.Printf("emailCommand:\n%v\n", strings.Join(res, "\n"))
		info := fmt.Sprintf("%v\n", strings.Join(res, "\n"))
		return info, int(SearchFound)
	default:
		return "Unknown command", int(Unknown)
	}
}
