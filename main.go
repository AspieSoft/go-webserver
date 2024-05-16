package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AspieSoft/go-regex/v8"
	"github.com/AspieSoft/goutil/bash"
	"github.com/AspieSoft/goutil/fs/v3"
	"github.com/AspieSoft/goutil/v7"
	"github.com/AspieSoft/webext"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/handlebars/v2"
)

var Config = struct{
	Origin []string
	Proxy []string
	Port []uint16
	AutoTLS bool

	DebugMode bool
}{
	Origin: []string{"localhost"},
	Proxy: []string{"127.0.0.1"},
	Port: []uint16{80, 443},
	AutoTLS: true,

	DebugMode: false,
}

var PWD string = "./website"

var handleApp map[int8][]func(app *fiber.App) = map[int8][]func(app *fiber.App){}

var termCommands map[string]func(args map[string]string) = make(map[string]func(args map[string]string))

func init(){
	PWD = webext.PWD+"/website"
}

func main(){
	// handle config
	fs.ReadConfig(PWD+"/config.yml", &Config)

	if len(Config.Port) == 0 {
		Config.Port = []uint16{80, 443}
	}else if len(Config.Port) == 1 {
		Config.Port = append(Config.Port, 443)
	}

	// handle database
	var runInitDB bool
	if _, err := os.Stat(PWD+"/db/server.db"); err != nil {
		runInitDB = true
		os.MkdirAll(PWD+"/db", webext.TryPerm(0664, 0775))
	}

	var err error
	DB, err = sql.Open("sqlite3", PWD+"/db/server.db")
	if err != nil {
		panic(err)
	}
	defer DB.Close()

	if runInitDB {
		initDB()
	}else{
		// run prepareDB queue
		for _, cb := range prepareDB {
			cb()
		}
	}

	// run afterDB queue
	for _, cb := range afterDB {
		cb()
	}


	// handle fiber app
	engine := handlebars.New(PWD+"/html", ".hbs")
	app := fiber.New(fiber.Config{
		Views: engine,
		ViewsLayout: "layout",
	})

	app.Use(webext.VerifyOrigin(Config.Origin, Config.Proxy))

	if Config.AutoTLS {
		app.Use(webext.RedirectSSL(Config.Port[0], Config.Port[1]))
	}

	app.Static("/assets", PWD+"/assets")
	app.Static("/theme", PWD+"/theme/dist")

	cmd := exec.Command(`./compile`, `0`)
	cmd.Dir = PWD+"/theme"

	if stdin, err := cmd.StdinPipe(); err == nil {
		defer func(){
			if _, err := stdin.Write([]byte("stop\n")); err == nil {
				cmd.Wait()
			}else{
				cmd.Cancel()
			}
		}()
	}else{
		defer cmd.Cancel()
	}

	cmd.Start()

	// handle_app(app)
	for p := int8(-128); p < 127; p++ {
		for _, cb := range handleApp[p] {
			cb(app)
		}
	}
	for _, cb := range handleApp[127] {
		cb(app)
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		fmt.Println("\x1b[1;32mRunning Server!", "\x1b[0m")

		std := bufio.NewReader(os.Stdin)

		blankSize := 50
		if out, err := bash.Run([]string{`stty`, `size`}, "", nil); err == nil {
			s := bytes.SplitN(out, []byte{' '}, 2)
			if len(s) > 1 {
				if i, err := strconv.Atoi(string(s[1])); err == nil {
					blankSize = i
				}
			}
		}

		for {
			buf := make([]byte, 1024)
			fmt.Print("> ")
			std.Read(buf)
			buf = bytes.SplitN(buf, []byte{0}, 2)[0]
			buf = bytes.TrimRight(buf, "\r\n\t ")
			
			if len(buf)+2 > blankSize {
				fmt.Print("\033[1A", strings.Repeat(" ", len(buf)+2), "\r")
			}else{
				fmt.Print("\033[1A", strings.Repeat(" ", blankSize), "\r")
			}

			input := bytes.SplitN(buf, []byte{' '}, 2)
			action := string(input[0])
	
			if action == "" || action == "stop" || action == "exit" {
				fmt.Println("\x1b[1;31mStopping Server!", "\x1b[0m")
	
				app.Shutdown()
	
				time.Sleep(300 * time.Millisecond)

				if len(input) > 1 && len(input[1]) != 0 {
					if i, err := strconv.Atoi(string(input[1])); err == nil {
						os.Exit(i)
					}
				}

				os.Exit(0)
			}else{
				var args map[string]string
				if len(input) > 1 && len(input[1]) != 0 {
					argListBuf := regex.Comp(`((?:[^\s]+=|)(?:".*?"|'.*?'|\'.*?\'|[^\s]+))`).Split(input[1])
					argList := []string{}
					for _, v := range argListBuf {
						if len(bytes.TrimSpace(v)) != 0 {
							argList = append(argList, string(bytes.TrimSpace(v)))
						}
					}
					args = goutil.MapArgs(argList)
				}

				if cb, ok := termCommands[action]; ok {
					cb(args)
				}
			}
		}
	}()

	err = webext.ListenAutoTLS(app, Config.Port[0], Config.Port[1], PWD+"/db/ssl/auto_ssl", Config.Proxy)
	if err != nil {
		log.Fatal(err)
	}
}

// HandleApp is an initialized queue to run when gofiber is ready
// and is used to handle app.Use listeners and middleware
//
// @priority: what order to run your listeners in
//  - positive numbers (127) will make your callback run first
//  - negative numbers (-128) will make your callback run last
//  - 0 is the default for the main app
func HandleApp(priority int8, cb func(app *fiber.App)){
	if handleApp[priority] == nil {
		handleApp[priority] = []func(app *fiber.App){}
	}
	handleApp[priority] = append(handleApp[priority], cb)
}
