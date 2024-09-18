package email

import (
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
)

type Conf struct {
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
}

type Core struct {
	conf   Conf
	log    *slog.Logger
	client *imapclient.Client
	done   chan struct{}
	ticker *time.Ticker
}

func New(conf Conf) (*Core, error) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}

	target := fmt.Sprintf("%s:%s", conf.Host, conf.Port)
	log.Info("new email client", "target", target)

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
	e.log.Info("logging in", "username", username)
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

func (e *Core) Search(query string) ([]string, []uint32, error) {
	e.log.Info("searching", "query", query)

	err := e.healthCheck()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrClientError, err)
	}

	c := e.client.Search(&imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Subject", Value: query},
		},
		NotFlag: []imap.Flag{}}, &imap.SearchOptions{})
	res, err := c.Wait()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrClientError, err)
	}

	seqnums := res.AllSeqNums()

	e.log.Info("email count", "count", len(seqnums))

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
	e.log.Info("health check")

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

func (e *Core) Archive(seqs []uint32) error {
	seqSet := imap.SeqSetNum(seqs...)

	e.log.Info("archiving", "seqSet", seqSet)
	c := e.client.Move(seqSet, "Archive")
	if _, err := c.Wait(); err != nil {
		return err
	}

	return nil
}

func (e *Core) Run() {
	e.Login(e.conf.Username, e.conf.Password)
	e.SelectMailbox("INBOX")

	for range e.ticker.C {
		// @TODO: do we rly need this?
	}

	e.done <- struct{}{}
}

func (e *Core) Close() error {
	e.log.Info("closing email client")
	e.ticker.Stop()
	<-e.done

	return e.client.Close()
}
