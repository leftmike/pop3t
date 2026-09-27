package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/leftmike/gjevt/jev"
	"github.com/pemistahl/lingua-go"
)

var emailQuestions = map[string]jev.Question{
	"spam": {
		Type:         "noul",
		Instructions: "Is this email spam?",
	},
	"malicious": {
		Type:         "noul",
		Instructions: "Is this email malicious?",
	},
	"language": {
		Type:         "choice",
		Instructions: "What language is this email written in?",
		Criteria: map[string]string{
			"afrikaans":   "Die taal is Afrikaans",
			"albanian":    "Gjuha është shqip",
			"arabic":      "اللغة هي العربية",
			"armenian":    "Լեզուն հայերենն է",
			"azerbaijani": "Dil Azərbaycan dilidir",
			"basque":      "Hizkuntza euskara da",
			"belarusian":  "Мова — беларуская",
			"bengali":     "ভাষাটি বাংলা",
			"bokmal":      "Språket er bokmål",
			"bosnian":     "Jezik je bosanski",
			"bulgarian":   "Езикът е български",
			"catalan":     "La llengua és el català",
			"chinese":     "语言是中文",
			"croatian":    "Jezik je hrvatski",
			"czech":       "Jazyk je čeština",
			"danish":      "Sproget er dansk",
			"dutch":       "De taal is Nederlands",
			"english":     "The language is English",
			"esperanto":   "La lingvo estas Esperanto",
			"estonian":    "Keel on eesti keel",
			"finnish":     "Kieli on suomi",
			"french":      "La langue est le français",
			"ganda":       "Olulimi lwe Luganda",
			"georgian":    "ენა არის ქართული",
			"german":      "Die Sprache ist Deutsch",
			"greek":       "Η γλώσσα είναι τα Ελληνικά",
			"gujarati":    "ભાષા ગુજરાતી છે",
			"hebrew":      "השפה היא עברית",
			"hindi":       "भाषा हिन्दी है",
			"hungarian":   "A nyelv magyar",
			"icelandic":   "Tungumálið er íslenska",
			"indonesian":  "Bahasanya adalah Bahasa Indonesia",
			"irish":       "Is í an Ghaeilge an teanga",
			"italian":     "La lingua è l'italiano",
			"japanese":    "言語は日本語です",
			"kazakh":      "Тіл — қазақ тілі",
			"korean":      "언어는 한국어입니다",
			"latin":       "Lingua est Latina",
			"latvian":     "Valoda ir latviešu",
			"lithuanian":  "Kalba yra lietuvių",
			"macedonian":  "Јазикот е македонски",
			"malay":       "Bahasanya ialah Bahasa Melayu",
			"maori":       "Ko te reo Māori te reo",
			"marathi":     "भाषा मराठी आहे",
			"mongolian":   "Хэл нь монгол хэл",
			"nynorsk":     "Språket er nynorsk",
			"persian":     "زبان فارسی است",
			"polish":      "Językiem jest polski",
			"portuguese":  "A língua é o português",
			"punjabi":     "ਭਾਸ਼ਾ ਪੰਜਾਬੀ ਹੈ",
			"romanian":    "Limba este română",
			"russian":     "Язык — русский",
			"serbian":     "Језик је српски",
			"shona":       "Mutauro iChiShona",
			"slovak":      "Jazyk je slovenčina",
			"slovene":     "Jezik je slovenščina",
			"somali":      "Luqaddu waa Soomaali",
			"sotho":       "Puo ke Sesotho",
			"spanish":     "El idioma es Español",
			"swahili":     "Lugha ni Kiswahili",
			"swedish":     "Språket är svenska",
			"tagalog":     "Ang wika ay Tagalog",
			"tamil":       "மொழி தமிழ்",
			"telugu":      "భాష తెలుగు",
			"thai":        "ภาษาคือภาษาไทย",
			"tsonga":      "Ririmi i Xitsonga",
			"tswana":      "Puo ke Setswana",
			"turkish":     "Dil Türkçe",
			"ukrainian":   "Мова — українська",
			"urdu":        "زبان اردو ہے",
			"vietnamese":  "Ngôn ngữ là tiếng Việt",
			"welsh":       "Cymraeg yw'r iaith",
			"xhosa":       "Ulwimi sisiXhosa",
			"yoruba":      "Èdè náà ni Yorùbá",
			"zulu":        "Ulimi isiZulu",
		},
	},
}

type jevDetector struct {
	client *jev.Client
}

func (cfg *config) newJevDetector() *jevDetector {
	if cfg.Jev.APIKey == "" {
		return nil
	}

	return &jevDetector{
		client: jev.NewClient(jev.Config{
			APIKey:  cfg.Jev.APIKey,
			BaseURL: cfg.Jev.BaseURL,
			Model:   cfg.Jev.Model,
		}),
	}
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
	if jd == nil {
		return nil, nil
	}
	resp, _, err := jd.client.DecideTruncated(context.Background(), jev.DecisionRequest{
		State:     msg.jevState(),
		Questions: emailQuestions,
	})
	if err != nil {
		return nil, err
	}
	return resp.Answers, nil
}

func (jd *jevDetector) describe(msg *message, lang lingua.Language, conf float64) (string,
	error) {

	answers, err := jd.detect(msg)
	return formatJevAnswers(lang, conf, answers), err
}

func printJevError(err error) {
	if err != nil {
		fmt.Printf("    jev: %s\n", err)
	}
}

func isNoul(a jev.Answer) bool {
	return a.Noul != nil && *a.Noul >= 0.5
}

func formatNoul(name string, a jev.Answer) string {
	if a.Noul == nil {
		return ""
	}
	if isNoul(a) {
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
