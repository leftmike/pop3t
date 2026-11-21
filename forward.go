package main

import (
	"bytes"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"

	msgformat "github.com/emersion/go-message"
)

func wrapForward(from string, to []string, raw []byte) []byte {
	var subject string
	if entity, _ := msgformat.Read(bytes.NewReader(raw)); entity != nil {
		wd := &mime.WordDecoder{CharsetReader: msgformat.CharsetReader}
		subject = entity.Header.Get("Subject")
		if decoded, err := wd.DecodeHeader(subject); err == nil {
			subject = decoded
		}
	}
	if !strings.HasPrefix(strings.ToLower(subject), "fwd:") {
		subject = "Fwd: " + subject
	}

	var buf bytes.Buffer
	writeHeaders(&buf, from, to, subject)
	mw := multipart.NewWriter(&buf)
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", mw.Boundary())

	h := textproto.MIMEHeader{}
	h.Set("Content-Type", "message/rfc822")
	h.Set("Content-Disposition", "inline")
	if !isASCII(raw) {
		h.Set("Content-Transfer-Encoding", "8bit")
	}
	w, _ := mw.CreatePart(h)
	w.Write(raw)
	mw.Close()
	return buf.Bytes()
}

func forward(cfg *config, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <addr> <id|range>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}

	to := args[0]

	ids, err := parseIDs(args[1:])
	if err != nil {
		fatal(err)
	}
	sender, err := cfg.newSender()
	if err != nil {
		fatal(err)
	}

	conn, err := cfg.newConn()
	if err != nil {
		fatal(err)
	}
	defer conn.Quit()

	for _, id := range ids {
		raw, err := retrRaw(conn, id)
		if err != nil {
			fatal(err)
		}

		msg := wrapForward(cfg.smtpUser(), []string{to}, raw)
		err = sender.send(cfg.smtpUser(), []string{to}, msg)
		if err != nil {
			fatal(err)
		}

		fmt.Printf("forwarded %d to %s\n", id, to)
	}
}
