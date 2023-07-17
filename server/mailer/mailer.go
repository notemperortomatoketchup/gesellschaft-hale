package mailer

import "gopkg.in/gomail.v2"

type MailDialer struct {
	host     string
	port     int
	username string
	password string
}

func (m *MailDialer) New() {
	gomail.NewDialer(m.host, m.port, m.username, m.password)
}
