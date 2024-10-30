package main

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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

var (
	ErrUnknown = errors.New("unknown error")
	ErrClient  = errors.New("client error")
	ErrSearch  = errors.New("search error")
	ErrArchive = errors.New("archive error")
)

func emailCommand(args []string) (string, int) {
	service := args[0]
	command := args[1]
	query := args[2]

	log.Debug("email command", "service", service, "command", command, "query", query)

	conf, err := readConfig(cfgFile)
	if err != nil {
		return "", int(Unknown)
	}

	var srv *email.Core
	for _, e := range conf.Emails {
		if e.Name != service {
			continue
		}
		if debug {
			l := slog.LevelDebug
			e.LogLevel = &l
		}
		srv, err = email.New(e)
		if err != nil {
			return "", int(Unknown)
		}
		break
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

		if status {
			info := fmt.Sprintf("status%v\n", res)
			return info, int(SearchFound)
		}

		if seqs {
			info := fmt.Sprintf("%v", seqnums)
			info = strings.Trim(info, "[]")
			info = strings.ReplaceAll(info, " ", "\n")
			return info, int(SearchFound)
		}

		info := strings.Join(res, "\n")
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

		info := fmt.Sprintf("%v\n", res)
		return info, int(OK)
	default:
		return "Unknown command", int(Unknown)
	}
}
