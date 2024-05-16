package main

import (
	"github.com/AspieSoft/webext"
	"github.com/gofiber/fiber/v2"
)

func init(){
	HandleApp(0, func(app *fiber.App) {
		app.Use(webext.VerifyLogin())

		app.Use("/", func(c *fiber.Ctx) error {
			return c.Render("index", fiber.Map{
				// "Title": "",
			})
		})
	})
}
