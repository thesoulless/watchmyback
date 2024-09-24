package main

import (
	"errors"
	"fmt"
	"strconv"
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
	OK
	ReadOK
	ArchiveOK
	ArchiveError
)

func emailCommand(args []string) (string, int) {
	flags := pflag.NewFlagSet("email", pflag.ExitOnError)
	// flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.BoolVarP(&status, "status", "s", false, "just exit with status code")
	flags.BoolVar(&seqs, "seqs", false, "print sequence numbers")
	flags.StringVarP(&from, "from", "f", "", "from email address")
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
		res, seqnums, err := srv.Search(query, from)
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
				info := fmt.Sprintf("%s %v: %s\n", "failed to archive", seqnums, err.Error())
				return info, int(ArchiveError)
			}
		}

		fmt.Println(args)
		if status {
			info := fmt.Sprintf("status%v\n", res)
			return info, int(SearchFound)
		}

		if seqs {
			seqnumsStr := fmt.Sprintf("%v", seqnums)
			seqnumsStr = strings.Trim(seqnumsStr, "[]")
			seqnumsStr = strings.ReplaceAll(seqnumsStr, " ", "\n")
			info := fmt.Sprintf("%s\n", seqnumsStr)
			return info, int(SearchFound)
		}

		fmt.Printf("emailCommand:\n%v\n", strings.Join(res, "\n"))
		info := fmt.Sprintf("%v\n", strings.Join(res, "\n"))
		return info, int(SearchFound)
	case "inbox":
		seqnum, err := strconv.Atoi(query)
		if err != nil {
			info := fmt.Sprintf("%s: %s\n", "invalid sequence number", err.Error())
			return info, int(ClientError)
		}

		err = srv.Move([]uint32{uint32(seqnum)}, "INBOX")
		if err != nil {
			if errors.Is(err, email.ErrNotFound) {
				info := fmt.Sprintf("%s: %s\n", "not found", err.Error())
				return info, int(SearchNotFound)
			}

			info := fmt.Sprintf("%s: %s\n", "failed to move", err.Error())
			return info, int(ClientError)
		}

		if status {
			info := fmt.Sprintf("status%v\n", "OK")
			return info, int(OK)
		}

		// fmt.Printf("emailCommand:\n%v\n", res)
		info := fmt.Sprintf("%v\n", "OK")
		return info, int(OK)
	case "archive":
		seqnum, err := strconv.Atoi(query)
		if err != nil {
			info := fmt.Sprintf("%s: %s\n", "invalid sequence number", err.Error())
			return info, int(ClientError)
		}
		err = srv.Archive([]uint32{uint32(seqnum)})
		if err != nil {
			if errors.Is(err, email.ErrNotFound) {
				info := fmt.Sprintf("%s: %s\n", "not found", err.Error())
				return info, int(ArchiveError)
			}

			info := fmt.Sprintf("%s: %s\n", "failed to archive", err.Error())
			return info, int(ClientError)
		}

		if status {
			info := fmt.Sprintf("status%v\n", "OK")
			return info, int(OK)
		}

		// fmt.Printf("emailCommand:\n%v\n", res)
		info := fmt.Sprintf("%v\n", "OK")
		return info, int(OK)
	case "read":
		seqnum, err := strconv.Atoi(query)
		if err != nil {
			info := fmt.Sprintf("%s: %s\n", "invalid sequence number", err.Error())
			return info, int(ClientError)
		}

		res, err := srv.Body(uint32(seqnum))
		if err != nil {
			if errors.Is(err, email.ErrNotFound) {
				info := fmt.Sprintf("%s: %s\n", "not found", err.Error())
				return info, int(SearchNotFound)
			}

			info := fmt.Sprintf("%s: %s\n", "failed to read", err.Error())
			return info, int(ClientError)
		}

		if archive {
			err = srv.Archive([]uint32{uint32(seqnum)})
			if err != nil {
				info := fmt.Sprintf("%s: %s\n", "failed to archive", err.Error())
				return info, int(ArchiveError)
			}
		}

		if status {
			info := fmt.Sprintf("status%v\n", res)
			return info, int(SearchFound)
		}

		// fmt.Printf("emailCommand:\n%v\n", res)
		info := fmt.Sprintf("%v\n", res)
		return info, int(OK)
	default:
		return "Unknown command", int(Unknown)
	}
}
