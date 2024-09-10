package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"

	"github.com/thesoulless/watchmyback/services/email"
)

type ExitCode int

const (
	ExitCodeOK ExitCode = iota
	ExitCodeError
)

var (
	mu   = sync.Mutex{}
	conn net.Conn
)

func Write(conn net.Conn, response string) error {
	header := make([]byte, 8)
	copy(header[:3], Magic)
	copy(header[3:], Version)

	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(response)))

	_, err := conn.Write(header)
	if err != nil {
		return err
	}

	_, err = conn.Write(lengthBytes)
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte(response))
	if err != nil {
		return err
	}

	return nil
}

func runDaemon(configFile string) error {
	conf, err := readConfig(configFile)
	if err != nil {
		return err
	}

	if len(conf.Emails) > 0 {
		for _, e := range conf.Emails {
			emailSrv, err := email.New(e)
			if err != nil {
				log.Error("fail to email.New", "error", err)
				continue
			}
			emailSrvs.Add(e.Name, emailSrv)
			go emailSrv.Run()
		}
	}

	defer func(es *List[*email.Core]) {
		for n := es.head; n != nil; n = n.next {
			n.val.Close()
		}
	}(emailSrvs)

	listener, err := listen()
	if err != nil {
		return fmt.Errorf("failed listening: %v", err)
	}
	defer listener.Close()

	fmt.Println("Daemon is running...")
	for {
		conn, err = listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}

	fmt.Println("Daemon is shutting down...")

	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		commandArgs, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		commandArgs = strings.TrimSpace(commandArgs)
		if len(commandArgs) < 1 {
			conn.Write([]byte("invalid operation, run --help for more help\n"))
			continue
		}
		caslice := strings.Split(commandArgs, " ")
		var args []string
		command := caslice[0]
		command = strings.TrimSpace(command)

		if len(caslice) > 1 {
			args = caslice[1:]
		}

		var (
			response string
			ex       int
		)
		mu.Lock()
		switch command {
		case "email":
			response, ex = emailCommand(args)
		case "len":
			response = fmt.Sprintf("%s %s: %d\n", command, strings.Join(args, " "), emailSrvs.Len())
		default:
			response = "Unknown command\n"
		}
		mu.Unlock()

		if slices.Contains(args, "--status") {
			msg := fmt.Sprintf("%d", ex)
			err = Write(conn, msg)
			if err != nil {
				fmt.Println("Error writing response:", err)
				return
			}
			return
		}

		err = Write(conn, response)
		if err != nil {
			fmt.Println("Error writing response:", err)
			return
		}
	}
}
