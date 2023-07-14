package main

import (
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func (app *Application) StartRouter(f *fiber.App) {
	f.Use(cors.New())

	auth := f.Group("/auth")
	auth.Post("/register", app.handleRegister)
	auth.Post("/login", app.handleLogin)

	api := f.Group("/api")
	api.Use(jwtware.New(jwtware.Config{
		SigningKey:   jwtware.SigningKey{Key: app.config.secret},
		ErrorHandler: ErrorHandler(),
	}))
	api.Use(localsIDMiddleware)
	api.Post("/getmails", app.handleMails)
	api.Post("/keyword", app.handleKeyword)
	api.Post("/keywordmail", app.handleKeywordMails)

	account := api.Group("/account")
	account.Get("/info", app.handleAccountInfo)
	account.Patch("/password/change", app.handleChangePassword)
	account.Post("/password/reset", app.handleResetPassword)

	campaign := api.Group("/campaign")
	campaign.Use(campaignMiddleware)
	campaign.Post("/create", app.handleCreateCampaign)
	campaign.Get("/:id<int>", app.handleGetCampaign)
	campaign.Patch("/:id<int>", app.handleEditCampaign)
	campaign.Delete("/:id<int>", app.handleDeleteCampaign)
	campaign.Get("/results/:id<int>", app.handleGetResultsCampaign)
	campaign.Delete("/results/:id<int>", app.handleDeleteResultsCampaign)
	campaign.Get("/sync/:id<int>", app.handleCampaignSync)

	finder := api.Group("/finder")
	finder.Post("/", app.handleFinderGet)

	admin := api.Group("/admin")
	admin.Use(adminOnlyMiddleware)

	users := admin.Group("/users")
	users.Get("/", app.handleGetAllUsers)
	users.Post("/create", app.handleCreateUser)
	users.Get("/:id<int>", app.handleGetUser)
	users.Patch("/:id<int>", app.handleEditUser)
	users.Delete("/:id<int>", app.handleDeleteUser)

}
