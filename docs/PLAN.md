# upstream-ledger: plan

A GitHub Action that keeps a durable, text-first record of your open-source work in other
people's repositories: pull requests (with the issues they fixed), issues you reported (with
their outcome), and reviews you gave. It writes `CONTRIBUTIONS.md` and keeps the underlying
data in a committed `ledger.json`, so the record survives repositories being deleted, renamed
or made private.

## Why it exists

Profile widgets render live through GitHub's image proxy, so they break when a free host goes
down, rate-limits, or is slow. Card generators that commit static SVGs fix rendering, and one
already covers upstream pull requests (`Sagargupta16/oss-contributions-card-action`). What no
tool does:

| Gap | This project |
|---|---|
| Every run re-searches GitHub, so work in a since-deleted or private repo vanishes | `ledger.json` is append-only; an entry is never dropped |
| Output is images: not searchable, not linkable per item, not accessible | Markdown first |
| Pull requests only | Also issues reported (with outcome), issues each PR fixed, reviews given |
| No roles or highlights | Roles (for example, maintainer) and pinned highlights from a config file |

## Design

- **Data source:** GitHub GraphQL search with the workflow's own `GITHUB_TOKEN`.
  - `is:pr author:LOGIN -user:LOGIN` (with `closingIssuesReferences`)
  - `is:issue author:LOGIN -user:LOGIN`
  - `is:pr reviewed-by:LOGIN -author:LOGIN -user:LOGIN`
- **Store:** `ledger.json`, keyed by URL.
  - A run updates the entries it finds and keeps every entry it does not.
  - Output is sorted, so an unchanged run produces byte-identical files.
- **Render:** `CONTRIBUTIONS.md`, grouped by repository, with a summary line. Titles are escaped for tables.
- **Packaging:** a Docker action that builds the Go binary. It writes files only; the workflow commits them when they change.
- **Config:** inputs for `username`, `token`, `ledger-path`, `output`, `exclude-owners` and `exclude-repos`.
  - Roles and highlights come from an optional YAML file (v1).

## Milestones

1. **MVP:** fetch PRs and issues; merge into `ledger.json`; render `CONTRIBUTIONS.md`; golden-file tests; local run with a token.
2. **Dogfood:** replace the hand-written `CONTRIBUTIONS.md` in `YJ-928/YJ-928`.
3. **v1:**
   - reviews given;
   - roles and highlights config;
   - exclude filters;
   - a compact SVG summary card;
   - docs with examples;
   - Marketplace listing;
   - `v1` tag.
4. **Share:** friends test it on their profiles; collect issues.

## Ground rules

- Apache-2.0, a header on every Go file, `gofmt` and `go vet` clean, and `go test ./...` green in CI.
- No comments that restate code. Docstrings are one line of contract.
- Output must be deterministic, and every network call must have a timeout.
