package main

import (
	"fmt"
	"log"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

type Campaign struct {
	ID        int      `json:"campaign_id,omitempty"`
	OwnerID   int      `json:"owner_id"`
	Title     string   `json:"title"`
	CreatedAt string   `json:"created_at"`
	Websites  []string `json:"websites"` // array of websites db base_url reference
}

func createCampaign(ownerID int, title string) (*Campaign, error) {
	if len(title) < 3 || len(title) > 128 {
		return nil, protocol.ErrCampaignTitleLen
	}

	return &Campaign{
		OwnerID:   ownerID,
		Title:     title,
		CreatedAt: getCurrentTime(),
	}, nil
}

func (c *Campaign) Insert() error {
	var results []Campaign
	if err := db.DB.From("campaigns").Insert(&c).Execute(&results); err != nil {
		return err
	}

	return nil
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
