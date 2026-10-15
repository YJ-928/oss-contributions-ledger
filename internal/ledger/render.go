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
	name        string
	prs         []Entry
	issues      []Entry
	discussions []Entry
	reviews     []Entry
	merged      int
}

// Markdown renders the ledger as CONTRIBUTIONS.md; a non-empty header replaces the default title.
func (l *Ledger) Markdown(header string) string {
	groups := l.groups()

	var b strings.Builder
	if header = strings.TrimSpace(header); header != "" {
		b.WriteString(header + "\n\n")
	} else {
		b.WriteString("# Open-source contributions\n\n")
	}
	b.WriteString(l.summary())
	b.WriteString("\n")
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
		if len(g.discussions) > 0 {
			b.WriteString("\n### Discussions\n\n| Issue | Title | Status |\n|---|---|---|\n")
			for _, e := range g.discussions {
				fmt.Fprintf(&b, "| [#%d](%s) | %s | %s |\n", e.Number, e.URL, cell(e.Title), issueOutcome(e))
			}
		}
		if len(g.reviews) > 0 {
			b.WriteString("\n### Reviews given\n\n| PR | Author | Title | Status |\n|---|---|---|---|\n")
			for _, e := range g.reviews {
				fmt.Fprintf(&b, "| [#%d](%s) | %s | %s | %s |\n", e.Number, e.URL, login(e.Author), cell(e.Title), prStatus(e))
			}
		}
	}
	return b.String()
}

// summary is the one-line count of merged pull requests, repositories, issues and discussions.
func (l *Ledger) summary() string {
	groups := l.groups()
	merged, issues, discussions, reviews := 0, 0, 0, 0
	for _, g := range groups {
		merged += g.merged
		issues += len(g.issues)
		discussions += len(g.discussions)
		reviews += len(g.reviews)
	}
	s := fmt.Sprintf("**%s** across **%s**, and **%s** reported.",
		plural(merged, "merged pull request"), plural(len(groups), "repository"), plural(issues, "issue"))
	if discussions > 0 {
		s += fmt.Sprintf(" Took part in **%s**.", plural(discussions, "discussion"))
	}
	if reviews > 0 {
		s += fmt.Sprintf(" Reviewed **%s**.", plural(reviews, "pull request"))
	}
	return s
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
		case KindDiscussion:
			g.discussions = append(g.discussions, e)
		case KindReview:
			g.reviews = append(g.reviews, e)
		}
	}
	for _, g := range groups {
		sort.Slice(g.prs, func(i, j int) bool { return g.prs[i].Number > g.prs[j].Number })
		sort.Slice(g.issues, func(i, j int) bool { return g.issues[i].Number > g.issues[j].Number })
		sort.Slice(g.discussions, func(i, j int) bool { return g.discussions[i].Number > g.discussions[j].Number })
		sort.Slice(g.reviews, func(i, j int) bool { return g.reviews[i].Number > g.reviews[j].Number })
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

// login links a GitHub user; a deleted account has no login.
func login(name string) string {
	if name == "" {
		return "ghost"
	}
	return fmt.Sprintf("[@%s](https://github.com/%s)", name, name)
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
