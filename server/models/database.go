package models

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/util"
)

var db *Database

type Entity interface {
	Get(id int)
	Delete() error
	Edit() error
}

type Database struct {
	*gorm.DB
}

// Prevent postgres to return it as uint8 byte slice
func StartDB(dsn string) {
	sqldb, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("err spinning up postgres: %v", err)
	}

	db = &Database{
		sqldb,
	}
}

func (u User) Get(id int) *User {
	return &u
}

func (u User) Delete() error {
	return nil
}

func (u User) Edit() error {
	hello(u)
	return nil
}

func hello(e Entity) {

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
	c := new(Campaign)

	if err := db.Table("campaigns").First(c, id).Error; err != nil {
		return nil, err
	}

	//link the campaign to the Notion Client if it's integrated.
	if c.NotionIntegrated {
		if err := c.Link(); err != nil {
			return nil, err
		}
	}

	fmt.Printf("initialized Notion -> %+v", *c.NotionIntegration)
	return c, nil
}

type WebsiteSQL struct {
	BaseUrl     string         `gorm:"column:base_url"`
	Paths       pq.StringArray `gorm:"type:text[]"`
	Title       string
	Description string
	Language    pq.StringArray `gorm:"type:text[]"`
	Region      pq.StringArray `gorm:"type:text[]"`
	Mails       pq.StringArray `gorm:"type:text[]"`
	Socials     pq.StringArray `gorm:"type:text[]"`
	Timeout     bool
}

func (ws *WebsiteSQL) Transform() *protocol.Website {
	// first we init our string array or they'll be nil if empty..
	website := &protocol.Website{
		Paths:    make([]string, len(ws.Paths)),
		Language: make([]string, len(ws.Language)),
		Region:   make([]string, len(ws.Region)),
		Mails:    make([]string, len(ws.Mails)),
		Socials:  make([]string, len(ws.Socials)),
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

	if len(ws.Socials) != 0 {
		website.Socials = ws.Socials
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
		Socials:  make([]string, len(w.Socials)),
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

	if len(w.Socials) != 0 {
		website.Socials = w.Socials
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, protocol.ErrWebsiteNotFound
		}
		return nil, err
	}

	return sqlWebsite.Transform(), nil
}

func SaveWebsites(websites []*protocol.Website) {
	var wg sync.WaitGroup

	for _, w := range websites {
		wg.Add(1)
		go func(wb *protocol.Website) {
			defer wg.Done()
			if wb.Timeout {
				return
			}
			found, _ := GetWebsite(wb.BaseUrl)
			// update or insert if not found
			if found == nil {
				if err := db.Table("websites").Create(makeWebsiteSQL(wb)).Error; err != nil {
					log.Printf("failed to save %s: %v", wb.BaseUrl, err)
				}
				return
			} else {
				merged := CompareUpdateWebsite(wb, found)
				if err := db.Table("websites").Where("base_url = ?", wb.BaseUrl).Save(makeWebsiteSQL(merged)).Error; err != nil {
					log.Printf("failed to update %s: %v", wb.BaseUrl, err)
				}
			}

		}(w)
	}
	wg.Wait()
}

// take the two websites, first the new value, and second the old value.
// Return a website that contains both merged, with conservation of equals
// appending unique of the slices, and erasement of difference (new wins on old)
// bool is for is the same or not
func CompareUpdateWebsite(new, old *protocol.Website) *protocol.Website {
	merged := old

	if equal := util.AssertEqual(new.Title, old.Title); !equal {
		merged.Title = new.Title
	}

	if equal := util.AssertEqual(new.Description, old.Description); !equal {
		merged.Description = new.Description
	}

	if equal := reflect.DeepEqual(new.Mails, old.Mails); !equal {
		merged.Mails = protocol.AppendUnique(merged.Mails, new.Mails...)
	}

	if equal := reflect.DeepEqual(new.Paths, old.Paths); !equal {
		merged.Paths = protocol.AppendUnique(merged.Paths, new.Paths...)
	}

	if equal := reflect.DeepEqual(new.Language, old.Language); !equal {
		merged.Language = protocol.AppendUnique(merged.Language, new.Language...)
	}

	if equal := reflect.DeepEqual(new.Region, old.Region); !equal {
		merged.Region = protocol.AppendUnique(merged.Region, new.Region...)
	}

	return merged
}
