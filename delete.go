package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/go-pop3"
)

func archiveName(uid string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, uid)
}

func deleteId(cfg *config, conn *pop3.Conn, id int) (string, error) {
	if cfg.Archive != "" {
		if _, err := os.Stat(cfg.Archive); err != nil {
			return "", fmt.Errorf("archive directory does not exist: %s", cfg.Archive)
		}

		uidls, err := conn.Uidl(id)
		if err != nil {
			return "", err
		}
		if len(uidls) == 0 || uidls[0].UID == "" {
			return "", fmt.Errorf("uidl(%d): no uid returned", id)
		}

		raw, err := retrRaw(conn, id)
		if err != nil {
			return "", err
		}

		path := filepath.Join(cfg.Archive, archiveName(uidls[0].UID)+".eml")
		err = os.WriteFile(path, raw, 0644)
		if err != nil {
			return "", err
		}
		err = conn.Dele(id)
		if err != nil {
			return "", err
		}

		return path, nil
	}

	return "", conn.Dele(id)
}

func delete(cfg *config, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: %s %s <id|range>...\n", os.Args[0], os.Args[1])
		os.Exit(1)
	}

	ids, err := parseIDs(args)
	if err != nil {
		fatal(err)
	}

	conn, err := cfg.newConn()
	if err != nil {
		fatal(err)
	}
	defer conn.Quit()

	for _, id := range ids {
		path, err := deleteId(cfg, conn, id)
		if err != nil {
			fatal(err)
		}

		fmt.Printf("deleted %d", id)
		if path != "" {
			fmt.Printf(" (saved to %s)", path)
		}
		fmt.Println()
	}
}
