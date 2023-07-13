package models

import (
	"fmt"
	"log"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/util"
)

type Campaign struct {
	ID        uint     `json:"campaign_id,omitempty"`
	OwnerID   uint     `json:"owner_id"`
	Title     string   `json:"title"`
	CreatedAt string   `json:"created_at"`
	Websites  []string `json:"websites"` // array of websites db base_url reference
}

func CreateCampaign(ownerID uint, title string) (*Campaign, error) {
	if len(title) < 3 || len(title) > 128 {
		return nil, protocol.ErrCampaignTitleLen
	}

	return &Campaign{
		OwnerID:   ownerID,
		Title:     title,
		CreatedAt: util.GetCurrentTime(),
	}, nil
}

func (c *Campaign) Insert() error {
	var results []Campaign
	if err := db.DB.From("campaigns").Insert(&c).Execute(&results); err != nil {
		return err
	}

	return nil
}

func (c *Campaign) Delete() error {
	var results []any
	if err := db.DB.From("campaigns").Delete().Eq("campaign_id", fmt.Sprint(c.ID)).Execute(&results); err != nil {
		return err
	}
	return nil
}

func (c *Campaign) Update() error {
	var results []Campaign

	if err := db.DB.From("campaigns").Update(&c).Eq("campaign_id", fmt.Sprint(c.ID)).Execute(&results); err != nil {
		return err
	}

	return nil
}

func (c *Campaign) SetTitle(title string) {
	c.Title = title
}

func (c *Campaign) AddWebsites(websites ...*protocol.Website) error {
	for _, w := range websites {
		// cleaning duplicates
		c.Websites = protocol.AppendUnique(c.Websites, w.BaseUrl)
	}

	var result []Campaign
	if err := db.DB.From("campaigns").Update(&c).Eq("campaign_id", fmt.Sprint(c.ID)).Execute(&result); err != nil {
		log.Printf("err adding to campaign update: %v", err)
	}

	return nil
}

func SaveToCampaign(u *User, id uint, websites []*protocol.Website) error {
	if err := VerifyCampaignOwnership(u, id); err != nil {
		return err
	}

	campaign, err := GetCampaign(id)
	if err != nil {
		return protocol.ErrCampaignNotFound
	}

	if err := campaign.AddWebsites(websites...); err != nil {
		return err
	}

	return nil
}

func VerifyCampaignOwnership(u *User, campaignID uint) error {
	has, err := u.HasCampaign(campaignID)
	if err != nil {
		return err
	}
	if !has {
		return protocol.ErrCampaignUnowned
	}
	return nil
}
