// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type repoGroup struct {
	name   string
	prs    []Entry
	issues []Entry
	merged int
}

// Markdown renders the ledger as CONTRIBUTIONS.md.
func (l *Ledger) Markdown() string {
	groups := l.groups()
	merged, issues := 0, 0
	for _, g := range groups {
		merged += g.merged
		issues += len(g.issues)
	}

	var b strings.Builder
	b.WriteString("# Open-source contributions\n\n")
	fmt.Fprintf(&b, "**%s** across **%s**, and **%s** reported.\n",
		plural(merged, "merged pull request"), plural(len(groups), "repository"), plural(issues, "issue"))
	for _, g := range groups {
		fmt.Fprintf(&b, "\n## [%s](https://github.com/%s)\n", g.name, g.name)
		if len(g.prs) > 0 {
			b.WriteString("\n### Pull requests\n\n| PR | Status | Title | Fixes |\n|---|---|---|---|\n")
			for _, e := range g.prs {
				fmt.Fprintf(&b, "| [#%d](%s) | %s | %s | %s |\n", e.Number, e.URL, prStatus(e), cell(e.Title), refs(e.Fixes))
			}
		}
		if len(g.issues) > 0 {
			b.WriteString("\n### Issues reported\n\n| Issue | Opened | Title | Outcome |\n|---|---|---|---|\n")
			for _, e := range g.issues {
				fmt.Fprintf(&b, "| [#%d](%s) | %s | %s | %s |\n", e.Number, e.URL, day(e.CreatedAt), cell(e.Title), issueOutcome(e))
			}
		}
	}
	return b.String()
}

func (l *Ledger) groups() []*repoGroup {
	byName := map[string]*repoGroup{}
	var groups []*repoGroup
	for _, e := range l.Entries {
		g, ok := byName[e.Repo]
		if !ok {
			g = &repoGroup{name: e.Repo}
			byName[e.Repo] = g
			groups = append(groups, g)
		}
		switch e.Kind {
		case KindPullRequest:
			g.prs = append(g.prs, e)
			if e.State == "MERGED" {
				g.merged++
			}
		case KindIssue:
			g.issues = append(g.issues, e)
		}
	}
	for _, g := range groups {
		sort.Slice(g.prs, func(i, j int) bool { return g.prs[i].Number > g.prs[j].Number })
		sort.Slice(g.issues, func(i, j int) bool { return g.issues[i].Number > g.issues[j].Number })
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].merged != groups[j].merged {
			return groups[i].merged > groups[j].merged
		}
		return groups[i].name < groups[j].name
	})
	return groups
}

func prStatus(e Entry) string {
	switch e.State {
	case "MERGED":
		return dated("Merged", e.ClosedAt)
	case "OPEN":
		return "Open"
	default:
		return "Closed"
	}
}

func issueOutcome(e Entry) string {
	if e.State == "OPEN" {
		return "Open"
	}
	switch e.StateReason {
	case "NOT_PLANNED":
		return "Closed as not planned"
	case "DUPLICATE":
		return "Closed as duplicate"
	case "COMPLETED":
		return dated("Completed", e.ClosedAt)
	default:
		return dated("Closed", e.ClosedAt)
	}
}

func refs(rs []Ref) string {
	parts := make([]string, len(rs))
	for i, r := range rs {
		parts[i] = fmt.Sprintf("[#%d](%s)", r.Number, r.URL)
	}
	return strings.Join(parts, ", ")
}

func cell(s string) string {
	return strings.NewReplacer("|", `\|`, "\r", " ", "\n", " ").Replace(strings.TrimSpace(s))
}

func day(t time.Time) string { return t.UTC().Format("2006-01-02") }

func dated(label string, t *time.Time) string {
	if t == nil {
		return label
	}
	return label + " " + day(*t)
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	if strings.HasSuffix(noun, "y") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(noun, "y"))
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
