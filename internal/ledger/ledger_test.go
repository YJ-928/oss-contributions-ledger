// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func at(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func pr(repo string, n int, state string) Entry {
	return Entry{
		Kind: KindPullRequest, Repo: repo, Number: n, Title: "pr", State: state,
		URL: "https://github.com/" + repo + "/pull/" + strconv.Itoa(n), CreatedAt: *at("2026-09-01"),
	}
}

func TestMergeKeepsEntriesNoLongerFound(t *testing.T) {
	l := &Ledger{}
	l.Merge([]Entry{pr("a/x", 1, "MERGED"), pr("a/x", 2, "OPEN")})
	l.Merge([]Entry{pr("a/x", 2, "MERGED")})

	if len(l.Entries) != 2 {
		t.Fatalf("got %d entries, want 2: a vanished entry must be kept", len(l.Entries))
	}
	for _, e := range l.Entries {
		if e.State != "MERGED" {
			t.Errorf("#%d state = %s, want MERGED", e.Number, e.State)
		}
	}
}

func TestMergeOrderIsDeterministic(t *testing.T) {
	a, b := &Ledger{}, &Ledger{}
	a.Merge([]Entry{pr("b/y", 3, "OPEN"), pr("a/x", 1, "MERGED"), pr("a/x", 2, "MERGED")})
	b.Merge([]Entry{pr("a/x", 2, "MERGED"), pr("b/y", 3, "OPEN"), pr("a/x", 1, "MERGED")})

	for i := range a.Entries {
		if a.Entries[i].URL != b.Entries[i].URL {
			t.Fatalf("entry %d: %s != %s", i, a.Entries[i].URL, b.Entries[i].URL)
		}
	}
}

func TestSaveReportsChangeOnlyWhenContentDiffers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	l := &Ledger{User: "u"}
	l.Merge([]Entry{pr("a/x", 1, "MERGED")})

	if changed, err := l.Save(path); err != nil || !changed {
		t.Fatalf("first save: changed=%t err=%v, want true", changed, err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := reloaded.Save(path); err != nil || changed {
		t.Fatalf("second save: changed=%t err=%v, want false", changed, err)
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	l, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil || len(l.Entries) != 0 {
		t.Fatalf("got %v, %v; want empty ledger", l, err)
	}
}

func TestExclude(t *testing.T) {
	in := []Entry{pr("Org/One", 1, "MERGED"), pr("org/two", 2, "MERGED"), pr("other/x", 3, "MERGED")}
	got := Exclude(in, []string{"org/*", " Other/X "})
	if len(got) != 0 {
		t.Fatalf("got %d entries, want 0", len(got))
	}
	if got := Exclude([]Entry{pr("organic/x", 4, "MERGED")}, []string{"org/*"}); len(got) != 1 {
		t.Fatal(`"org/*" must not match "organic/x"`)
	}
}
