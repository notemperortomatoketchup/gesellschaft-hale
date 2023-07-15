package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/sethvargo/go-password/password"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/models"
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

type DomainOpts struct {
	Domain string `json:"domain" validate:"omitempty,startswith=google,contains=."`
}

type TitleRequest struct {
	Title string `json:"title" validate:"required,min=3,max=32"`
}

type CreateCampaignRequest struct {
	TitleRequest
	LinkNotion bool `json:"notion_integration"`
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
	DomainOpts
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
	Username     string `json:"username" validate:"omitempty,min=3,max=32"`
	NotionSecret string `json:"notion_secret"  gorm:"notion_secret_id"`
	NotionParent string `json:"notion_parent" gorm:"notion_parent_id"`
}

func (app *Application) handleKeyword(c *fiber.Ctx) error {
	request := new(KeywordRequest)
	response := new(WebsitesResponse)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return err
	}

	results, err := app.getKeywordResults(request.Keyword, request.Pages, request.Domain, true)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		models.SaveToCampaign(u, *request.Campaign.ID, results)
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

	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		if err := models.VerifyCampaignOwnership(u, *request.Campaign.ID); err != nil {
			return err
		}
	}

	results, err := app.getMailsFromUrls(request.Urls, request.Method)
	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		if err := models.SaveToCampaign(u, *request.Campaign.ID, results); err != nil {
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

	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return err
	}

	scraped, err := app.getKeywordResults(request.Keyword, request.Pages, request.Domain, false) // we don't save as we'll proceed with the results.
	if err != nil {
		return err
	}

	results, err := app.getMailsFromWebsites(scraped, request.Method)

	if err != nil {
		return err
	}

	if request.Campaign.ID != nil {
		models.SaveToCampaign(u, *request.Campaign.ID, results)
	}

	response.Websites = results

	return c.Status(fiber.StatusOK).JSON(response)
}

func (app *Application) handleRegister(c *fiber.Ctx) error {
	request := new(AuthRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	user := new(models.User)

	if err := user.SetUsername(request.Username).SetPassword(request.Password); err != nil {
		return internalError(protocol.ErrPasswordEncryption)
	}

	if err := user.Insert(); err != nil {
		fmt.Printf("user -> %v", *user)
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
	user, err := models.GetUserByUsername(request.Username)
	if err != nil {
		return internalError(err)
	}

	if err := user.IsPassword(request.Password); err != nil {
		return badRequest(fmt.Errorf("%s", protocol.ErrInvalidCredentials))
	}

	// got user right, need to generate token jwt
	token, err := models.GenerateJWT(app.config.secret, user)
	if err != nil {
		return internalError(fmt.Errorf("err generating jwt"))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"Token": token,
	})
}

func (app *Application) handleResetPassword(c *fiber.Ctx) error {
	response := struct {
		Password string `json:"password"`
	}{}

	u, err := models.GetUserFromJWT(c)
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

func (app *Application) handleAccountEdit(c *fiber.Ctx) error {
	request := new(EditUserRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return badRequest(err)
	}

	request.Matches(u)

	if err := u.Update(); err != nil {
		return internalError(err)
	}

	return c.Status(fiber.StatusOK).JSON(Message{
		Message: "Edited account successfully",
	})
}
func (app *Application) handleAccountInfo(c *fiber.Ctx) error {
	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return badRequest(err)
	}

	return c.Status(fiber.StatusOK).JSON(u)
}
func (app *Application) handleChangePassword(c *fiber.Ctx) error {
	request := new(ChangePasswordRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := models.GetUserFromJWT(c)
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
	request := new(CreateCampaignRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	u, err := models.GetUserFromJWT(c)
	if err != nil {
		return err
	}

	campaign, err := models.CreateCampaign(*u.ID, request.Title, request.LinkNotion)
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

	campaign, err := models.GetCampaign(id)
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

	campaign, err := models.GetCampaign(id)
	if err != nil {
		return badRequest(err)
	}

	if len(campaign.Websites) == 0 {
		return badRequest(protocol.ErrCampaignEmpty)
	}

	results, err := campaign.GetResults()
	if err != nil {
		return err
	}

	response.Websites = results

	return c.Status(fiber.StatusOK).JSON(response)
}

func (app *Application) handleDeleteCampaign(c *fiber.Ctx) error {
	id, _ := getIDInLocals(c)

	campaign, err := models.GetCampaign(id)
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

	campaign, err := models.GetCampaign(id)
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

func (app *Application) handleCampaignSync(c *fiber.Ctx) error {
	id, _ := getIDInLocals(c)
	campaign, err := models.GetCampaign(id)
	if err != nil {
		return badRequest(err)
	}

	if err := campaign.Sync(); err != nil {
		return internalError(err)
	}

	return c.Status(fiber.StatusOK).JSON(Message{
		Message: "Synced",
	})
}

func (app *Application) handleDeleteResultsCampaign(c *fiber.Ctx) error {
	request := new(UrlsRequest)

	if err := bind(c, request); err != nil {
		return validationError(c, err)
	}

	id, _ := getIDInLocals(c)
	campaign, err := models.GetCampaign(id)
	if err != nil {
		return badRequest(err)
	}

	var has bool
	var removedUrls []string
	for _, w := range campaign.Websites {
		if protocol.IsExists(request.Urls, w) {
			if !has {
				has = true
			}
			campaign.Websites = protocol.RemoveStrFromSlice(campaign.Websites, w)
			removedUrls = append(removedUrls, w)
		}
	}

	if campaign.NotionIntegrated {
		campaign.NotionIntegration.DeleteEntry(removedUrls...)
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
	u := new(models.User)
	id, _ := getIDInLocals(c)

	u, err := models.GetUserByID(id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(u)
}

func (app *Application) handleGetAllUsers(c *fiber.Ctx) error {
	users, err := models.GetAllUsers()
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(users)
}

func (app *Application) handleDeleteUser(c *fiber.Ctx) error {
	id, _ := getIDInLocals(c)

	u, err := models.GetUserByID(id)
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

	u, err := models.GetUserByID(id)
	if err != nil {
		return err
	}

	request.Matches(u)

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

	user := new(models.User)
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

type FinderFilter struct {
	Region      string `json:"region"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type FinderGetRequest struct {
	Filters FinderFilter `json:"filter" validate:"omitempty"`
}

func (app *Application) handleFinderGet(c *fiber.Ctx) error {

	return c.Status(fiber.StatusOK).JSON("")

}
