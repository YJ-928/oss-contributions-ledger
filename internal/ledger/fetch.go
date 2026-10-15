// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const searchQuery = `query($q: String!, $after: String) {
  search(query: $q, type: ISSUE, first: 100, after: $after) {
    pageInfo { hasNextPage endCursor }
    nodes {
      __typename
      ... on PullRequest {
        number title url state createdAt mergedAt closedAt
        author { login }
        repository { nameWithOwner }
        closingIssuesReferences(first: 10) { nodes { number url } }
      }
      ... on Issue {
        number title url state stateReason createdAt closedAt
        repository { nameWithOwner }
      }
    }
  }
}`

type Client struct {
	Endpoint string
	Token    string
	HTTP     *http.Client
}

type node struct {
	Typename    string     `json:"__typename"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	State       string     `json:"state"`
	StateReason string     `json:"stateReason"`
	CreatedAt   time.Time  `json:"createdAt"`
	MergedAt    *time.Time `json:"mergedAt"`
	ClosedAt    *time.Time `json:"closedAt"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	ClosingIssuesReferences struct {
		Nodes []Ref `json:"nodes"`
	} `json:"closingIssuesReferences"`
}

type searchResponse struct {
	Data struct {
		Search struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []node `json:"nodes"`
		} `json:"search"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type query struct {
	search string
	kind   Kind
}

// Options picks the optional sections; pull requests and issues are always recorded.
type Options struct {
	Discussions bool // issues the user commented on without opening
	Reviews     bool // pull requests the user reviewed without opening
}

// Contributions returns the user's pull requests and issues in repositories they do not own,
// plus the discussions and reviews there that opts asks for.
func (c *Client) Contributions(ctx context.Context, user string, opts Options) ([]Entry, error) {
	queries := []query{
		{fmt.Sprintf("is:pr author:%s -user:%s", user, user), KindPullRequest},
		{fmt.Sprintf("is:issue author:%s -user:%s", user, user), KindIssue},
	}
	if opts.Discussions {
		queries = append(queries, query{fmt.Sprintf("is:issue commenter:%s -author:%s -user:%s", user, user, user), KindDiscussion})
	}
	if opts.Reviews {
		queries = append(queries, query{fmt.Sprintf("is:pr reviewed-by:%s -author:%s -user:%s", user, user, user), KindReview})
	}
	var entries []Entry
	for _, q := range queries {
		found, err := c.search(ctx, q)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}
	return entries, nil
}

func (c *Client) search(ctx context.Context, q query) ([]Entry, error) {
	var entries []Entry
	var after *string
	for {
		resp, err := c.post(ctx, map[string]any{"query": searchQuery, "variables": map[string]any{"q": q.search, "after": after}})
		if err != nil {
			return nil, err
		}
		for _, n := range resp.Data.Search.Nodes {
			if e, ok := n.entry(); ok {
				switch q.kind {
				case KindDiscussion:
					e.Kind = q.kind
				case KindReview:
					e.Kind, e.Author = q.kind, n.Author.Login
				}
				entries = append(entries, e)
			}
		}
		page := resp.Data.Search.PageInfo
		if !page.HasNextPage {
			return entries, nil
		}
		cursor := page.EndCursor
		after = &cursor
	}
}

func (c *Client) post(ctx context.Context, body any) (*searchResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github graphql: HTTP %d", res.StatusCode)
	}
	var out searchResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("github graphql: %w", err)
	}
	if len(out.Errors) > 0 {
		msgs := make([]string, len(out.Errors))
		for i, e := range out.Errors {
			msgs[i] = e.Message
		}
		return nil, fmt.Errorf("github graphql: %s", strings.Join(msgs, "; "))
	}
	return &out, nil
}

func (n node) entry() (Entry, bool) {
	e := Entry{
		Repo:      n.Repository.NameWithOwner,
		Number:    n.Number,
		Title:     n.Title,
		URL:       n.URL,
		State:     n.State,
		CreatedAt: n.CreatedAt,
		ClosedAt:  n.ClosedAt,
	}
	switch n.Typename {
	case "PullRequest":
		e.Kind = KindPullRequest
		if n.MergedAt != nil {
			e.ClosedAt = n.MergedAt
		}
		e.Fixes = n.ClosingIssuesReferences.Nodes
	case "Issue":
		e.Kind = KindIssue
		e.StateReason = n.StateReason
	default:
		return Entry{}, false
	}
	return e, true
}

// Exclude drops entries whose repository matches a pattern: "owner/repo" or "owner/*".
func Exclude(entries []Entry, patterns []string) []Entry {
	if len(patterns) == 0 {
		return entries
	}
	kept := entries[:0]
	for _, e := range entries {
		if !matchesAny(e.Repo, patterns) {
			kept = append(kept, e)
		}
	}
	return kept
}

func matchesAny(repo string, patterns []string) bool {
	repo = strings.ToLower(repo)
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == repo || (strings.HasSuffix(p, "/*") && strings.HasPrefix(repo, strings.TrimSuffix(p, "*"))) {
			return true
		}
	}
	return false
}
