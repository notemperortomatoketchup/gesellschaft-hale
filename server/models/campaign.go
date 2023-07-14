package models

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/lib/pq"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/util"
	"gorm.io/gorm"
)

type Campaign struct {
	ID               uint           `json:"id,omitempty" gorm:"primarykey"`
	OwnerID          uint           `json:"owner_id" gorm:"column:owner_id"`
	Title            string         `json:"title" `
	CreatedAt        string         `json:"created_at" gorm:"column:created_at"`
	Websites         pq.StringArray `json:"websites" gorm:"type:text[]"` // array of websites db base_url reference
	NotionIntegrated bool           `json:"notion_integrated" gorm:"notion_integrated"`
	NotionPageID     string         `json:"notion_page_id" gorm:"column:notion_page_id"`
	NotionDatabaseID string         `json:"notion_database_id" gorm:"column:notion_database_id"`
	*NotionClient    `json:"-" gorm:"-"`
}

func CreateCampaign(ownerID uint, title string, notion bool) (*Campaign, error) {
	if len(title) < 3 || len(title) > 128 {
		return nil, protocol.ErrCampaignTitleLen
	}

	campaign := &Campaign{
		OwnerID:   ownerID,
		Title:     title,
		CreatedAt: util.GetCurrentTime(),
		Websites:  []string{},
	}

	if notion {
		u, err := GetUserByID(ownerID)
		if err != nil {
			return nil, err
		}

		err = campaign.StartNotionClient(*u.ID)
		if err != nil {
			return nil, err
		}

		page, db, err := campaign.NotionCreateCampaign(title)
		if err != nil {
			return nil, protocol.ErrNotionCreatePage
		}

		campaign.NotionIntegrated = true
		campaign.NotionPageID = page.ID
		campaign.NotionDatabaseID = db.ID
	}

	return campaign, nil
}

func (c *Campaign) Insert() error {
	if err := db.Table("campaigns").Create(&c).Error; err != nil {
		return err
	}

	return nil
}

func (c *Campaign) Sync() error {
	if c.NotionIntegrated == false || c.NotionClient == nil {
		return protocol.ErrNotionMissingCreds
	}

	if err := c.NotionDeletePage(c.NotionPageID); err != nil {
		return err
	}

	page, db, err := c.NotionCreateCampaign(c.Title)
	if err != nil {
		return err
	}

	results, err := c.GetResults()
	if err != nil {
		return err
	}
	fmt.Println("Results:", results)

	c.NotionAddEntries(db.ID, results...)
	c.NotionDatabaseID = db.ID
	c.NotionPageID = page.ID

	if err := c.Update(); err != nil {
		return err
	}

	return nil
}

func (c *Campaign) GetResults() ([]*protocol.Website, error) {
	var websites []*protocol.Website
	websitesCh := make(chan *protocol.Website, 0)
	ctx, cancel := context.WithCancel(context.Background())
	// go func so we can select, and channel prevent raec condition for writing as blocking
	go func() {
		var wg sync.WaitGroup
		for _, url := range c.Websites {
			wg.Add(1)
			go func(w string) {
				defer wg.Done()
				website, _ := GetWebsite(w)
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
			websites = append(websites, w)
		}
	}
	return websites, nil
}
func (c *Campaign) Delete() error {
	if err := db.Table("campaigns").Delete(&Campaign{}, c.ID).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return protocol.ErrCampaignNotFound
		}
		return err
	}
	return nil
}

func (c *Campaign) Update() error {
	if err := db.Table("campaigns").Save(&c).Error; err != nil {
		return err
	}

	return nil
}

func (c *Campaign) SetTitle(title string) {
	c.Title = title
}

func (c *Campaign) AddWebsites(websites ...*protocol.Website) error {
	for _, w := range websites {
		c.Websites = protocol.AppendUnique(c.Websites, w.BaseUrl)
	}

	if err := db.Table("campaigns").Save(&c).Error; err != nil {
		log.Printf("err adding to campaign update: %v", err)
	}

	if c.NotionIntegrated {
		c.NotionClient.NotionAddEntries(c.NotionDatabaseID, websites...)
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
