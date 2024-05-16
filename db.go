package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/AspieSoft/go-regex-re2/v2"
	"github.com/AspieSoft/goutil/crypt/v2"
	"github.com/AspieSoft/goutil/v7"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/term"
)

var DB *sql.DB
var prepareDB []func() = []func(){}
var afterDB []func() = []func(){}

func init(){
	PrepareDB(func() {
		// create users database
		st, _ := DB.Prepare("CREATE TABLE IF NOT EXISTS users (uuid TEXT, username TEXT, password TEXT, email TEXT, verified BOOL)")
		st.Exec()
	})
}

func initDB(){
	fmt.Println("\x1b[1;36mRunning Initial Setup...\n\x1b[0m")

	args := goutil.MapArgs()

	hashKey := crypt.RandBytes(256)
	err := os.WriteFile(PWD+"/db/hash.key", hashKey, 0600)
	if err != nil {
		panic(err)
	}

	adminUser := args["username"]
	adminPass := args["password"]

	if adminUser == "" || adminPass == "" {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("New Admin:")

		for adminUser == "" {
			fmt.Print("  Username: ")

			user, _ := reader.ReadString('\n')
			user = strings.TrimSuffix(user, "\n")
			if !regex.Comp(`^[A-Za-z0-9_\-@\.]+$`).Match([]byte(user)) {
				fmt.Println("\x1b[1;31m    username must match: [A-Za-z0-9_]\x1b[0m")
				continue
			}

			adminUser = user
			break
		}

		for adminPass == "" {
			fmt.Print("  Password: ")

			pass, _ := term.ReadPassword(int(syscall.Stdin))
			fmt.Println("")

			if !Config.DebugMode && (len(pass) < 8 || !regex.Comp(`[A-Z]`).Match([]byte(pass)) || !regex.Comp(`[a-z]`).Match([]byte(pass)) || !regex.Comp(`[0-9]`).Match([]byte(pass)) || !regex.Comp(`[^A-Za-z0-9]`).Match([]byte(pass))) {
				fmt.Println("\x1b[1;31m    password must contain at least:\n      8 characters total\n      1 uppercase letter\n      1 lowercase letter\n      1 number\n      1 special character\x1b[0m")
				continue
			}

			fmt.Print("  Confirm Password: ")

			confPass, _ := term.ReadPassword(int(syscall.Stdin))
			fmt.Println("")

			if !bytes.Equal(pass, confPass) {
				fmt.Println("\x1b[1;31m    passwords do not match!\x1b[0m")
				continue
			}

			adminPass = string(pass)
			break
		}
	}

	// hash admin password
	pass, err := crypt.Hash.New([]byte(adminPass), hashKey)
	if err != nil {
		panic(err)
	}
	adminPass = base64.StdEncoding.EncodeToString(pass)


	// run prepareDB queue
	for _, cb := range prepareDB {
		cb()
	}


	// create admin user
	st, _ := DB.Prepare("INSERT INTO users (uuid, username, password, email, verified) VALUES (?, ?, ?, '', true)")
	st.Exec(crypt.GenUUID(8), adminUser, adminPass)

	fmt.Println("\n\x1b[1;36mInitial Setup Complete!\n\x1b[0m")
}

// PrepareDB is an initialized queue to run after the database is open,
// to create database tables if they dont exist
func PrepareDB(cb func()){
	prepareDB = append(prepareDB, cb)
}

// AfterDB is an initialized queue to run after the database is open,
// and after tables are prepared
func AfterDB(cb func()){
	afterDB = append(afterDB, cb)
}
