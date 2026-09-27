package main

import (
	"fmt"
	"os"

	msgformat "github.com/emersion/go-message"
	"github.com/pemistahl/lingua-go"
)

func testFile(ld lingua.LanguageDetector, jd *jevDetector, name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	entity, err := msgformat.Read(f)
	if err != nil && !msgformat.IsUnknownCharset(err) {
		return err
	}
	msg, err := messageFromEntity(entity)
	if err != nil {
		return err
	}
	lang, conf, _ := msg.detectLanguage(ld)
	if jd == nil {
		fmt.Printf("%s  [%s %.0f%%] %s\n", name, lang, conf*100, msg.subject)
		return nil
	}

	answers, err := jd.detect(msg)
	if err != nil {
		fmt.Printf("%s  [%s %.0f%%] %s\n", name, lang, conf*100, msg.subject)
		fmt.Printf("    jev: %s\n", err)
		return nil
	}
	fmt.Printf("%s  [%s] %s\n", name, formatJevAnswers(lang, conf, answers), msg.subject)
	return nil
}

func test(cfg *config, args []string) {
	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	jd, err := cfg.newJevDetector()
	if err != nil {
		fatal(err)
	}
	tot := 0
	for _, name := range args {
		if err := testFile(ld, jd, name); err != nil {
			fmt.Printf("skipping: %s: %s\n", name, err)
			continue
		}
		tot += 1
	}
	fmt.Printf("%d messages\n", tot)
}
