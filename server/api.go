package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

const (
	METHOD_FAST int = iota
	METHOD_SLOW
)

type CampaignOpts struct {
	ID *uint `json:"id"`
}

type GetMailsRequest struct {
	Urls     []string     `json:"urls"`
	Campaign CampaignOpts `json:"campaign,omitempty"`
	Method   int          `json:"method,omitempty"`
}

type GetMailsResponse struct {
	Websites []*protocol.Website `json:"data"`
}

type KeywordRequest struct {
	Keyword  string       `json:"keyword"`
	Pages    int          `json:"pages"`
	Campaign CampaignOpts `json:"campaign,omitempty"`
	Method   int          `json:"method,omitempty"`
}

type KeywordResponse struct {
	Websites []*protocol.Website `json:"data"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type CreateCampaignRequest struct {
	Title string `json:"title"`
}

type EditCampaignRequest struct {
	Title string `json:"title"`
}

type DeleteResultsCampaignRequest struct {
	Urls []string `json:"urls"`
}

func (app *Application) initAPI() {
	e := echo.New()
	e.Use(middleware.CORS())

	auth := e.Group("auth")
	auth.POST("/register", app.handleRegister)
	auth.POST("/login", app.handleLogin)

	api := e.Group("api")
	if app.UseJWT {
		api.Use(echojwt.WithConfig(echojwt.Config{
			SigningKey:    jwtsecret,
			SigningMethod: "HS256",
			TokenLookup:   "header:Token",
		}))
	}
	api.POST("/mail", app.handleMails)
	api.POST("/keyword", app.handleKeyword)
	api.POST("/keywordmail", app.handleMailsFromKeyword)

	campaigns := api.Group("/campaigns")
	campaigns.Use(verifyOwnershipMiddleware)
	campaigns.POST("/create", app.handleCreateCampaign)
	campaigns.DELETE("/delete/:id", app.handleDeleteCampaign)
	campaigns.PATCH("/edit/:id", app.handleEditCampaign)
	campaigns.GET("/results/:id", app.handleGetResultsCampaign)
	campaigns.DELETE("/results/:id", app.handleDeleteResultsCampaign)

	user := api.Group("/user")
	user.POST("/changepassword", app.handleChangePassword)

	if err := e.StartTLS(":8443", "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatal(err)
	}
}

func (app *Application) handleKeyword(c echo.Context) error {
	request := new(KeywordRequest)
	response := new(KeywordResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleKeyword(request); err != nil {
		return err
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

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleMails(c echo.Context) error {
	request := new(GetMailsRequest)
	response := new(GetMailsResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleMails(request); err != nil {
		return err
	}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	results, err := app.getMailsFromUrls(request.Urls, request.Method)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		saveToCampaign(u, *request.Campaign.ID, results)
	}

	response.Websites = results

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleMailsFromKeyword(c echo.Context) error {
	request := new(KeywordRequest)
	response := new(KeywordResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleKeyword(request); err != nil {
		return err
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

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleRegister(c echo.Context) error {
	request := new(RegisterRequest)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleRegister(request); err != nil {
		return err
	}

	user := new(User)
	user.SetUsername(request.Username)

	if err := user.SetPassword(request.Password); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := user.Insert(); err != nil {
		return internalError(err)
	}

	return c.JSON(http.StatusOK, "good")
}

func (app *Application) handleLogin(c echo.Context) error {
	request := new(LoginRequest)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleLogin(request); err != nil {
		return err
	}

	// pull user from db
	user, err := getUser(request.Username)
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

	return c.JSON(http.StatusOK, token)
}

func (app *Application) handleChangePassword(c echo.Context) error {
	request := new(ChangePasswordRequest)

	if err := bind(c, request); err != nil {
		return err
	}

	user, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	if err := validateHandleChangePassword(request); err != nil {
		return err
	}

	if err := user.IsPassword(request.OldPassword); err != nil {
		return badRequest(err)
	}

	if err := user.SetPassword(request.NewPassword); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := user.Update(); err != nil {
		return internalError(err)
	}

	return c.JSON(http.StatusOK, nil)
}

func (app *Application) handleCreateCampaign(c echo.Context) error {
	request := new(CreateCampaignRequest)

	if err := bind(c, request); err != nil {
		return err
	}

	if err := validateHandleCreateCampaign(request); err != nil {
		return err
	}

	u, err := getUserFromJWT(c)
	if err != nil {
		return err
	}

	campaign, err := createCampaign(u.ID, request.Title)
	if err != nil {
		return err
	}

	if err := campaign.Insert(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, "created")
}

type GetListsCampaignResponse struct {
	Websites []*protocol.Website `json:"data"`
}

func (app *Application) handleGetResultsCampaign(c echo.Context) error {
	cc := c.(*CustomContext)
	response := new(GetListsCampaignResponse)

	campaign, err := getCampaign(cc.GetID())
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

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleDeleteCampaign(c echo.Context) error {
	cc := c.(*CustomContext)

	campaign, err := getCampaign(cc.GetID())
	if err != nil {
		return err
	}

	if err := campaign.Delete(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, "deleted")
}

func (app *Application) handleEditCampaign(c echo.Context) error {
	cc := c.(*CustomContext)
	request := new(EditCampaignRequest)

	if err := bind(c, request); err != nil {
		return err
	}

	if err := verifyHandleEditCampaign(request); err != nil {
		return err
	}

	campaign, err := getCampaign(cc.GetID())
	if err != nil {
		return err
	}

	campaign.SetTitle(request.Title)
	if err := campaign.Update(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, "edited")

}

func (app *Application) handleDeleteResultsCampaign(c echo.Context) error {
	cc := c.(*CustomContext)
	request := new(DeleteResultsCampaignRequest)

	if err := bind(c, request); err != nil {
		return err
	}

	if err := validateHandleDeleteResultsCampaign(request); err != nil {
		return err
	}

	campaign, err := getCampaign(cc.GetID())
	if err != nil {
		return badRequest(err)
	}

	var has bool
	for _, w := range campaign.Websites {
		if protocol.IsExists(request.Urls, w) {
			has = true
			campaign.Websites = protocol.RemoveStrFromSlice(campaign.Websites, w)
		}
	}

	// avoid update if no change.
	if !has {
		return c.JSON(http.StatusBadRequest, fmt.Errorf("no matching websites found in the campaign, deleted 0 entry"))
	}

	if err := campaign.Update(); err != nil {
		return internalError(err)
	}

	return c.JSON(http.StatusOK, "deleted entries from result")
}
