{
  "title": "Deduplicate bare-key emission path in convert",
  "id": "20260428T055654Z-37192ab0",
  "state": "backlog",
  "created": "2026-04-28T05:56:54Z",
  "labels": [
    "refactor"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T05:56:54Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Concept

`convert` (toml_line.go:576-600) reimplements what `handleDottedKeyValue` does for the single-key case: `markKey`, comma, write key, `:`, `writeValue`. This is faster (skips `parseTOMLKeyPath`) but duplicates emission logic — if the dotted path adds a new validation or changes how it emits, the bare path won't get it.

## Recommended fix

Extract a shared `emitKeyValue(key []byte, rest []byte, lineNum, valCol int)` once correctness tests are in place. The bare-key path calls it directly with the parsed key; the dotted path calls it after opening prefix frames.

## Anti-goals

Don't merge to the point of losing the bare-key fast path's allocation/parse savings — keep the `tomlBareKeyValue` shortcut, just share the emission tail.
