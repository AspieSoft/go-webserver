package main

import (
	"encoding/base64"
	"errors"
	"os"
	"time"

	"github.com/AspieSoft/goutil/crypt/v2"
	"github.com/AspieSoft/webext"
	"github.com/gofiber/fiber/v2"
)

func init(){
	PrepareDB(func() {
		st, _ := DB.Prepare("CREATE TABLE IF NOT EXISTS login_sessions (token TEXT, uuid TEXT, exp INTEGER)")
		st.Exec()
	})

	AfterDB(func(){
		hashKey, err := os.ReadFile(PWD+"/db/hash.key")
		if err != nil {
			panic(err)
		}

		webext.Hooks.LoginForm.VerifyUserPass = func(username, password string) (uuid string, verified bool) {
			if username == "" || password == "" {
				return "", false
			}

			rows, _ := DB.Query("SELECT uuid, username, password, email, verified FROM users WHERE username = ? OR email = ?", username, username)

			var rowUUID string
			var rowUsername string
			var rowPassword string
			var rowEmail string
			var rowVerified bool
			for rows.Next() {
				rows.Scan(&rowUUID, &rowUsername, &rowPassword, &rowEmail, &rowVerified)

				passBytes, err := base64.StdEncoding.DecodeString(rowPassword)
				if err != nil {
					return "", false
				}

				if (username == rowUsername || username == rowEmail) && crypt.Hash.Compare([]byte(password), passBytes, hashKey) {
					rows.Close()

					return rowUUID, true
				}
			}

			return "", false
		}

		webext.Hooks.LoginForm.VerifySession = func(token string) (uuid string, verified bool) {
			if token == "" {
				return "", false
			}

			now := time.Now().UnixMilli()

			rows, _ := DB.Query("SELECT token, uuid, exp FROM login_sessions WHERE token = ?", token)

			var rowToken string
			var rowUUID string
			var rowExp int64

			for rows.Next() {
				rows.Scan(&rowToken, &rowUUID, &rowExp)
				rows.Close()
			}

			if now > rowExp {
				webext.Hooks.LoginForm.RemoveSession(token)
				return "", false
			}

			return rowUUID, true
		}

		webext.Hooks.LoginForm.CreateSession = func(uuid string) (string, time.Time, error) {
			exp := time.Now().Add(30 * 24 * time.Hour)
			token := string(crypt.RandBytes(256))

			rows, _ := DB.Query("SELECT token FROM login_sessions WHERE token = ?", token)
			loops := 100
			for rows.Next() {
				rows.Close()
				loops--
				if loops < 0 {
					break
				}
				token = string(crypt.RandBytes(256))
				rows, _ = DB.Query("SELECT token FROM login_sessions WHERE token = ?", token)
			}
			rows.Close()

			if loops < 1 {
				return "", time.Now().Add(-24 * time.Hour), errors.New("503:Server Is Full")
			}

			st, _ := DB.Prepare("INSERT INTO login_sessions (token, uuid, exp) VALUES (?, ?, ?)")
			st.Exec(token, uuid, exp.UnixMilli())

			return token, exp, nil
		}

		webext.Hooks.LoginForm.RemoveSession = func(token string) {
			if token != "" {
				DB.Query("DELETE FROM login_sessions WHERE token = ?", token)
			}
		}

		webext.Hooks.LoginForm.Render = func(c *fiber.Ctx, session string) error {
			return c.Render("form/login", fiber.Map{
				"Title": "Login",
				"session": session,
			})
		}

		webext.Hooks.LoginForm.OnAttempt = func(c *fiber.Ctx, method string) (allow bool) {
			//todo: check database for too many failed attempts
			return true
		}

		webext.Hooks.LoginForm.OnFailedAttempt = func(c *fiber.Ctx, method string) {
			//todo: add failed attempt counter to database with expiration
		}

		//todo: add cron job for clearing expired login sessions and login attempts from database
	})
}
