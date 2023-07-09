package main

import (
	"fmt"
	"log"
	"net/http"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

// add to campaign? -> which id?
type CampaignOpts struct {
	ID int `json:"id"`
}

type HandleGetMailsRequest struct {
	Urls     []string     `json:"urls"`
	Campaign CampaignOpts `json:"campaign,omitempty"`
}

type HandleGetMailsResponse struct {
	Websites []*protocol.Website `json:"data"`
}

type HandleKeywordRequest struct {
	Keyword string `json:"keyword"`
	Pages   int    `json:"pages"`
}

type HandleKeywordResponse struct {
	Websites []*protocol.Website `json:"data"`
}

type handleLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type HandleRegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type handleChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type handleCreateCampaignRequest struct {
	Title string `json:"title"`
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
	campaigns.POST("/create", handleCreateCampaign)

	user := api.Group("/user")
	user.POST("/changepassword", app.handleChangePassword)

	if err := e.StartTLS(":8443", "./certs/cert.pem", "./certs/key.pem"); err != nil {
		log.Fatal(err)
	}
}

func (app *Application) handleKeyword(c echo.Context) error {
	request := new(HandleKeywordRequest)
	response := new(HandleKeywordResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleKeyword(request); err != nil {
		return err
	}

	results, err := app.getKeywordResults(request.Keyword, request.Pages)
	if err != nil {
		return err
	}

	response.Websites = results

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleMails(c echo.Context) error {
	request := new(HandleGetMailsRequest)
	response := new(HandleGetMailsResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleMails(request); err != nil {
		return err
	}

	// u, err := getUserFromJWT(c)
	// if err != nil {
	// 	return err
	// }

	// // verification of ownership.
	// if request.Campaign.ID != 0 {
	// 	if err := verifyCampaignOwnership(u, request.Campaign.ID); err != nil {
	// 		return err
	// 	}
	// }

	results, err := app.getMailsFromUrls(request.Urls)
	if err != nil {
		return err
	}

	// for _, r := range results {
	// 	fmt.Println("we here 2")
	// 	var results []protocol.Website
	// 	if err := db.DB.From("websites").Insert(r).Execute(&results); err != nil {
	// 		log.Printf("failed to save website %s: %v", r.BaseUrl, err)
	// 	}
	// }

	response.Websites = results

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleMailsFromKeyword(c echo.Context) error {
	request := new(HandleKeywordRequest)
	response := new(HandleKeywordResponse)

	if err := bind(c, &request); err != nil {
		return err
	}

	if err := validateHandleKeyword(request); err != nil {
		return err
	}

	scraped, err := app.getKeywordResults(request.Keyword, request.Pages)
	if err != nil {
		return err
	}

	results, err := app.getMailsFromWebsites(scraped)
	if err != nil {
		return err
	}

	response.Websites = results

	return c.JSON(http.StatusOK, response)
}

func (app *Application) handleRegister(c echo.Context) error {
	request := new(HandleRegisterRequest)

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
	request := new(handleLoginRequest)

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
	request := new(handleChangePasswordRequest)

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

	return c.JSON(http.StatusOK, "good")
}

func handleCreateCampaign(c echo.Context) error {
	request := new(handleCreateCampaignRequest)

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

	return c.JSON(http.StatusOK, "good")
}
