package models

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/dstotijn/go-notion"
	"github.com/wotlk888/gesellschaft-hale/protocol"
	"github.com/wotlk888/gesellschaft-hale/server/util"
)

type NotionIntegration struct {
	Client     *notion.Client `json:"-" gorm:"-"` // initialized whenever the campaign is there
	ParentID   string         `json:"-" gorm:"-"` // main page ID
	PageID     string         `json:"notion_page_id" gorm:"column:notion_page_id"`
	DatabaseID string         `json:"notion_database_id" gorm:"column:notion_database_id"`
}

func (n *NotionIntegration) CreatePage(title string) (*notion.Page, error) {
	page, err := n.Client.CreatePage(context.Background(), notion.CreatePageParams{
		ParentType: notion.ParentTypePage,
		ParentID:   n.ParentID,
		Title: []notion.RichText{
			{
				Text: &notion.Text{
					Content: title,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &page, err
}

func (n *NotionIntegration) DeletePage(pageID string) error {
	if _, err := n.Client.UpdatePage(context.Background(), pageID, notion.UpdatePageParams{
		Archived: util.PtrBool(true),
	}); err != nil {
		log.Printf("err archiving notion page -> %v", err)
		return err
	}

	return nil
}

func (n *NotionIntegration) CreateDatabase(pageID string) (*notion.Database, error) {
	db, err := n.Client.CreateDatabase(context.Background(), notion.CreateDatabaseParams{
		ParentPageID: pageID,
		Title: []notion.RichText{
			{
				Text: &notion.Text{
					Content: "Database of websites",
				},
			},
		},
		Properties: notion.DatabaseProperties{
			// add a title, cause necessary for api validation
			// along title above
			"": notion.DatabaseProperty{
				Type:  notion.DBPropTypeTitle,
				Title: &notion.EmptyMetadata{},
			},
			// map[string --> column name]content
			"Emails": notion.DatabaseProperty{
				Type:  notion.DBPropTypeEmail,
				Name:  "Emails",
				Email: &notion.EmptyMetadata{},
			},
			"Website Address": notion.DatabaseProperty{
				Type: notion.DBPropTypeURL,
				Name: "Website Address",
				URL:  &notion.EmptyMetadata{},
			},
		},
		IsInline: true,
	})

	if err != nil {
		return nil, err
	}

	return &db, nil
}

func (n *NotionIntegration) DeleteDatabase() error {
	if _, err := n.Client.UpdateDatabase(context.Background(), n.DatabaseID, notion.UpdateDatabaseParams{
		Archived: util.PtrBool(true),
		Title: []notion.RichText{
			{
				Text: &notion.Text{
					Content: "Database of websites",
				},
			},
		},
	}); err != nil {
		return err
	}
	return nil

}

func (n *NotionIntegration) SearchDatabase(url string) (*notion.Page, map[string]string, error) {
	result, err := n.Client.QueryDatabase(context.Background(), n.DatabaseID, &notion.DatabaseQuery{
		Filter: &notion.DatabaseQueryFilter{
			Property: "Website Address",
			DatabaseQueryPropertyFilter: notion.DatabaseQueryPropertyFilter{
				URL: &notion.TextPropertyFilter{
					Equals: url,
				},
			},
		},
	})
	if err != nil {
		return nil, nil, err
	}

	for _, p := range result.Results {
		properties := p.Properties.(notion.DatabasePageProperties)
		u := *properties["Website Address"].URL
		m := properties["Emails"].Email
		if u == url {
			content := make(map[string]string) // [url]mail
			if m != nil {
				content[u] = *m
			}
			return &p, content, nil
		}
	}

	return nil, nil, nil
}

func (n *NotionIntegration) AddEntry(websites ...*protocol.Website) {
	entries := make(map[string]string) // url:mail

	for _, w := range websites {
		if len(w.Mails) != 0 {
			entries[w.BaseUrl] = strings.Join(w.Mails, " ")
		} else {
			entries[w.BaseUrl] = " "
		}
	}

	var wg sync.WaitGroup
	for url, mails := range entries {
		wg.Add(1)
		go func(u string, m string) {
			defer wg.Done()
			page, content, err := n.SearchDatabase(u)
			if err != nil {
				fmt.Println("err retrieving search ->", err)
				return
			}

			if content[u] != m && page != nil {
				if err := n.UpdateEntry(page, m); err != nil {
					return
				}
			}

			// we got an entry for that website, but mail list is not updated. so let's update.
			if page == nil {
				_, err = n.Client.CreatePage(context.Background(), notion.CreatePageParams{
					ParentType: notion.ParentTypeDatabase,
					ParentID:   n.DatabaseID,
					DatabasePageProperties: &notion.DatabasePageProperties{
						// map[string --> column name]content
						"Emails": notion.DatabasePageProperty{
							Email: &m,
						},
						"Website Address": notion.DatabasePageProperty{
							URL: &u,
						},
					},
				})
				if err != nil {
					fmt.Println("err creating page ->", err)
					return
				}
			}
		}(url, mails)
	}

	wg.Wait()
}

func (n *NotionIntegration) UpdateEntry(entry *notion.Page, mails string) error {
	_, err := n.Client.UpdatePage(context.Background(), entry.ID, notion.UpdatePageParams{
		DatabasePageProperties: notion.DatabasePageProperties{
			"Emails": notion.DatabasePageProperty{
				Email: &mails,
			},
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (n *NotionIntegration) DeleteEntry(websites ...string) {
	var wg sync.WaitGroup

	for _, u := range websites {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()
			page, _, err := n.SearchDatabase(url)
			if err != nil {
				return
			}

			if page != nil {
				if _, err := n.Client.UpdatePage(context.Background(), page.ID, notion.UpdatePageParams{
					Archived: util.PtrBool(true),
				}); err != nil {
					log.Printf("err archiving notion entry -> %v", err)
				}
			}
		}(u)
	}

	wg.Wait()
}
