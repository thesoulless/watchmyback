package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func runClient(args []string) {
	conn, err := dial()
	if err != nil {
		fmt.Println("Error connecting to daemon:", err)
		return
	}
	defer conn.Close()

	fmt.Fprintf(conn, strings.Join(args, " ")+"\n")

	headerLen := make([]byte, 8)
	_, err = io.ReadFull(conn, headerLen)
	if err != nil {
		fmt.Println("Error reading response:", err)
		// @TODO: use a named status code
		os.Exit(200)
	}

	lengthBytes := make([]byte, 4)
	_, err = io.ReadFull(conn, lengthBytes)
	if err != nil {
		fmt.Println("Error reading response:", err)
		// @TODO: use a named status code
		os.Exit(200)
	}

	// Convert the length prefix to an integer
	length := binary.BigEndian.Uint32(lengthBytes)

	response := make([]byte, length)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		fmt.Println("Error reading response:", err)
		// @TODO: use a named status code
		os.Exit(200)
	}

	if status {
		ex, err := strconv.Atoi(strings.TrimSpace(string(response)))
		if err != nil {
			// @TODO: use a named status code
			os.Exit(200)
		}
		os.Exit(ex)
		return
	}

	fmt.Print(string(response))
}
