// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	ReadmeStart = "<!-- oss-contributions-ledger:start -->"
	ReadmeEnd   = "<!-- oss-contributions-ledger:end -->"
)

// ErrNoMarkers means the file has no start marker, so nothing was written.
var ErrNoMarkers = errors.New("start marker not found")

// ReadmeBlock renders the summary line and the latest merged pull requests.
func (l *Ledger) ReadmeBlock(latest int) string {
	var b strings.Builder
	b.WriteString(l.summary())
	b.WriteString("\n")

	merged := make([]Entry, 0, len(l.Entries))
	for _, e := range l.Entries {
		if e.Kind == KindPullRequest && e.State == "MERGED" && e.ClosedAt != nil {
			merged = append(merged, e)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if !merged[i].ClosedAt.Equal(*merged[j].ClosedAt) {
			return merged[i].ClosedAt.After(*merged[j].ClosedAt)
		}
		return merged[i].URL < merged[j].URL
	})
	if latest > 0 && len(merged) > 0 {
		b.WriteString("\nLatest merged:\n\n")
		for _, e := range merged[:min(latest, len(merged))] {
			fmt.Fprintf(&b, "- [%s#%d](%s): %s (%s)\n", e.Repo, e.Number, e.URL, oneLine(e.Title), day(*e.ClosedAt))
		}
	}
	return b.String()
}

// UpdateReadme replaces the text between the markers in path and reports whether the file changed.
func UpdateReadme(path, block string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	content := string(data)
	start := strings.Index(content, ReadmeStart)
	if start < 0 {
		return false, ErrNoMarkers
	}
	bodyStart := start + len(ReadmeStart)
	end := strings.Index(content[bodyStart:], ReadmeEnd)
	if end < 0 {
		return false, fmt.Errorf("%s: %q has no matching %q", path, ReadmeStart, ReadmeEnd)
	}
	updated := content[:bodyStart] + "\n" + block + content[bodyStart+end:]
	return WriteIfChanged(path, []byte(updated))
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
