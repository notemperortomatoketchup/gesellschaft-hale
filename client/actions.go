package main

import (
	"context"
	"fmt"

	"github.com/wotlk888/gesellschaft-hale/protocol"
)

func actionExtractMails(b *Browser, w *protocol.Website) {
	patterns := []string{"info", "more", "contact", "about", "legal", "privacy"}
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	successCh := make(chan struct{}, 1)
	go func() {
		page := b.createPage()
		stepExtractPaths(page, w, patterns)
		stepExtractMails(page, w)
		successCh <- struct{}{}
		page.MustClose()
	}()

	select {
	case <-successCh:
		break
	case <-ctx.Done():
		fmt.Println("context timed out for", w.BaseUrl)
		break
	}
}
