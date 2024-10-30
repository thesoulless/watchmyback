package email

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"jaytaylor.com/html2text"
)

type Conf struct {
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	LogLevel *slog.Level
}

type Core struct {
	conf   Conf
	log    *slog.Logger
	client *imapclient.Client
	done   chan struct{}
	ticker *time.Ticker
}

func New(conf Conf) (*Core, error) {
	l := slog.LevelInfo
	if conf.LogLevel != nil {
		l = *conf.LogLevel
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	}))

	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}

	target := fmt.Sprintf("%s:%s", conf.Host, conf.Port)
	log.Debug("new email client", "target", target)

	client, err := imapclient.DialTLS(target, options)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(1 * time.Minute)

	return &Core{
		client: client,
		log:    log,
		conf:   conf,
		done:   make(chan struct{}, 1),
		ticker: ticker,
	}, nil
}

func (e *Core) Login(username, password string) error {
	e.log.Debug("logging in", "username", username)
	c := e.client.Login(username, password)
	if err := c.Wait(); err != nil {
		e.log.Error("error logging in", "error", err)
		return err
	}

	return nil
}

func (e *Core) Logout() error {
	c := e.client.Logout()
	if err := c.Wait(); err != nil {
		return err
	}

	err := e.client.Close()
	if err != nil {
		return err
	}

	return nil
}

func (e *Core) SelectMailbox(mailbox string) error {
	c := e.client.Select(mailbox, nil)
	if _, err := c.Wait(); err != nil {
		return err
	}

	return nil
}

var (
	ErrClientError = errors.New("client error")
	ErrNotFound    = errors.New("not found")
)

// Body reads an email by sequence number and returns the email body
func (e *Core) Body(seqnum uint32) (string, error) {
	e.log.Debug("reading", "seqnum", seqnum)

	err := e.healthCheck()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrClientError, err)
	}

	c := e.client.Fetch(imap.SeqSetNum(seqnum), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{Peek: true},
		},
	})
	defer c.Close()

	msg := c.Next()
	if msg == nil {
		e.log.Error("FETCH command returned no message")
		return "", ErrNotFound
	}

	var body string
	var bodySection imapclient.FetchItemDataBodySection
	ok := false
	for {
		item := msg.Next()
		if item == nil {
			break
		}
		bodySection, ok = item.(imapclient.FetchItemDataBodySection)
		if ok {
			break
		}
	}
	if !ok {
		e.log.Debug("FETCH command did not return body section")
		return "", fmt.Errorf("%w: %v", ErrClientError, "FETCH command did not return body section")
	}

	// read the message via the go-message library
	mr, err := mail.CreateReader(bodySection.Literal)
	if err != nil {
		e.log.Error("failed to create mail reader", "error", err)
		return "", fmt.Errorf("%w: %v", ErrClientError, err)
	}

	// process the message's parts
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			e.log.Error("failed to read message part", "error", err)
			return "", fmt.Errorf("%w: %v", ErrClientError, err)
		}

		switch p.Header.(type) {
		case *mail.InlineHeader:
			b, _ := io.ReadAll(p.Body)
			body = string(b)
			// case *mail.AttachmentHeader:
		}
	}

	// convert HTML to plain text
	body, err = html2text.FromString(body)
	if err != nil {
		e.log.Error("failed to convert html to text", "error", err)
		return "", fmt.Errorf("%w: %v", ErrClientError, err)
	}

	return body, nil
}

func (e *Core) Search(query string, from string) ([]string, []uint32, error) {
	e.log.Debug("searching", "query", query)

	err := e.healthCheck()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrClientError, err)
	}

	header := []imap.SearchCriteriaHeaderField{
		{Key: "Subject", Value: query},
	}

	if from != "" {
		header = append(header, imap.SearchCriteriaHeaderField{Key: "From", Value: from})
	}

	c := e.client.Search(&imap.SearchCriteria{
		Header:  header,
		NotFlag: []imap.Flag{}}, &imap.SearchOptions{})
	res, err := c.Wait()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrClientError, err)
	}

	seqnums := res.AllSeqNums()

	e.log.Debug("email count", "count", len(seqnums))

	numSet := imap.SeqSetNum(seqnums...)

	if len(seqnums) == 0 {
		return nil, nil, ErrNotFound
	}

	fetchCmd := e.client.Fetch(numSet, &imap.FetchOptions{
		Envelope: true,
		UID:      true,
		BodySection: []*imap.FetchItemBodySection{
			{Peek: true, Specifier: imap.PartSpecifierHeader},
		},
	})
	defer fetchCmd.Close()

	var result []string
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		data, err := msg.Collect()
		if err != nil {
			e.log.Error("failed to collect msg", "error", err)
			return nil, seqnums, fmt.Errorf("%w: %v", ErrClientError, err)
		}

		result = append(result, data.Envelope.Subject)
	}

	return result, seqnums, nil
}

func (e *Core) healthCheck() error {
	e.log.Debug("health check")

	state := e.client.State()
	switch state {

	case imap.ConnStateNone, imap.ConnStateNotAuthenticated, imap.ConnStateLogout:
		options := &imapclient.Options{
			WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
		}

		target := fmt.Sprintf("%s:%s", e.conf.Host, e.conf.Port)

		var err error
		e.client, err = imapclient.DialTLS(target, options)
		if err != nil {
			e.log.Error("error reconnecting to the imap server", "error", err)
			return errors.New("error reconnecting to the imap server")
		}

		e.client.Login(e.conf.Username, e.conf.Password)
		e.SelectMailbox("INBOX")
		return nil
	default:
		return nil
	}
}

type customOutput struct{}

func (c customOutput) Write(p []byte) (int, error) {
	fmt.Print(string(p))
	return len(p), nil
}

func (e *Core) Move(seqs []uint32, mailbox string) error {
	seqSet := imap.SeqSetNum(seqs...)

	e.log.Debug("moving", "seqSet", seqSet, "mailbox", mailbox)
	c := e.client.Move(seqSet, mailbox)
	if _, err := c.Wait(); err != nil {
		return err
	}

	return nil
}

func (e *Core) Archive(seqs []uint32) error {
	seqSet := imap.SeqSetNum(seqs...)

	e.log.Debug("archiving", "seqSet", seqSet)
	c := e.client.Move(seqSet, "Archive")
	if _, err := c.Wait(); err != nil {
		return err
	}

	return nil
}

func (e *Core) Close() error {
	e.log.Debug("closing email client")
	e.ticker.Stop()
	<-e.done

	return e.client.Close()
}
