package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/wotlk888/gesellschaft-hale/server/cmd/api/middlewares"
	"gorm.io/gorm"
)

var db *gorm.DB

func (app *Application) StartRouter(f *fiber.App) {
	auth := f.Group("/auth")
	auth.Post("/register", app.handleRegister)
	auth.Post("/login", app.handleLogin)
	auth.Get("/logout", app.handleLogout)

	api := f.Group("/api")
	api.Use(middlewares.SessionChecker)
	api.Use(middlewares.LocalsStorer)
	api.Post("/getmails", app.handleMails)
	api.Post("/keyword", app.handleKeyword)
	api.Post("/keywordmail", app.handleKeywordMails)

	account := api.Group("/account")
	account.Patch("/password/change", app.handleChangePassword)
	account.Post("/password/reset", app.handleResetPassword)
	account.Post("/mailer", app.handleAccountAddMailer)
	account.Delete("/mailer/:id<int>", app.handleAccountDeleteMailer)
	account.Get("/info", app.handleAccountInfo)
	account.Patch("/", app.handleAccountEdit)

	campaign := api.Group("/campaign")
	campaign.Use(middlewares.CampaignChecker)
	campaign.Post("/create", app.handleCreateCampaign)
	campaign.Get("/:id<int>", app.handleGetCampaign)
	campaign.Patch("/:id<int>", app.handleEditCampaign)
	campaign.Delete("/:id<int>", app.handleDeleteCampaign)
	campaign.Get("/results/:id<int>", app.handleGetResultsCampaign)
	campaign.Delete("/results/:id<int>", app.handleDeleteResultsCampaign)
	campaign.Get("/sync/:id<int>", app.handleCampaignSync)

	finder := api.Group("/finder")
	finder.Post("/", app.handleFinderGet)

	mailer := api.Group("/mailer")
	mailer.Post("/", app.handleMailerSend)

	admin := api.Group("/admin")
	admin.Use(middlewares.AdminOnly)

	users := admin.Group("/users")
	users.Get("/", app.handleGetAllUsers)
	users.Post("/create", app.handleCreateUser)
	users.Get("/:id<int>", app.handleGetUser)
	users.Patch("/:id<int>", app.handleEditUser)
	users.Delete("/:id<int>", app.handleDeleteUser)
}
