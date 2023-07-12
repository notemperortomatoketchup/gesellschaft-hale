package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func (app *Application) initAPI() {
	f := fiber.New(fiber.Config{
		Immutable:    true,
		ServerHeader: "Fiber",
		AppName:      "gesellschaft-hale",
		ErrorHandler: ErrorHandler(),
	})

	app.initRouter(f)

	if err := f.ListenTLS(":8443", "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatalf("error spinning fiber: %v", err)
	}

}
