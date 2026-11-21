package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"mime"
	"os"
	"strings"
	"time"
)

var (
	subject string
)

func sendFlags(fs *flag.FlagSet) {
	fs.StringVar(&subject, "subject", "", "email subject")
}

func isASCII(b []byte) bool {
	for _, c := range b {
		if c >= 0x80 {
			return false
		}
	}
	return true
}

func messageID(from string) string {
	domain := "localhost"
	if i := strings.LastIndex(from, "@"); i >= 0 && i < len(from)-1 {
		domain = strings.Trim(from[i+1:], "<> ")
	}
	var b [16]byte
	rand.Read(b[:])
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(b[:]), domain)
}

func writeHeaders(buf *bytes.Buffer, from string, to []string, subject string) {
	fmt.Fprintf(buf, "From: %s\r\n", from)
	fmt.Fprintf(buf, "To: %s\r\n", strings.Join(to, ", "))
	if subject != "" {
		fmt.Fprintf(buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	}
	fmt.Fprintf(buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(buf, "Message-ID: %s\r\n", messageID(from))
	fmt.Fprintf(buf, "MIME-Version: 1.0\r\n")
}

func send(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s [-subject <subject>] <to-address>...\n",
			os.Args[0], os.Args[1])
		os.Exit(1)
	}

	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fatal(err)
	}

	var buf bytes.Buffer
	writeHeaders(&buf, cfg.smtpUser(), args, subject)
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n")
	if !isASCII(body) {
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: 8bit\r\n")
	}
	fmt.Fprintf(&buf, "\r\n")
	buf.Write(body)

	sender, err := cfg.newSender()
	if err != nil {
		fatal(err)
	}

	if err := sender.send(cfg.smtpUser(), args, buf.Bytes()); err != nil {
		fatal(err)
	}

	fmt.Printf("sent to %s\n", strings.Join(args, ", "))
}
