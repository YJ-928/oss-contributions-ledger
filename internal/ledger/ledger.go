// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"time"
)

type Kind string

const (
	KindPullRequest Kind = "pull_request"
	KindIssue       Kind = "issue"
)

type Ref struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type Entry struct {
	Kind        Kind       `json:"kind"`
	Repo        string     `json:"repo"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	State       string     `json:"state"`
	StateReason string     `json:"stateReason,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	ClosedAt    *time.Time `json:"closedAt,omitempty"`
	Fixes       []Ref      `json:"fixes,omitempty"`
}

type Ledger struct {
	User    string  `json:"user"`
	Entries []Entry `json:"entries"`
}

// Load reads the ledger at path; a missing file is an empty ledger.
func Load(path string) (*Ledger, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Ledger{}, nil
	}
	if err != nil {
		return nil, err
	}
	var l Ledger
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &l, nil
}

// Merge updates entries found in fresh and keeps every entry it does not mention.
func (l *Ledger) Merge(fresh []Entry) {
	byURL := make(map[string]int, len(l.Entries))
	for i, e := range l.Entries {
		byURL[e.URL] = i
	}
	for _, e := range fresh {
		if i, ok := byURL[e.URL]; ok {
			l.Entries[i] = e
			continue
		}
		byURL[e.URL] = len(l.Entries)
		l.Entries = append(l.Entries, e)
	}
	sort.Slice(l.Entries, func(i, j int) bool {
		a, b := l.Entries[i], l.Entries[j]
		if a.Repo != b.Repo {
			return a.Repo < b.Repo
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Number > b.Number
	})
}

// Save writes the ledger to path and reports whether the file content changed.
func (l *Ledger) Save(path string) (bool, error) {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return false, err
	}
	return WriteIfChanged(path, append(data, '\n'))
}

// WriteIfChanged writes data to path only when it differs from the current content.
func WriteIfChanged(path string, data []byte) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
		return false, nil
	}
	return true, os.WriteFile(path, data, 0o644)
}
