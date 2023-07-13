package models

import (
	"errors"
	"log"

	"github.com/lib/pq"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/util"
	"gorm.io/gorm"
)

type Campaign struct {
	ID        uint           `json:"id,omitempty" gorm:"primarykey"`
	OwnerID   uint           `json:"owner_id" gorm:"column:owner_id"`
	Title     string         `json:"title" `
	CreatedAt string         `json:"created_at" gorm:"column:created_at"`
	Websites  pq.StringArray `json:"websites" gorm:"type:text[]"` // array of websites db base_url reference
}

func CreateCampaign(ownerID uint, title string) (*Campaign, error) {
	if len(title) < 3 || len(title) > 128 {
		return nil, protocol.ErrCampaignTitleLen
	}

	return &Campaign{
		OwnerID:   ownerID,
		Title:     title,
		CreatedAt: util.GetCurrentTime(),
		Websites:  []string{},
	}, nil
}

func (c *Campaign) Insert() error {
	if err := db.Table("campaigns").Create(&c).Error; err != nil {
		return err
	}

	return nil
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
