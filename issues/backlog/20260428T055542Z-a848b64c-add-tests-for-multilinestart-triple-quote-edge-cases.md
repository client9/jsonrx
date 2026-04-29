{
  "title": "Add tests for multilineStart triple-quote edge cases",
  "id": "20260428T055542Z-a848b64c",
  "state": "backlog",
  "created": "2026-04-28T05:55:42Z",
  "labels": [
    "refactor"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T05:55:42Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Concept

`multilineStart` rejects same-line triple-terminators via `bytes.Contains(s[3:], …)`. Edge cases worth pinning with tests:

- `s = """"""` (empty multi-line basic) — `s[3:] = """` contains `"""`, so single-line. Confirm.
- `s = """abc"""extra` — same: stays single-line. The trailing junk is then a problem for the single-line value parser; pin its error.
- Same matrix for `'''`.

No bug suspected — just no tests.
