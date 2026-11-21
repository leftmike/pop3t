package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/leftmike/gjevt/jev"
	"github.com/pemistahl/lingua-go"
)

func TestParseIDs(t *testing.T) {
	cases := []struct {
		args []string
		ids  []int
		fail bool
	}{
		{args: []string{"1"}, ids: []int{1}},
		{args: []string{"3", "1"}, ids: []int{3, 1}},
		{args: []string{"2-4"}, ids: []int{2, 3, 4}},
		{args: []string{"5-5", "7"}, ids: []int{5, 7}},
		{args: []string{"0"}, fail: true},
		{args: []string{"-1"}, fail: true},
		{args: []string{"4-2"}, fail: true},
		{args: []string{"1-2-3"}, fail: true},
		{args: []string{"x"}, fail: true},
	}
	for _, c := range cases {
		ids, err := parseIDs(c.args)
		if c.fail {
			if err == nil {
				t.Errorf("parseIDs(%v): expected an error", c.args)
			}
		} else if err != nil {
			t.Errorf("parseIDs(%v): %s", c.args, err)
		} else if !slices.Equal(ids, c.ids) {
			t.Errorf("parseIDs(%v): got %v, want %v", c.args, ids, c.ids)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 90, "hello"},
		{"  hello   world  ", 90, "hello world"},
		{"line one\r\n\r\n\r\nline two", 90, "line one\nline two"},
		{"a\tb\x00c", 90, "a b c"},
		{"abcdefghij", 5, "abcde..."},
	}
	for _, c := range cases {
		if got := truncate(c.s, c.n); got != c.want {
			t.Errorf("truncate(%q, %d): got %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestMatchLanguage(t *testing.T) {
	en, fr := lingua.English, lingua.French
	cases := []struct {
		lang    lingua.Language
		filters []lingua.Language
		exclude bool
		want    bool
	}{
		{en, nil, false, true},
		{en, []lingua.Language{en}, false, true},
		{fr, []lingua.Language{en}, false, false},
		{en, []lingua.Language{en}, true, false},
		{fr, []lingua.Language{en}, true, true},
		{fr, []lingua.Language{en, fr}, false, true},
	}
	for _, c := range cases {
		if got := matchLanguage(c.lang, c.filters, c.exclude); got != c.want {
			t.Errorf("matchLanguage(%s, %v, %v): got %v, want %v", c.lang, c.filters, c.exclude,
				got, c.want)
		}
	}
}

func TestMatchJev(t *testing.T) {
	yes, no := 0.9, 0.1
	spam := map[string]jev.Answer{"spam": {Noul: &yes}, "malicious": {Noul: &no}}
	cases := []struct {
		answers map[string]jev.Answer
		filters []string
		exclude bool
		want    bool
	}{
		{spam, nil, false, true},
		{spam, []string{"spam"}, false, true},
		{spam, []string{"malicious"}, false, false},
		{spam, []string{"spam"}, true, false},
		{spam, []string{"malicious"}, true, true},
		{spam, []string{"malicious", "spam"}, false, true},
		{nil, []string{"spam"}, false, false},
	}
	for _, c := range cases {
		if got := matchJev(c.answers, c.filters, c.exclude); got != c.want {
			t.Errorf("matchJev(%v, %v, %v): got %v, want %v", c.answers, c.filters, c.exclude,
				got, c.want)
		}
	}
}

func TestArchiveName(t *testing.T) {
	cases := map[string]string{
		"abc-123_X.y": "abc-123_X.y",
		"../../etc":   ".._.._etc",
		`a/b\c`:       "a_b_c",
		"a b<c>":      "a_b_c_",
	}
	for uid, want := range cases {
		if got := archiveName(uid); got != want {
			t.Errorf("archiveName(%q): got %q, want %q", uid, got, want)
		}
	}
}

func TestPlainText(t *testing.T) {
	cases := []struct {
		file    string
		subject string
		text    string
	}{
		{"base64.eml", "base64",
			"Hello world, this is a test of the emergency broadcast system."},
		{"qp.eml", "quoted printable",
			"Café costs 5=5 dollars, a very long line that is soft wrapped right here.\r\n"},
		{"multipart.eml", "multipart", "Café crème brûlée."},
		{"html-only.eml", "html only", "日本語"},
		{"unknown-encoding.eml", "unknown encoding", "hello there\r\n"},
	}
	for _, c := range cases {
		raw, err := os.ReadFile(filepath.Join("testdata", c.file))
		if err != nil {
			t.Fatal(err)
		}
		entity, err := parseMessage(raw)
		if err != nil {
			t.Errorf("%s: parseMessage: %s", c.file, err)
			continue
		}
		msg, err := messageFromEntity(entity)
		if err != nil {
			t.Errorf("%s: messageFromEntity: %s", c.file, err)
			continue
		}
		if msg.subject != c.subject {
			t.Errorf("%s: subject: got %q, want %q", c.file, msg.subject, c.subject)
		}
		if got := msg.plainText(); got != c.text {
			t.Errorf("%s: plainText: got %q, want %q", c.file, got, c.text)
		}
	}
}
