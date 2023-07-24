package models

import (
	"sync"

	"github.com/wotlk888/gesellschaft-hale/protocol"
	"gopkg.in/gomail.v2"
)

type DialerCreds struct {
	ID       uint   `gorm:"column:id"`
	OwnerID  *uint  `gorm:"column:owner_id"`
	Username string `json:"username" validate:"required" gorm:"column:username"`
	Password string `json:"password" validate:"required" gorm:"column:app_password"`
}

type MailDialer struct {
	Sender *Sender
	Dialer *gomail.Dialer
}

type Sender struct {
	Username string
	Password string
}

func (dc *DialerCreds) Insert(ownerID uint) error {
	// we take id of the owner that want to insert it
	dc.OwnerID = &ownerID
	if err := DB.Table("gmails").Create(&dc).Error; err != nil {
		return err
	}
	return nil
}

func (dc *DialerCreds) Delete() error {
	if err := DB.Table("gmails").Delete(&dc).Error; err != nil {
		return err
	}
	return nil
}

func NewDialer(username, password string) (*MailDialer, error) {
	d := gomail.NewDialer("smtp.gmail.com", 587, username, password)

	s, err := d.Dial()
	if err != nil {
		return nil, protocol.ErrMailerMissingCreds
	}
	s.Close()

	return &MailDialer{
		Sender: &Sender{
			Username: username,
			Password: password,
		},
		Dialer: d,
	}, nil
}

func (md *MailDialer) Send(subject, body string, recipients ...string) map[string]string {
	var syncFailed sync.Map
	failed := make(map[string]string)
	var wg sync.WaitGroup

	for _, r := range recipients {
		wg.Add(1)
		go func(rcp string) {
			defer wg.Done()

			mail := gomail.NewMessage()
			mail.SetHeader("From", md.Sender.Username)
			mail.SetHeader("To", rcp)
			mail.SetHeader("Subject", subject)
			mail.SetBody("text/html", body)

			if err := md.Dialer.DialAndSend(mail); err != nil {
				syncFailed.Store(rcp, err.Error()) // map[email]err
			}
		}(r)
	}

	wg.Wait()

	// easier to work with normal map, so we return a normal one.
	syncFailed.Range(func(key, value any) bool {
		failed[key.(string)] = value.(string)
		return true
	})

	return failed
}
