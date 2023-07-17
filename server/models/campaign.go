package models

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/dstotijn/go-notion"
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
	*NotionIntegration
}

func CreateCampaign(ownerID uint, title string, notion bool) (*Campaign, error) {
	if len(title) < 3 || len(title) > 128 {
		return nil, protocol.ErrCampaignTitleLen
	}

	c := &Campaign{
		OwnerID:           ownerID,
		Title:             title,
		CreatedAt:         util.GetCurrentTime(),
		Websites:          []string{},
		NotionIntegration: new(NotionIntegration),
	}

	if notion {
		if err := c.Link(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Link instances the Notion Integration in order to be able to uses
// Notion related utilities.
func (c *Campaign) Link() error {
	u, err := GetUserByID(c.OwnerID)
	if err != nil {
		return err
	}

	if u.NotionParent == "" || u.NotionSecret == "" {
		return protocol.ErrNotionMissingCreds
	}

	c.NotionIntegration = &NotionIntegration{
		Client:     notion.NewClient(u.NotionSecret),
		ParentID:   u.NotionParent,
		PageID:     c.PageID,
		DatabaseID: c.DatabaseID,
	}

	if c.PageID == "" || c.DatabaseID == "" {
		page, err := c.NotionIntegration.CreatePage(c.Title)
		if err != nil {
			return protocol.ErrNotionCreatePage
		}

		db, err := c.NotionIntegration.CreateDatabase(page.ID)
		if err != nil {
			return protocol.ErrNotionCreatePage
		}

		c.NotionIntegration.PageID = page.ID
		c.NotionIntegration.DatabaseID = db.ID
		c.NotionIntegrated = true
	}

	return nil
}

func (c *Campaign) Insert() error {
	if err := db.Table("campaigns").Create(&c).Error; err != nil {
		return err
	}

	return nil
}

func (c *Campaign) Sync() error {
	// user asks to resync
	results, err := c.GetResults()
	if err != nil {
		return err
	}

	// delete old database
	if err := c.NotionIntegration.DeleteDatabase(); err != nil {
		// if we have not find, just create it again, surely a bug occured
		if !strings.Contains(err.Error(), "not find database with ID") {
			return err
		}
	}

	// recreate
	db, err := c.NotionIntegration.CreateDatabase(c.PageID)
	if err != nil {
		return err
	}

	c.NotionIntegration.DatabaseID = db.ID
	c.NotionIntegration.AddEntry(results...)

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
	if c.NotionIntegrated {
		if err := c.NotionIntegration.DeletePage(c.PageID); err != nil {
			// if it's already archived, it's ok to continue processing the deletion. Else, no.
			if !strings.Contains(err.Error(), "that is archived") {
				return err
			}
		}
	}

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
		c.NotionIntegration.AddEntry(websites...)
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
