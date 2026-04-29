{
  "title": "Add re-entry test matrix for openSection table/AoT ordering",
  "id": "20260428T055607Z-8fe97eec",
  "state": "backlog",
  "created": "2026-04-28T05:56:07Z",
  "labels": [
    "bug",
    "design"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T05:56:07Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Concept

`openSection` (toml_line.go:188) handles a subtle interaction between `tomlClosedTables.reopens`, the `cd == len(path)` implicit-vs-explicit check, and AoT next-element shortcut. Walking through the cases manually:

- `[a.b.c]` then `[a.b]` — implicit `b` becomes explicit. OK.
- `[a.b]` then `[a.b]` — duplicate-header error. OK.
- `[a.b]` then `[a]` — implicit `a` becomes explicit. OK.
- `[a]` then `[a.b]` then `[a]` — third `[a]` is duplicate-header. OK.
- `[[a.b]]` next-element — `},{`, clears `usedKeys`/`needComma`, leaves `explicit`. OK.
- `[[a.b]]` then `[[a.c]]` — closes b, opens c as new AoT. OK.

I don't see a hole, but no single test covers the full matrix and the logic is subtle enough that a refactor could regress quietly.

## Recommended phasing

1. Write an exhaustive test table over orderings of `[a]`, `[a.b]`, `[a.b.c]`, `[[a.b]]`, `[[a.c]]`.
2. For each ordering, assert: success, plain error, or `errReentry` (fallback to tree parser).
3. Use this as a regression net before any changes to `openSection` or `closed.reopens`.
