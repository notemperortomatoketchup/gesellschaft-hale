package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func (app *Application) StartAPI() {
	app.fiber = fiber.New(fiber.Config{
		Immutable:    true,
		ServerHeader: "Fiber",
		AppName:      "gesellschaft-hale",
		ErrorHandler: ErrorHandler(),
	})

	app.StartRouter(app.fiber)
	app.StartValidator()

	if err := app.fiber.ListenTLS(app.config.port, "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatalf("error spinning fiber: %v", err)
	}
}
