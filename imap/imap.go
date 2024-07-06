package imap

import (
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func Login(server, user, password string) (imapClient *client.Client, err error) {
	// Connect to the server
	imapClient, err = client.DialTLS(server, nil)
	if err != nil {
		err = errors.New("Connect to IMAP Server ERROR - " + err.Error())
		return
	}

	// Login
	if err = imapClient.Login(user, password); err != nil {
		err = errors.New("Login ERROR - " + err.Error())
		return
	}

	return
}

func FetchUnreadEmails(imapClient *client.Client) (emails []*imap.Message, err error) {
	emails = make([]*imap.Message, 0)

	// Select INBOX
	_, err = imapClient.Select("INBOX", true)
	if err != nil {
		return nil, errors.New("Select INBOX mailbox ERROR - " + err.Error())
	}

	// Search for unread emails
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}
	uids, err := imapClient.Search(criteria)
	if err != nil {
		return nil, errors.New("Search Unread Emails ERROR - " + err.Error())
	}

	if len(uids) == 0 {
		return
	}

	// Create a sequence set for the unread messages
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	// Get the whole message body
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}

	// Create a channel to receive messages
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.Fetch(seqset, items, messages)
	}()

	var msgs = make([]*imap.Message, 0)

	for msg := range messages {
		msgs = append(msgs, msg)
	}

	if err := <-done; err != nil {
		return nil, errors.New("Channel done ERROR - " + err.Error())
	}

	return msgs, nil
}
