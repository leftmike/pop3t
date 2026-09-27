package main

import (
	"fmt"
	"os"

	msgformat "github.com/emersion/go-message"
	"github.com/pemistahl/lingua-go"
)

func testFile(ld lingua.LanguageDetector, name string) error {
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
	fmt.Printf("%s  [%s %.0f%%] %s\n", name, lang, conf*100, msg.subject)
	return nil
}

func test(cfg *config, args []string) {
	ld := lingua.NewLanguageDetectorBuilder().FromAllLanguages().Build()
	tot := 0
	for _, name := range args {
		if err := testFile(ld, name); err != nil {
			fmt.Printf("skipping: %s: %s\n", name, err)
			continue
		}
		tot += 1
	}
	fmt.Printf("%d messages\n", tot)
}
