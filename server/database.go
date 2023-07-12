package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/nedpals/supabase-go"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

var db *supabase.Client

func (app *Application) startDB() {
	db = supabase.CreateClient("https://nthqdxpnkubmeefmssat.supabase.co", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Im50aHFkeHBua3VibWVlZm1zc2F0Iiwicm9sZSI6ImFub24iLCJpYXQiOjE2ODg3NTcwOTgsImV4cCI6MjAwNDMzMzA5OH0.JdNuCMg1XkxlKpWT8KQOrm4nbAa_-4gnshZwfg4G3pw")
	log.Println("spinned up supabase client")
}

func getUserByUsername(username string) (*User, error) {
	var user []User
	if err := db.DB.From("users").Select("*").Eq("username", username).Execute(&user); err != nil {
		return nil, err
	}

	if len(user) != 0 {
		return &user[0], nil
	}

	return nil, protocol.ErrUserNotFound
}

func getUserByID(id uint) (*User, error) {
	var user []User
	if err := db.DB.From("users").Select("*").Eq("id", fmt.Sprint(id)).Execute(&user); err != nil {
		return nil, err
	}

	if len(user) != 0 {
		return &user[0], nil
	}

	return nil, protocol.ErrUserNotFound
}

func getAllUsers() ([]User, error) {
	var users []User
	if err := db.DB.From("users").Select("*").Execute(&users); err != nil {
		return nil, err
	}

	return users, nil
}

func getCampaign(id uint) (*Campaign, error) {
	var campaign []Campaign
	if err := db.DB.From("campaigns").Select("*").Eq("campaign_id", fmt.Sprint(id)).Execute(&campaign); err != nil {
		return nil, protocol.ErrCampaignNotFound
	}

	if len(campaign) != 0 {
		return &campaign[0], nil
	}

	return nil, protocol.ErrCampaignNotFound
}

func getWebsite(url string) (*protocol.Website, error) {
	var results []protocol.Website
	url = strings.TrimSuffix(url, "/")

	if err := db.DB.From("websites").Select("*").Filter("base_url", "eq", url).Execute(&results); err != nil {
		return nil, protocol.ErrWebsiteNotFound
	}

	if len(results) != 0 {
		return &results[0], nil
	}

	return nil, protocol.ErrWebsiteNotFound
}

func saveWebsites(websites []*protocol.Website) {
	var wg sync.WaitGroup

	for _, w := range websites {
		wg.Add(1)
		go func(wb *protocol.Website) {
			var results []protocol.Website
			found, _ := getWebsite(wb.BaseUrl)
			// update or insert if not found
			if found == nil {
				if err := db.DB.From("websites").Insert(&wb).Execute(&results); err != nil {
					log.Printf("failed to save %s: %v", wb.BaseUrl, err)
				}
			} else {
				if err := db.DB.From("websites").Update(&wb).Filter("base_url", "eq", wb.BaseUrl).Execute(&results); err != nil {
					log.Printf("failed to update %s: %v", wb.BaseUrl, err)
				}
			}
			defer wg.Done()
		}(w)
	}

	wg.Wait()
}
