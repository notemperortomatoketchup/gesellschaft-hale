package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/nedpals/supabase-go"
	"github.com/wotlk888/gesellschaft-hale/protocol"
)

var db *supabase.Client

func (app *Application) startDB() {
	db = supabase.CreateClient("https://nthqdxpnkubmeefmssat.supabase.co", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Im50aHFkeHBua3VibWVlZm1zc2F0Iiwicm9sZSI6ImFub24iLCJpYXQiOjE2ODg3NTcwOTgsImV4cCI6MjAwNDMzMzA5OH0.JdNuCMg1XkxlKpWT8KQOrm4nbAa_-4gnshZwfg4G3pw")
	log.Println("spinned up supabase client")
}

func getUser(username string) (*User, error) {
	var user []User
	if err := db.DB.From("users").Select("*").Eq("username", username).Execute(&user); err != nil {
		return nil, err
	}

	if len(user) != 0 {
		return &user[0], nil
	}

	return nil, protocol.ErrUserNotFound
}

func getCampaign(id int) (*Campaign, error) {
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
			if err := db.DB.From("websites").Insert(&wb).Execute(&results); err != nil {
				log.Printf("failed to save %s: %v", wb.BaseUrl, err)
			}
			defer wg.Done()
		}(w)
	}

	wg.Wait()
}
