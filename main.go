// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/YJ-928/oss-contributions-ledger/internal/ledger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "oss-contributions-ledger:", err)
		os.Exit(1)
	}
}

func run() error {
	user := flag.String("user", env("INPUT_USERNAME", os.Getenv("GITHUB_REPOSITORY_OWNER")), "GitHub login to record")
	ledgerPath := flag.String("ledger", env("INPUT_LEDGER", "ledger.json"), "path of the ledger data file")
	output := flag.String("output", env("INPUT_OUTPUT", "CONTRIBUTIONS.md"), "path of the rendered Markdown")
	exclude := flag.String("exclude", os.Getenv("INPUT_EXCLUDE"), `comma-separated "owner/repo" or "owner/*" to leave out`)
	discussions := flag.Bool("discussions", env("INPUT_DISCUSSIONS", "false") == "true", "also record issues the user commented on")
	headerPath := flag.String("header", os.Getenv("INPUT_HEADER"), "Markdown file shown at the top instead of the default title")
	readme := flag.String("readme", os.Getenv("INPUT_README"), "file whose marked block gets the summary and latest merged pull requests")
	readmeLatest := flag.String("readme-latest", env("INPUT_README_LATEST", "3"), "how many latest merged pull requests the block lists")
	flag.Parse()

	token := env("INPUT_TOKEN", os.Getenv("GITHUB_TOKEN"))
	if *user == "" || token == "" {
		return errors.New("a username and a token (GITHUB_TOKEN) are required")
	}
	latest, err := strconv.Atoi(strings.TrimSpace(*readmeLatest))
	if err != nil || latest < 0 {
		return fmt.Errorf("readme-latest must be a whole number, got %q", *readmeLatest)
	}
	header := ""
	if *headerPath != "" {
		data, err := os.ReadFile(*headerPath)
		if err != nil {
			return fmt.Errorf("header: %w", err)
		}
		header = string(data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := &ledger.Client{
		Endpoint: env("GITHUB_GRAPHQL_URL", "https://api.github.com/graphql"),
		Token:    token,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
	fresh, err := client.Contributions(ctx, *user, *discussions)
	if err != nil {
		return err
	}

	l, err := ledger.Load(*ledgerPath)
	if err != nil {
		return err
	}
	l.User = *user
	l.Merge(ledger.Exclude(fresh, split(*exclude)))

	dataChanged, err := l.Save(*ledgerPath)
	if err != nil {
		return err
	}
	docChanged, err := ledger.WriteIfChanged(*output, []byte(l.Markdown(header)))
	if err != nil {
		return err
	}
	readmeChanged := false
	if *readme != "" {
		readmeChanged, err = ledger.UpdateReadme(*readme, l.ReadmeBlock(latest))
		if errors.Is(err, ledger.ErrNoMarkers) {
			fmt.Printf("::warning::%s has no %s marker; add %s and %s where the block should go\n", *readme, ledger.ReadmeStart, ledger.ReadmeStart, ledger.ReadmeEnd)
		} else if err != nil {
			return err
		}
	}
	changed := dataChanged || docChanged || readmeChanged
	fmt.Printf("oss-contributions-ledger: %d entries, changed=%t\n", len(l.Entries), changed)
	return setOutput("changed", fmt.Sprint(changed))
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func setOutput(name, value string) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s=%s\n", name, value)
	return err
}
