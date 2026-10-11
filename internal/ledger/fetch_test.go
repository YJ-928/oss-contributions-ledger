// Copyright 2026 Yash B Joshi
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const prPage1 = `{"data":{"search":{"pageInfo":{"hasNextPage":true,"endCursor":"c1"},"nodes":[
 {"__typename":"PullRequest","number":615,"title":"fix: resolve inputs","url":"https://github.com/zigflow/zigflow/pull/615",
  "state":"MERGED","createdAt":"2026-10-08T05:00:00Z","mergedAt":"2026-10-08T18:00:00Z","closedAt":"2026-10-08T18:00:00Z",
  "repository":{"nameWithOwner":"zigflow/zigflow"},
  "closingIssuesReferences":{"nodes":[{"number":462,"url":"https://github.com/zigflow/zigflow/issues/462"}]}}]}}}`

const prPage2 = `{"data":{"search":{"pageInfo":{"hasNextPage":false,"endCursor":""},"nodes":[
 {"__typename":"PullRequest","number":7,"title":"docs","url":"https://github.com/a/b/pull/7","state":"OPEN",
  "createdAt":"2026-10-01T00:00:00Z","mergedAt":null,"closedAt":null,"repository":{"nameWithOwner":"a/b"},
  "closingIssuesReferences":{"nodes":[]}}]}}}`

const issuePage = `{"data":{"search":{"pageInfo":{"hasNextPage":false,"endCursor":""},"nodes":[
 {"__typename":"Issue","number":493,"title":"dropped task","url":"https://github.com/zigflow/zigflow/issues/493",
  "state":"CLOSED","stateReason":"NOT_PLANNED","createdAt":"2026-06-21T00:00:00Z","closedAt":"2026-06-25T00:00:00Z",
  "repository":{"nameWithOwner":"zigflow/zigflow"}}]}}}`

func TestContributionsPaginatesAndParses(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("missing bearer token")
		}
		var body struct {
			Variables struct {
				Q     string  `json:"q"`
				After *string `json:"after"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		queries = append(queries, body.Variables.Q)
		switch {
		case strings.HasPrefix(body.Variables.Q, "is:issue"):
			fmt.Fprint(w, issuePage)
		case body.Variables.After == nil:
			fmt.Fprint(w, prPage1)
		default:
			fmt.Fprint(w, prPage2)
		}
	}))
	defer srv.Close()

	c := &Client{Endpoint: srv.URL, Token: "tok", HTTP: srv.Client()}
	got, err := c.Contributions(context.Background(), "YJ-928")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 3 {
		t.Fatalf("got %d entries, want 3", len(got))
	}
	if want := "is:pr author:YJ-928 -user:YJ-928"; queries[0] != want {
		t.Errorf("first query = %q, want %q", queries[0], want)
	}
	merged := got[0]
	if merged.Kind != KindPullRequest || merged.State != "MERGED" || merged.ClosedAt == nil || len(merged.Fixes) != 1 || merged.Fixes[0].Number != 462 {
		t.Errorf("merged PR parsed wrong: %+v", merged)
	}
	if issue := got[2]; issue.Kind != KindIssue || issue.StateReason != "NOT_PLANNED" {
		t.Errorf("issue parsed wrong: %+v", issue)
	}
}

func TestContributionsSurfacesGraphQLErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"errors":[{"message":"rate limited"}]}`)
	}))
	defer srv.Close()

	c := &Client{Endpoint: srv.URL, Token: "tok", HTTP: srv.Client()}
	if _, err := c.Contributions(context.Background(), "u"); err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("err = %v, want the GraphQL error", err)
	}
}

func TestContributionsFailsOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := &Client{Endpoint: srv.URL, Token: "bad", HTTP: srv.Client()}
	if _, err := c.Contributions(context.Background(), "u"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want HTTP 401", err)
	}
}
