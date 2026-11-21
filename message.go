package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"unicode"
	"unicode/utf8"

	msgformat "github.com/emersion/go-message"
	"github.com/pemistahl/lingua-go"
	"golang.org/x/net/html"
)

func (msg *message) eachPart(fn func(mediaType string, part *msgformat.Entity)) {
	entity, err := msgformat.New(*msg.header, bytes.NewReader(msg.body))
	if err != nil {
		return
	}
	mr := entity.MultipartReader()
	if mr == nil {
		return
	}
	for {
		part, _ := mr.NextPart()
		if part == nil {
			break
		}
		mediaType, _, _ := part.Header.ContentType()
		if mediaType == "" {
			mediaType = "text/plain"
		}
		fn(mediaType, part)
	}
}

func stripHTML(s string) string {
	var buf strings.Builder
	z := html.NewTokenizer(strings.NewReader(s))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.TextToken {
			buf.Write(z.Text())
		}
	}
	return buf.String()
}

func (msg *message) plainText() string {
	mediaType, _, err := mime.ParseMediaType(msg.header.Get("Content-Type"))
	if err != nil {
		return string(msg.body)
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		text := string(msg.body)
		if mediaType == "text/html" {
			text = stripHTML(text)
		}
		return text
	}

	var plainParts, htmlParts []string
	msg.eachPart(func(mediaType string, part *msgformat.Entity) {
		switch mediaType {
		case "text/plain":
			b, _ := io.ReadAll(part.Body)
			plainParts = append(plainParts, string(b))
		case "text/html":
			b, _ := io.ReadAll(part.Body)
			htmlParts = append(htmlParts, string(b))
		}
	})
	if len(plainParts) > 0 {
		return strings.Join(plainParts, "\n")
	}
	var stripped []string
	for _, h := range htmlParts {
		stripped = append(stripped, stripHTML(h))
	}
	return strings.Join(stripped, "\n")
}

type message struct {
	header  *msgformat.Header
	subject string
	body    []byte
}

func messageFromEntity(entity *msgformat.Entity) (*message, error) {
	body, err := io.ReadAll(entity.Body)
	if err != nil {
		return nil, err
	}

	wd := &mime.WordDecoder{CharsetReader: msgformat.CharsetReader}
	subject, err := wd.DecodeHeader(entity.Header.Get("Subject"))
	if err != nil {
		return nil, err
	}

	return &message{
		header:  &entity.Header,
		subject: subject,
		body:    body,
	}, nil
}

func truncate(s string, n int) string {
	var buf strings.Builder
	pr := rune('\n')
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == '' || r == ' ' ||
			r == ' ' {

			if pr != '\n' {
				buf.WriteRune('\n')
			}
			pr = '\n'
		} else if unicode.IsSpace(r) || !unicode.IsPrint(r) {
			if pr != '\n' {
				pr = ' '
			}
		} else {
			if pr == ' ' {
				buf.WriteRune(' ')
			}
			buf.WriteRune(r)
			pr = r
		}

		if buf.Len() >= n {
			buf.WriteString("...")
			break
		}
	}

	return strings.TrimSpace(buf.String())
}

func (msg *message) printHeaders(fields []string) {
	for _, field := range fields {
		raw := msg.header.Get(field)
		decoded, err := (&mime.WordDecoder{
			CharsetReader: msgformat.CharsetReader,
		}).DecodeHeader(raw)
		if err == nil {
			raw = decoded
		}
		if raw != "" {
			fmt.Printf("%s: %s\n", field, raw)
		}
	}
	fmt.Printf("Subject: %s\n", msg.subject)
}

func (msg *message) formatBody(summarize bool) string {
	mediaType, _, err := mime.ParseMediaType(msg.header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		text := string(msg.body)
		if summarize {
			return truncate(text, 90)
		}
		return text
	}

	var parts []string
	msg.eachPart(func(mediaType string, part *msgformat.Entity) {
		ct := part.Header.Get("Content-Type")
		cte := part.Header.Get("Content-Transfer-Encoding")
		if mediaType == "text/plain" {
			b, _ := io.ReadAll(part.Body)
			text := string(b)
			if summarize {
				var header string
				if cte != "" {
					header = fmt.Sprintf("[%s, %s, %d bytes]\n", ct, cte, len(text))
				} else {
					header = fmt.Sprintf("[%s, %d bytes]\n", ct, len(text))
				}
				parts = append(parts, header+truncate(text, 90))
			} else {
				parts = append(parts, text)
			}
		} else {
			b, _ := io.ReadAll(part.Body)
			if cte != "" {
				parts = append(parts, fmt.Sprintf("[%s, %s, %d bytes]", ct, cte, len(b)))
			} else {
				parts = append(parts, fmt.Sprintf("[%s, %d bytes]", ct, len(b)))
			}
		}
	})
	return strings.Join(parts, "\n")
}

func printPartContent(contentType, transferEnc string, body []byte, indent int) {
	prefix := strings.Repeat("  ", indent)
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		if contentType == "" {
			mediaType = "text/plain"
		} else {
			mediaType = contentType
		}
	}
	if transferEnc != "" {
		fmt.Printf("%s%s [%s]\n", prefix, mediaType, strings.ToLower(strings.TrimSpace(transferEnc)))
	} else {
		fmt.Printf("%s%s\n", prefix, mediaType)
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			partBody, _ := io.ReadAll(part)
			printPartContent(
				part.Header.Get("Content-Type"),
				part.Header.Get("Content-Transfer-Encoding"),
				partBody,
				indent+1,
			)
		}
	}
}

func (msg *message) printContent() {
	printPartContent(
		msg.header.Get("Content-Type"),
		msg.header.Get("Content-Transfer-Encoding"),
		msg.body,
		0,
	)
}

func stripURLs(s string) string {
	var buf strings.Builder
	for _, word := range strings.Fields(s) {
		if strings.Contains(word, "://") {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(word)
	}
	return buf.String()
}

func (msg *message) detectLanguage(ld lingua.LanguageDetector) (lingua.Language, float64, bool) {
	detectText := msg.subject + "\n"
	if len(msg.subject) == utf8.RuneCountInString(msg.subject) {
		detectText += stripURLs(msg.plainText())
	}
	vals := ld.ComputeLanguageConfidenceValues(detectText)
	if len(vals) == 0 {
		return lingua.Unknown, 0, false
	}
	/*
		if vals[1].Value() > 0 {
			fmt.Printf("%s %.0f%% ", vals[1].Language(), vals[1].Value()*100)
		}
	*/
	return vals[0].Language(), vals[0].Value(), true
}
