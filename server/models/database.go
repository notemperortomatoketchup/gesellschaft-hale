package models

import (
	"log"
	"strings"
	"sync"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

var db *gorm.DB

// Prevent postgres to return it as uint8 byte slice

func StartDB(dsn string) {
	sqldb, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("err spinning up postgres: %v", err)
	}

	db = sqldb

	log.Printf("spinned up postgres")
}

// Use it only if truly needed, prefer getUserByID at all costs.
func GetUserByUsername(username string) (*User, error) {
	u := new(User)
	if err := db.Model(User{}).Where("username = ?", username).First(u).Error; err != nil {
		return nil, err
	}

	return u, nil
}

func GetUserByID(id uint) (*User, error) {
	u := new(User)

	if err := db.First(u, id).Error; err != nil {
		return nil, err
	}

	return u, nil
}

func GetAllUsers() ([]User, error) {
	var users []User

	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func GetCampaign(id uint) (*Campaign, error) {
	campaign := new(Campaign)

	if err := db.Table("campaigns").First(campaign, id).Error; err != nil {
		return nil, err
	}

	return campaign, nil
}

type WebsiteSQL struct {
	BaseUrl     string         `gorm:"column:base_url"`
	Paths       pq.StringArray `gorm:"type:text[]"`
	Title       string
	Description string
	Language    pq.StringArray `gorm:"type:text[]"`
	Region      pq.StringArray `gorm:"type:text[]"`
	Mails       pq.StringArray `gorm:"type:text[]"`
	Timeout     bool
}

func (ws *WebsiteSQL) Transform() *protocol.Website {
	// first we init our string array or they'll be nil if empty..
	website := &protocol.Website{
		Paths:    make([]string, len(ws.Paths)),
		Language: make([]string, len(ws.Language)),
		Region:   make([]string, len(ws.Region)),
		Mails:    make([]string, len(ws.Mails)),
	}

	website.BaseUrl = ws.BaseUrl
	if len(ws.Paths) != 0 {
		website.Paths = ws.Paths
	}
	if len(ws.Language) != 0 {
		website.Language = ws.Language
	}

	if len(ws.Region) != 0 {
		website.Region = ws.Region
	}

	if len(ws.Mails) != 0 {
		website.Mails = ws.Mails
	}

	website.Title = ws.Title
	website.Description = ws.Description
	website.Timeout = ws.Timeout

	return website
}

func makeWebsiteSQL(w *protocol.Website) *WebsiteSQL {

	website := &WebsiteSQL{
		Paths:    make([]string, len(w.Paths)),
		Language: make([]string, len(w.Language)),
		Region:   make([]string, len(w.Region)),
		Mails:    make([]string, len(w.Mails)),
	}
	website.BaseUrl = w.BaseUrl
	if len(w.Paths) != 0 {
		website.Paths = w.Paths
	}
	if len(w.Language) != 0 {
		website.Language = w.Language
	}

	if len(w.Region) != 0 {
		website.Region = w.Region
	}

	if len(w.Mails) != 0 {
		website.Mails = w.Mails
	}

	website.Title = w.Title
	website.Description = w.Description
	website.Timeout = w.Timeout

	return website
}
func GetWebsite(url string) (*protocol.Website, error) {
	sqlWebsite := new(WebsiteSQL)
	url = strings.TrimSuffix(url, "/")

	if err := db.Table("websites").Model(WebsiteSQL{}).Where("base_url = ?", url).First(sqlWebsite).Error; err != nil {
		return nil, err
	}

	return sqlWebsite.Transform(), nil
}

func SaveWebsites(websites []*protocol.Website) {
	var wg sync.WaitGroup

	for _, w := range websites {
		wg.Add(1)
		go func(wb *protocol.Website) {
			found, _ := GetWebsite(wb.BaseUrl)
			// update or insert if not found
			if found == nil {
				if err := db.Table("websites").Create(makeWebsiteSQL(wb)).Error; err != nil {
					log.Printf("failed to save %s: %v", wb.BaseUrl, err)
				}
			} else {
				if err := db.Table("websites").Where("base_url = ?", wb.BaseUrl).Save(makeWebsiteSQL(wb)).Error; err != nil {
					log.Printf("failed to update %s: %v", wb.BaseUrl, err)
				}
			}
			defer wg.Done()
		}(w)
	}

	wg.Wait()
}
