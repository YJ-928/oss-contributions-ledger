// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestMarkdownGolden(t *testing.T) {
	l := &Ledger{User: "YJ-928"}
	merged := pr("zigflow/zigflow", 615, "MERGED")
	merged.Title = "fix: resolve | pipes\nand newlines"
	merged.ClosedAt = at("2026-10-08")
	merged.Fixes = []Ref{{Number: 462, URL: "https://github.com/zigflow/zigflow/issues/462"}}
	l.Merge([]Entry{
		merged,
		pr("zigflow/zigflow", 622, "OPEN"),
		pr("solo/repo", 3, "CLOSED"),
		{Kind: KindIssue, Repo: "zigflow/zigflow", Number: 493, Title: "dropped task", State: "CLOSED", StateReason: "NOT_PLANNED",
			URL: "https://github.com/zigflow/zigflow/issues/493", CreatedAt: *at("2026-06-21"), ClosedAt: at("2026-06-25")},
		{Kind: KindIssue, Repo: "zigflow/zigflow", Number: 608, Title: "input ignored", State: "CLOSED", StateReason: "COMPLETED",
			URL: "https://github.com/zigflow/zigflow/issues/608", CreatedAt: *at("2026-09-29"), ClosedAt: at("2026-09-30")},
	})

	got := l.Markdown()
	golden := filepath.Join("testdata", "contributions.golden.md")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("Markdown() mismatch; run go test ./... -update and review the diff\n--- got ---\n%s", got)
	}
}

func TestPlural(t *testing.T) {
	for in, want := range map[int]string{0: "0 repositories", 1: "1 repository", 2: "2 repositories"} {
		if got := plural(in, "repository"); got != want {
			t.Errorf("plural(%d) = %q, want %q", in, got, want)
		}
	}
}
