package email

import (
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"strings"

	// imapclient "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/charset"
)

type EmailCore struct {
	log    *slog.Logger
	client *imapclient.Client
}

func NewClient(host, port string) (*EmailCore, error) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	options := &imapclient.Options{
		WordDecoder: &mime.WordDecoder{CharsetReader: charset.Reader},
	}

	target := fmt.Sprintf("%s:%s", host, port)
	log.Info("new email client", "target", target)

	client, err := imapclient.DialTLS(target, options)
	if err != nil {
		return nil, err
	}

	return &EmailCore{client: client, log: log}, nil
}

func (e *EmailCore) Login(username, password string) error {
	e.log.Info("logging in", "username", username)
	c := e.client.Login(username, password)
	if err := c.Wait(); err != nil {
		e.log.Error("error logging in", "error", err)
		return err
	}

	return nil
}

func (e *EmailCore) Logout() error {
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

func (e *EmailCore) SelectMailbox(mailbox string) error {
	c := e.client.Select(mailbox, nil)
	if _, err := c.Wait(); err != nil {
		return err
	}

	return nil
}

func (e *EmailCore) Search(query string) ([]uint32, error) {
	// c := e.client.Search(&imap.SearchCriteria{Text: query, NotFlag: []imap.Flag{imap.FlagSeen}}, &imap.SearchOptions{})
	c := e.client.Search(&imap.SearchCriteria{NotFlag: []imap.Flag{imap.FlagSeen}}, &imap.SearchOptions{})
	res, err := c.Wait()
	if err != nil {
		return nil, err
	}

	e.log.Info("email count", "count", len(res.AllSeqNums()))

	for _, uid := range res.AllSeqNums() {
		e.log.Info("processing email", "uid", uid)
		if e.processEmail(uid, query) {
			return res.AllSeqNums(), nil
		}
	}

	return res.AllSeqNums(), nil
}

type customOutput struct{}

func (c customOutput) Write(p []byte) (int, error) {
	fmt.Print(string(p))
	return len(p), nil
}

// func processEmail(c *imapclient.Client, uid uint32, justArchive bool) bool {
func (e *EmailCore) processEmail(uid uint32, query string) bool {
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

func (e *EmailCore) Archive(seqs []uint32) error {
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(seqs...)

	e.log.Info("archiving", "seqSet", seqSet)
	c := e.client.Move(seqSet, "Archive")
	if _, err := c.Wait(); err != nil {
		return err
	}

	return nil
}
