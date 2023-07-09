package main

import (
	"log"

	"github.com/nedpals/supabase-go"
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

	return &user[0], nil
}
