package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
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

func Read(conn net.Conn) ([]byte, error) {
	headerLen := make([]byte, 8)
	_, err := io.ReadFull(conn, headerLen)
	if err != nil {
		// @TODO: use a named status code
		return nil, err
	}

	lengthBytes := make([]byte, 4)
	_, err = io.ReadFull(conn, lengthBytes)
	if err != nil {
		fmt.Println("Error reading length (Read):", err)
		// @TODO: use a named status code
		return nil, err
	}

	// Convert the length prefix to an integer
	length := binary.BigEndian.Uint32(lengthBytes)

	response := make([]byte, length)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		fmt.Println("Error reading:", err)
		// @TODO: use a named status code
		return nil, err
	}

	return response, nil
}

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
