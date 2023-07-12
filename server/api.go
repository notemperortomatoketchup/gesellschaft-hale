package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/sethvargo/go-password/password"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

const (
	METHOD_FAST int = iota
	METHOD_SLOW
)

type Message struct {
	Message string `json:"message"`
}
type CampaignOpts struct {
	ID *uint `json:"id" validate:"number"`
}

type MethodOpts struct {
	Method int `json:"method,omitempty" validate:"oneof=0 1"`
}

type TitleRequest struct {
	Title string `json:"title" validate:"required,min=3,max=32"`
}
type UrlsRequest struct {
	Urls []string `json:"urls" validate:"required,min=1,urls"`
}
type WebsitesResponse struct {
	Websites []*protocol.Website `json:"data"`
}

type GetMailsRequest struct {
	UrlsRequest
	MethodOpts
	Campaign CampaignOpts `json:"campaign,omitempty" validate:"-"`
}

type KeywordRequest struct {
	Keyword string `json:"keyword" validate:"required"`
	Pages   int    `json:"pages" validate:"required,number,min=1,max=20"`
	MethodOpts
	Campaign CampaignOpts `json:"campaign,omitempty" validate:"-"`
}

type AuthRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Password string `json:"password" validate:"required,min=3,max=32"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=3,max=32"`
	NewPassword string `json:"new_password" validate:"required,min=3,max=32"`
}

type EditUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
}

func (app *Application) initAPI() {
	f := fiber.New(fiber.Config{
		Immutable:    true,
		ServerHeader: "Fiber",
		AppName:      "gesellschaft-hale",
		ErrorHandler: ErrorHandler(),
	})
	f.Use(cors.New())

	auth := f.Group("/auth")
	auth.Post("/register", app.handleRegister)
	auth.Post("/login", app.handleLogin)

	api := f.Group("/api")
	api.Use(jwtware.New(jwtware.Config{
		SigningKey:   jwtware.SigningKey{Key: jwtsecret},
		ErrorHandler: ErrorHandler(),
	}))
	api.Use(localsIDMiddleware)
	api.Post("/getmails", app.handleMails)
	api.Post("/keyword", app.handleKeyword)
	api.Post("/keywordmail", app.handleKeywordMails)

	account := api.Group("/account")
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

	admin := api.Group("/admin")
	admin.Use(adminOnlyMiddleware)

	users := admin.Group("/users")
	users.Get("/", app.handleGetAllUsers)
	users.Post("/create", app.handleCreateUser)
	users.Get("/:id<int>", app.handleGetUser)
	users.Patch("/:id<int>", app.handleEditUser)
	users.Delete("/:id<int>", app.handleDeleteUser)

	if err := f.ListenTLS(":8443", "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatalf("error spinning fiber: %v", err)
	}

}

func (app *Application) handleKeyword(c *fiber.Ctx) error {
	request := new(KeywordRequest)
	response := new(WebsitesResponse)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	results, err := app.getKeywordResults(request.Keyword, request.Pages)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		saveToCampaign(u, *request.Campaign.ID, results)
	}

	response.Websites = results

	return c.Status(fiber.StatusOK).JSON(response)
}

func (app *Application) handleMails(c *fiber.Ctx) error {
	request := new(GetMailsRequest)
	response := new(WebsitesResponse)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		if err := verifyCampaignOwnership(u, *request.Campaign.ID); err != nil {
			return err
		}
	}

	results, err := app.getMailsFromUrls(request.Urls, request.Method)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		if err := saveToCampaign(u, *request.Campaign.ID, results); err != nil {
			return err
		}
	}

	response.Websites = results

	return c.Status(fiber.StatusOK).JSON(response)
}

func (app *Application) handleKeywordMails(c *fiber.Ctx) error {
	request := new(KeywordRequest)
	response := new(WebsitesResponse)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	scraped, err := app.getKeywordResults(request.Keyword, request.Pages)
	if err != nil {
		return err
	}

	results, err := app.getMailsFromWebsites(scraped, request.Method)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		saveToCampaign(u, *request.Campaign.ID, results)
	}

	response.Websites = results

	return c.Status(fiber.StatusOK).JSON(response)
}

func (app *Application) handleRegister(c *fiber.Ctx) error {
	request := new(AuthRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	user := new(User)
	user.SetUsername(request.Username)

	if err := user.SetPassword(request.Password); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := user.Insert(); err != nil {
		return internalError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(Message{
		Message: "Created account successfully.",
	})
}

func (app *Application) handleLogin(c *fiber.Ctx) error {
	request := new(AuthRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	// pull user from db
	user, err := getUserByUsername(request.Username)
	if err != nil {
		return internalError(err)
	}

	if err := user.IsPassword(request.Password); err != nil {
		return badRequest(fmt.Errorf("%s", protocol.ErrInvalidCredentials))
	}

	// got user right, need to generate token jwt
	token, err := user.generateJWT()
	if err != nil {
		return internalError(fmt.Errorf("err generating jwt"))
	}

	return c.Status(fiber.StatusOK).JSON(struct {
		Token string `json:"token"`
	}{
		Token: token,
	})
}

func (app *Application) handleResetPassword(c *fiber.Ctx) error {
	response := struct {
		Password string `json:"password"`
	}{}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	pass, err := password.Generate(24, 10, 10, false, false)
	if err != nil {
		return internalError(fmt.Errorf("error generating the random password"))
	}

	if err := u.SetPassword(pass); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := u.Update(); err != nil {
		return internalError(err)
	}

	response.Password = pass

	return c.Status(fiber.StatusOK).JSON(response)

}
func (app *Application) handleChangePassword(c *fiber.Ctx) error {
	request := new(ChangePasswordRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	if err := u.IsPassword(request.OldPassword); err != nil {
		return badRequest(err)
	}

	if err := u.SetPassword(request.NewPassword); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := u.Update(); err != nil {
		return internalError(err)
	}

	return c.Status(fiber.StatusOK).JSON(Message{
		Message: "Password changed successfully",
	})
}

func (app *Application) handleCreateCampaign(c *fiber.Ctx) error {
	request := new(TitleRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	campaign, err := createCampaign(*u.ID, request.Title)
	if err != nil {
		return err
	}

	if err := campaign.Insert(); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(Message{
		Message: "Campaign successfully created",
	})
}

func (app *Application) handleGetCampaign(c *fiber.Ctx) error {
	id, _ := getIDInLocals(c)

	campaign, err := getCampaign(id)
	if err != nil {
		return badRequest(err)
	}

	return c.Status(fiber.StatusFound).JSON(campaign)
}

func (app *Application) handleGetResultsCampaign(c *fiber.Ctx) error {
	response := new(WebsitesResponse)
	id, has := getIDInLocals(c)
	if !has {
		return badRequest(protocol.ErrInvalidID)
	}

	campaign, err := getCampaign(id)
	if err != nil {
		return badRequest(err)
	}

	if len(campaign.Websites) == 0 {
		return badRequest(protocol.ErrCampaignEmpty)
	}

	websitesCh := make(chan *protocol.Website, 0)
	ctx, cancel := context.WithCancel(context.Background())
	// go func so we can select, and channel prevent raec condition for writing as blocking
	go func() {
		var wg sync.WaitGroup
		for _, url := range campaign.Websites {
			wg.Add(1)
			go func(w string) {
				defer wg.Done()
				website, _ := getWebsite(w)
				websitesCh <- website
			}(url)
		}
		wg.Wait()
		cancel()
	}()
mainloop:
	for {
		select {
		case <-ctx.Done():
			break mainloop
		case w := <-websitesCh:
			response.Websites = append(response.Websites, w)
		}
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (app *Application) handleDeleteCampaign(c *fiber.Ctx) error {
	id, _ := getIDInLocals(c)

	campaign, err := getCampaign(id)
	if err != nil {
		return err
	}

	if err := campaign.Delete(); err != nil {
		return err
	}

	return c.Status(fiber.StatusNoContent).JSON("")
}

func (app *Application) handleEditCampaign(c *fiber.Ctx) error {
	request := new(TitleRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}
	id, _ := getIDInLocals(c)

	campaign, err := getCampaign(id)
	if err != nil {
		return err
	}

	campaign.SetTitle(request.Title)
	if err := campaign.Update(); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(Message{
		Message: "Edited successfully",
	})

}

func (app *Application) handleDeleteResultsCampaign(c *fiber.Ctx) error {
	request := new(UrlsRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	id, _ := getIDInLocals(c)
	campaign, err := getCampaign(id)
	if err != nil {
		return badRequest(err)
	}

	var has bool
	for _, w := range campaign.Websites {
		if protocol.IsExists(request.Urls, w) {
			if !has {
				has = true
			}
			campaign.Websites = protocol.RemoveStrFromSlice(campaign.Websites, w)
		}
	}

	if !has {
		return badRequest(fmt.Errorf("no matching websites found"))
	}

	if err := campaign.Update(); err != nil {
		return internalError(err)
	}

	return c.Status(fiber.StatusNoContent).JSON("")
}

func (app *Application) handleGetUser(c *fiber.Ctx) error {
	u := new(User)
	id, _ := getIDInLocals(c)

	u, err := getUserByID(id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(u)
}

func (app *Application) handleGetAllUsers(c *fiber.Ctx) error {
	users, err := getAllUsers()
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(users)
}

func (app *Application) handleDeleteUser(c *fiber.Ctx) error {
	id, _ := getIDInLocals(c)

	u, err := getUserByID(id)
	if err != nil {
		return err
	}

	if err := u.Delete(); err != nil {
		return internalError(err)
	}
	return c.Status(fiber.StatusNoContent).JSON("")
}

func (app *Application) handleEditUser(c *fiber.Ctx) error {
	request := new(EditUserRequest)
	id, _ := getIDInLocals(c)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := getUserByID(id)
	if err != nil {
		return err
	}

	if request.Username != "" {
		u.SetUsername(request.Username)
	}

	if err := u.Update(); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(Message{
		Message: "Edited user succesfully",
	})
}

func (app *Application) handleCreateUser(c *fiber.Ctx) error {
	request := new(AuthRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	user := new(User)
	if err := user.SetUsername(request.Username).SetPassword(request.Password); err != nil {
		return err
	}

	if err := user.Insert(); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(Message{
		Message: "Created user successfully",
	})
}
