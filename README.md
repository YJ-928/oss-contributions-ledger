# oss-contributions-ledger

A GitHub Action that keeps a durable, text-first record of your open-source work in other
people's repositories. It writes `CONTRIBUTIONS.md`: your pull requests (with the issues they
fixed), the issues you reported (with their outcome), and optionally the issues you took part in,
grouped by repository.

The data behind it lives in a committed `ledger.json`. Each run adds new work and updates what
changed, but never drops an entry. A pull request to a repository that is later deleted,
renamed or made private stays in your record.

## Quick start

Add `.github/workflows/contributions.yml` to the repository that should hold the ledger, usually
your profile repository:

```yaml
name: Contributions ledger

on:
  schedule:
    - cron: "23 4 * * *"
  workflow_dispatch:

permissions:
  contents: write

jobs:
  ledger:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - id: ledger
        uses: YJ-928/oss-contributions-ledger@v1
      - if: steps.ledger.outputs.changed == 'true'
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add ledger.json CONTRIBUTIONS.md README.md
          git commit -m "chore: update the contributions ledger"
          git push
```

Then link it from your README: `[All contributions](CONTRIBUTIONS.md)`.

## Inputs

| Input | Default | Description |
|---|---|---|
| `username` | the repository owner | GitHub login whose contributions are recorded |
| `token` | the workflow's token | Token for the GitHub GraphQL API; the default is enough for public work |
| `ledger` | `ledger.json` | Path of the data file. Commit it: it is the durable record |
| `output` | `CONTRIBUTIONS.md` | Path of the rendered Markdown |
| `exclude` | | Repositories to leave out, as `owner/repo` or `owner/*`, comma-separated |
| `discussions` | `false` | `true` also records issues you commented on without opening them |
| `header` | | Markdown file shown at the top instead of the default title, for example your roles |
| `readme` | | A file, usually `README.md`, whose marked block gets the summary and your latest merged pull requests |
| `readme-latest` | `3` | How many latest merged pull requests the README block lists |

Output `changed` is `"true"` when the ledger, the Markdown or the README changed, so the workflow commits only
then. Unchanged runs produce byte-identical files.

## Show it on your profile README

Add the markers where the block should go:

```markdown
<!-- oss-contributions-ledger:start -->
<!-- oss-contributions-ledger:end -->
```

Then set `readme: README.md` and add `README.md` to the files the workflow commits. Each run
rewrites only the text between the markers, for example:

```markdown
**10 merged pull requests** across **1 repository**, and **3 issues** reported.

Latest merged:

- [zigflow/zigflow#626](https://github.com/zigflow/zigflow/pull/626): refactor!: remove the GetVersion gates (2026-10-12)
```

If the markers are missing, the run leaves the file alone and prints a warning.

## Run it locally

```sh
GITHUB_TOKEN=<a token with public read access> go run . -user <login>
```

## License

Apache-2.0
