// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mergedPR(repo string, n int, day string) Entry {
	e := pr(repo, n, "MERGED")
	e.Title = "fix:  a\nmultiline   title"
	e.ClosedAt = at(day)
	return e
}

func TestReadmeBlockListsLatestMergedNewestFirst(t *testing.T) {
	l := &Ledger{}
	l.Merge([]Entry{
		mergedPR("a/x", 1, "2026-09-01"),
		mergedPR("a/x", 2, "2026-10-01"),
		mergedPR("b/y", 3, "2026-10-05"),
		pr("a/x", 4, "OPEN"),
	})

	got := l.ReadmeBlock(2)

	want := "**3 merged pull requests** across **2 repositories**, and **0 issues** reported.\n" +
		"\nLatest merged:\n\n" +
		"- [b/y#3](https://github.com/b/y/pull/3): fix: a multiline title (2026-10-05)\n" +
		"- [a/x#2](https://github.com/a/x/pull/2): fix: a multiline title (2026-10-01)\n"
	if got != want {
		t.Fatalf("ReadmeBlock(2) =\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(l.ReadmeBlock(0), "Latest merged") {
		t.Fatal("ReadmeBlock(0) must list no pull requests")
	}
}

func writeReadme(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpdateReadmeReplacesOnlyBetweenMarkers(t *testing.T) {
	path := writeReadme(t, "# Me\n\n"+ReadmeStart+"\nold\n"+ReadmeEnd+"\n\nRest stays.\n")

	changed, err := UpdateReadme(path, "new block\n")
	if err != nil || !changed {
		t.Fatalf("first update: changed=%t err=%v", changed, err)
	}
	got, _ := os.ReadFile(path)
	want := "# Me\n\n" + ReadmeStart + "\nnew block\n" + ReadmeEnd + "\n\nRest stays.\n"
	if string(got) != want {
		t.Fatalf("README =\n%s\nwant\n%s", got, want)
	}

	if changed, err := UpdateReadme(path, "new block\n"); err != nil || changed {
		t.Fatalf("second update: changed=%t err=%v, want unchanged", changed, err)
	}
}

func TestUpdateReadmeWithoutMarkersLeavesTheFileAlone(t *testing.T) {
	path := writeReadme(t, "# Me\n")

	if _, err := UpdateReadme(path, "block\n"); !errors.Is(err, ErrNoMarkers) {
		t.Fatalf("err = %v, want ErrNoMarkers", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "# Me\n" {
		t.Fatalf("file changed: %q", got)
	}
}

func TestUpdateReadmeWithoutEndMarkerFails(t *testing.T) {
	path := writeReadme(t, ReadmeStart+"\nno end\n")

	if _, err := UpdateReadme(path, "block\n"); err == nil || errors.Is(err, ErrNoMarkers) {
		t.Fatalf("err = %v, want a missing end marker error", err)
	}
}
