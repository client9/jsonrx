{
  "title": "Guard the cap-difference offset trick in startMultilineValue",
  "id": "20260428T055634Z-ec27df98",
  "state": "backlog",
  "created": "2026-04-28T05:56:34Z",
  "labels": [
    "bug",
    "refactor"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T05:56:34Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Concept

`startMultilineValue` recovers `rest`'s offset within `p.input` via `cap(p.input) - cap(rest)`. That works only if every slice operation between `p.input` and `rest` was 2-arg slicing (cap preserved). The doc comment says so.

Audit of the current path confirms the invariant holds:

- `input[pos:pos+nl]` — 2-arg
- `bytes.TrimRight(line, " \t\r")` — `s[:i]`, 2-arg
- `stripInlineComment` — returns `bytes.TrimRight(s[:i], …)` or `s` itself, 2-arg
- `bytes.TrimSpace`, `bytes.TrimLeft` — 2-arg
- `trimmed[eqPos+1:]` — 2-arg

If anyone introduces a 3-arg slice or a `make`+`copy` in this path, `accumStart` will silently point to the wrong byte and multi-line values will decode garbage.

## Suggested mitigation

Either:

1. Add a debug-only sanity check in `startMultilineValue`: `if p.accumStart < 0 || p.accumStart > len(p.input) { panic(...) }`. Cheap and catches any future regression.
2. Add `// MUST: 2-arg slicing only beyond this point` comments at each boundary above.

Option 1 alone is probably sufficient.
