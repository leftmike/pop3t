package main

import (
	"fmt"
	"os"
	"path/filepath"

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
	desc, err := jd.describe(msg, lang, conf)
	fmt.Printf("%s  [%s] %s\n", filepath.Base(name), desc, msg.subject)
	printJevError(err)
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
