package main

import (
	"context"
	"fmt"
	"log"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func actionExtractMails(b *Browser, w *protocol.Website) {
	patterns := []string{"info", "more", "contact", "about", "legal", "privacy"}
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	successCh := make(chan struct{}, 1)
	errorCh := make(chan error, 1)
	go func() {
		page, err := b.createPage()
		if err != nil {
			log.Printf("error creating page: %v", err)
			errorCh <- err
		}
		stepExtractPaths(page, w, patterns)
		stepExtractMails(page, w)
		successCh <- struct{}{}
		if err := page.Close(); err != nil {
			log.Printf("error closing page: %v", err)
			errorCh <- err
		}
	}()

	select {
	case <-successCh:
		break
	case <-errorCh:
		break
	case <-ctx.Done():
		fmt.Println("context timed out for", w.BaseUrl)
		break
	}
}
