package email

import (
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"strings"
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

func (e *Core) Search(query string) ([]string, error) {
	e.log.Info("searching", "query", query)
	// c := e.client.Search(&imap.SearchCriteria{Text: query, NotFlag: []imap.Flag{imap.FlagSeen}}, &imap.SearchOptions{})

	// c := e.client.Search(&imap.SearchCriteria{NotFlag: []imap.Flag{imap.FlagSeen}}, &imap.SearchOptions{})
	c := e.client.Search(&imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Subject", Value: query},
		},
		NotFlag: []imap.Flag{}}, &imap.SearchOptions{})
	res, err := c.Wait()
	if err != nil {
		return nil, err
	}

	e.log.Info("email count", "count", len(res.AllSeqNums()))

	/*for _, uid := range res.AllSeqNums() {
		e.log.Info("processing email", "uid", uid)
		if e.processEmail(uid, query) {
			return res.AllSeqNums(), nil
		}
	}*/
	// var numSet imap.NumSet
	// numSet := imap.UIDSetNum(res.AllUIDs()...)
	numSet := imap.SeqSetNum(res.AllSeqNums()...)
	// for _, uid := range res.AllUIDs() {
	// numSet = append(numSet, imap.SingleNum(uid))
	// }

	// uidset, ok := res.All.(imap.UIDSet)
	// if !ok {
	// 	e.log.Error("faild to convert uidset")
	// 	return nil, errors.New("invalid uidset")
	// }

	fetchCmd := e.client.Fetch(numSet, &imap.FetchOptions{
		Envelope: true,
		UID:      true,
		BodySection: []*imap.FetchItemBodySection{
			{Peek: true, Specifier: imap.PartSpecifierHeader},
		},
	})
	defer fetchCmd.Close()

	// err = fetchCmd.Wait()
	// if err != nil {
	// 	e.log.Error("failed to fetch", "error", err)
	// 	return nil, errors.New("failed tp fetch")
	// }

	var result []string
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		data, err := msg.Collect()
		if err != nil {
			e.log.Error("failed to collect msg", "error", err)
			return nil, err
		}
		fmt.Println(data.Envelope.Subject)
		fmt.Println(strings.Contains(data.Envelope.Subject, query))

		result = append(result, data.Envelope.Subject)
	}

	return result, nil
}

type customOutput struct{}

func (c customOutput) Write(p []byte) (int, error) {
	fmt.Print(string(p))
	return len(p), nil
}

// func processEmail(c *imapclient.Client, uid uint32, justArchive bool) bool {
func (e *Core) processEmail(uid uint32, query string) bool {
	seqSet := imap.SeqSetNum(uid)

	e.log.Info("fetching", "seqSet", seqSet)

	// BodyStructure     *FetchItemBodyStructure
	// Envelope          bool
	// Flags             bool
	// InternalDate      bool
	// RFC822Size        bool
	// UID               bool
	// BodySection       []*FetchItemBodySection
	// BinarySection     []*FetchItemBinarySection     // requires IMAP4rev2 or BINARY
	// BinarySectionSize []*FetchItemBinarySectionSize // requires IMAP4rev2 or BINARY
	// ModSeq            bool                          // requires CONDSTORE
	// ChangedSince uint64 // requires CONDSTORE

	msgs, err := e.client.Fetch(seqSet, &imap.FetchOptions{Flags: true,
		Envelope:    true,
		RFC822Size:  true,
		BodySection: []*imap.FetchItemBodySection{{Specifier: imap.PartSpecifierHeader}},
	}).Collect()
	if err != nil {
		e.log.Error("fetching message", "error", err)
		return false
	}

	for _, msg := range msgs {
		if msg == nil {
			e.log.Error("nil message", "error", errors.New("server didn't return the message"))
			return false
		}

		e.log.Info("searching email", "query", query, "subject", msg.Envelope.Subject)
		subject := msg.Envelope.Subject
		if strings.Contains(strings.ToLower(subject), strings.ToLower(query)) {
			e.log.Info("found email", "subject", subject)

			// if justArchive {
			// 	moveToArchive(c, seqSet)
			// 	return true
			// }

			e.log.Info("running bash script")

			// Run bash script
			/*cmd := exec.Command("/bin/bash", bashScriptPath, "newytdlp")
			cmd.Env = os.Environ()
			cmd.Dir = bashScriptPath[:strings.LastIndex(bashScriptPath, "/")]
			log.Println("Running bash script in directory:", cmd.Dir)

			cmd.Stdout = customOutput{}
			cmd.Stderr = customOutput{}

			if err = cmd.Run(); err != nil {
				// log.Printf("Error running bash script: %v: %s", err, string(res))
				log.Printf("Error running bash script: %v", err)
				return false
			}

			if cmd.ProcessState.ExitCode() == 0 {
				// sres := string(res)
				log.Println("Bash script executed successfully")
				// Archive the email
				moveToArchive(c, seqSet)

				// Send Slack message
				time.Sleep(5 * time.Minute)
				text := "<!channel> New version of helpercompanion is deployed:\n\n\t\t- yt-dlp updated to the latest version"
				text += "\n\n(it will be installed on your machine in the next auto update)"
				sendSlackMessage(text)

				return true
			}*/
			return true
		}
	}

	return false
}

func (e *Core) Archive(seqs []uint32) error {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqs...)

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
	// e.log.Info("refreshing email", "account", e.conf.Name)

	for range e.ticker.C {
		// @TODO: do we rly need this?
		// e.log.Info("refreshing email", "account", e.conf.Name)
	}

	e.done <- struct{}{}
}

func (e *Core) Close() error {
	e.ticker.Stop()
	<-e.done

	return e.client.Close()
}
