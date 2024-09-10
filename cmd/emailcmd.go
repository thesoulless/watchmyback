package main

import (
	"fmt"
	"strings"

	"github.com/thesoulless/watchmyback/services/email"
)

type EmailStatus int

const (
	Unknown EmailStatus = iota + 100
	ClientError
	SearchFound
	SearchNotFound
)

func emailCommand(args []string) (string, int) {
	// @TODO; select the correct mail client

	service := args[0]
	for _, srv := range args {
		if strings.HasPrefix(srv, "--") {
			continue
		}

		service = srv
		break
	}

	srv, ok := emailSrvs.Get(service)
	if !ok {
		log.Error("email service not found", "service", service)
		return "error: email service not found", int(Unknown)
	}

	res, err := srv.Search("Testing")
	if err != nil {
		if err == email.ErrNotFound {
			info := fmt.Sprintf("%s: %s\n", "not found", err.Error())
			return info, int(SearchNotFound)
		}

		info := fmt.Sprintf("%s: %s\n", "failed to search", err.Error())
		return info, int(ClientError)
	}

	fmt.Println(args)
	if args[0] == "--status" {
		info := fmt.Sprintf("status%v\n", res)
		return info, int(SearchFound)
	}

	fmt.Printf("emailCommand:\n%v\n", strings.Join(res, "\n"))
	info := fmt.Sprintf("%v\n", strings.Join(res, "\n"))
	return info, int(SearchFound)
}
