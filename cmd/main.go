package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"runtime"

	"github.com/thesoulless/watchmyback/services/email"

	// luar "layeh.com/gopher-luar"

	"gopkg.in/yaml.v3"
)

var (
	log = slog.New(slog.NewTextHandler(os.Stderr, nil))
)

func init() {
	// ulua.L = lua.NewState()
	// ulua.L.SetGlobal("import", luar.New(ulua.L, LuaImport))
	// ulua.L.SetGlobal("require", luar.New(ulua.L, LuaImport))
	// // ulua.InternalLibs()
}

// Execute executes the root command.
func run(ctx context.Context) error {
	Init()
	rootCmd.SetContext(ctx)
	return rootCmd.Execute()
}

func main() {
	ctx := context.Background()
	err := run(ctx)
	if err != nil {
		log.Error("faild to run", "error", err)
	}
}

const (
	unixSocket = "/tmp/app.sock"
	tcpPort    = "127.0.0.1:8080"
)

func getAddress() string {
	if runtime.GOOS == "windows" {
		return tcpPort
	}
	return unixSocket
}

func listen() (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return net.Listen("tcp", tcpPort)
	}
	os.Remove(unixSocket)
	return net.Listen("unix", unixSocket)
}

func dial() (net.Conn, error) {
	if runtime.GOOS == "windows" {
		return net.Dial("tcp", tcpPort)
	}
	return net.Dial("unix", unixSocket)
}

var (
	Magic   = []byte("WMB")
	Version = []byte("1.0")
)

// func LuaImport(pkg string) *lua.LTable {
// 	return ulua.Import(pkg)
// }

type T struct {
	Emails []email.Conf `yaml:"email"`
}

func readConfig(conf string) (T, error) {
	t := T{}

	f, err := os.Open(conf)
	if err != nil {
		return t, fmt.Errorf("failed to readConfig os.Open: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return t, fmt.Errorf("failed to readConfig io.ReadAll: %w", err)
	}
	err = yaml.Unmarshal([]byte(data), &t)
	if err != nil {
		return t, fmt.Errorf("failed to readConfig yaml.Unmarshal: %w", err)
	}

	fmt.Printf("--- t:\n%v\n\n", t)
	return t, nil
}
