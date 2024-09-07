package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"

	ulua "github.com/thesoulless/watchmyback/internal/lua"
	"github.com/thesoulless/watchmyback/services/email"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	mu         = sync.Mutex{}
	configFile string
	conn       net.Conn
	emailSrvs  = make([]*email.Core, 0)

	cfgFile     string
	status      bool
	userLicense string

	rootCmd = &cobra.Command{
		Use:              "cobra-cli",
		TraverseChildren: true,
		Short:            "A generator for Cobra based Applications",
		Long: `Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			runClient(os.Args[1:])
		},
	}

	daemonCmd = &cobra.Command{
		Use:   "daemon",
		Short: "Print the version number of Hugo",
		Long:  `All software has versions. This is Hugo's`,
		Run: func(cmd *cobra.Command, args []string) {
			runDaemon(configFile)
		},
	}
)

type Instance struct {
	emailSrvs []*email.Core
}

func init() {
	ulua.L = lua.NewState()
	ulua.L.SetGlobal("import", luar.New(ulua.L, LuaImport))
	ulua.L.SetGlobal("require", luar.New(ulua.L, LuaImport))
	// ulua.InternalLibs()

	// cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&status, "status", "s", false, "just exit with status code")
	// rootCmd.PersistentFlags().Bool("daemon", false, "")
	// rootCmd.PersistentFlags().StringVarP(&userLicense, "license", "l", "", "name of license for the project")
	// rootCmd.PersistentFlags().Bool("viper", true, "use Viper for configuration")
	// viper.BindPFlag("author", rootCmd.PersistentFlags().Lookup("author"))
	// viper.BindPFlag("useViper", rootCmd.PersistentFlags().Lookup("viper"))
	// viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
	// viper.SetDefault("license", "apache")
	// rootCmd.PersistentFlags().Parse()

	daemonCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "daemon's config")
	rootCmd.AddCommand(daemonCmd)
	// rootCmd.AddCommand(defaultCmd)
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()

	/*if err := rootCmd.Execute(); err != nil {
	  fmt.Fprintln(os.Stderr, err)
	  os.Exit(1)
	}*/
}

func main() {
	Execute()

	/*
		configFile := pflag.StringP("config", "c", "config.yaml", "config file")
		daemon := pflag.BoolP("daemon", "d", false, "run as daemon")
		nf := pflag.NewFlagSet("name", pflag.ExitOnError)
		nf.Parse(os.Args[1:])

		pflag.Parse()

		if !*daemon {
			runClient()
			return
		}

		runDaemon(*configFile)
		defer conn.Close()

		defer func(es []*email.Core) {
			for _, e := range es {
				e.Close()
			}
		}(emailSrvs)
	*/

	// defer ulua.L.Close()
	// if err := ulua.L.DoFile("hello.lua"); err != nil {
	// 	panic(err)
	// }
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

func runDaemon(configFile string) {
	conf := readConfig(configFile)

	if len(conf.Emails) > 0 {
		for _, e := range conf.Emails {
			emailSrv, err := email.New(e)
			if err != nil {
				log.Println(err)
				continue
			}
			emailSrvs = append(emailSrvs, emailSrv)
			go emailSrv.Run()
		}
	}

	defer func(es []*email.Core) {
		for _, e := range es {
			e.Close()
		}
	}(emailSrvs)

	listener, err := listen()
	if err != nil {
		fmt.Println("Error listening:", err)
		return
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
}

func handleConnection(conn net.Conn) {
	// Handle the connection (implementation omitted for brevity)
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
			// conn.Close()
			continue
		}
		caslice := strings.Split(commandArgs, " ")
		var args []string
		command := caslice[0]
		command = strings.TrimSpace(command)

		if len(caslice) > 1 {
			args = caslice[1:]
		}

		var response string
		mu.Lock()
		switch command {
		case "email":
			response = emailCommand(args)
		case "len":
			response = fmt.Sprintf("%s %s: %d\n", command, strings.Join(args, " "), len(emailSrvs))
		default:
			response = "Unknown command\n"
		}
		mu.Unlock()

		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println("Error writing response:", err)
			// conn.Close()
			return
		}
	}
}

func runClient(args []string) {
	conn, err := dial()
	if err != nil {
		fmt.Println("Error connecting to daemon:", err)
		return
	}
	defer conn.Close()

	fmt.Fprintf(conn, strings.Join(args, " ")+"\n")
	response, _ := bufio.NewReader(conn).ReadString('\n')
	fmt.Print("Response from daemon: ", response)
}

func LuaImport(pkg string) *lua.LTable {
	return ulua.Import(pkg)
}

type T struct {
	Emails []email.Conf `yaml:"email"`
}

func readConfig(conf string) T {
	t := T{}

	f, err := os.Open(conf)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal([]byte(data), &t)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("--- t:\n%v\n\n", t)
	return t
}

func emailCommand(args []string) string {
	// account := args[0]
	// op := args[1]

	res, err := emailSrvs[0].Search("Testing")
	if err != nil {
		info := fmt.Sprintf("%s: %s\n", "failed to search", err.Error())
		return info
	}

	fmt.Println(args)
	if args[0] == "--status" {
		info := fmt.Sprintf("status%v\n", res)
		return info
	}

	info := fmt.Sprintf("%v\n", strings.Join(res, "\n"))
	return info
}
