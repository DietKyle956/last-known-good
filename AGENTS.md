## Agent skills

### Issue tracker

GitHub issues via `gh` CLI. External PRs are not a triage surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Default label vocabulary — `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Workflow rules

- Never push to `main` and never merge PRs. Only push to working branches and submit PRs.
- The user reviews and merges PRs. Monitor the associated issue and wait for it to close.
- When the issue closes, update `context.md` and push to a working branch again.

### Domain docs

Single-context. See `docs/agents/domain.md`.
