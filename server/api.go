package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func (app *Application) initAPI() {
	app.Fiber = fiber.New(fiber.Config{
		Immutable:    true,
		ServerHeader: "Fiber",
		AppName:      "gesellschaft-hale",
		ErrorHandler: ErrorHandler(),
	})

	app.initRouter(app.Fiber)

	if err := app.Fiber.ListenTLS(":8443", "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatalf("error spinning fiber: %v", err)
	}
}
