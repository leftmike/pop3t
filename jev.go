package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leftmike/gjevt/jev"
	"github.com/pemistahl/lingua-go"
)

//go:embed email_questions.json
var emailQuestionsJSON []byte

type jevDetector struct {
	client    *jev.Client
	questions map[string]jev.Question
}

func (cfg *config) newJevDetector() (*jevDetector, error) {
	if cfg.Jev.APIKey == "" {
		return nil, nil
	}

	var questions map[string]jev.Question
	if err := json.Unmarshal(emailQuestionsJSON, &questions); err != nil {
		return nil, err
	}

	return &jevDetector{
		client: jev.NewClient(jev.Config{
			APIKey:  cfg.Jev.APIKey,
			BaseURL: cfg.Jev.BaseURL,
			Model:   cfg.Jev.Model,
		}),
		questions: questions,
	}, nil
}

func (msg *message) jevState() map[string]any {
	state := map[string]any{}
	for _, field := range []string{"From", "Reply-To", "Return-Path", "To", "Date"} {
		if v := msg.header.Get(field); v != "" {
			state[strings.ToLower(field)] = v
		}
	}
	state["subject"] = msg.subject
	state["body"] = extractPlainText(msg.header.Get("Content-Type"),
		msg.header.Get("Content-Transfer-Encoding"), msg.body)
	return state
}

func (jd *jevDetector) detect(msg *message) (map[string]jev.Answer, error) {
	resp, _, err := jd.client.DecideTruncated(context.Background(), jev.DecisionRequest{
		State:     msg.jevState(),
		Questions: jd.questions,
	})
	if err != nil {
		return nil, err
	}
	return resp.Answers, nil
}

func formatNoul(name string, a jev.Answer) string {
	if a.Noul == nil {
		return ""
	}
	if *a.Noul >= 0.5 {
		return fmt.Sprintf("%s %.2f", name, *a.Noul)
	}
	return fmt.Sprintf("not %s %.2f", name, 1-*a.Noul)
}

func formatJevAnswers(lang lingua.Language, conf float64,
	answers map[string]jev.Answer) string {

	parts := []string{formatLanguages(lang, conf, answers["language"])}
	for _, name := range []string{"spam", "malicious"} {
		if s := formatNoul(name, answers[name]); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}

func formatLanguages(lang lingua.Language, conf float64, a jev.Answer) string {
	if a.Choice == "" {
		return fmt.Sprintf("%s %.0f%%", lang, conf*100)
	}
	jevConf := "?"
	if a.Confidence != nil {
		jevConf = fmt.Sprintf("%.2f", *a.Confidence)
	}
	jevLang := a.Choice
	if l, ok := languages[strings.ToLower(a.Choice)]; ok {
		if l == lang {
			return fmt.Sprintf("%s %.0f%%/%s", lang, conf*100, jevConf)
		}
		jevLang = l.String()
	}
	return fmt.Sprintf("%s %.0f%% / %s %s", lang, conf*100, jevLang, jevConf)
}
