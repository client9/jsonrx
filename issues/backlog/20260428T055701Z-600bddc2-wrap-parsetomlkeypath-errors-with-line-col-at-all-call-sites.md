{
  "title": "Wrap parseTOMLKeyPath errors with line/col at all call sites",
  "id": "20260428T055701Z-600bddc2",
  "state": "backlog",
  "created": "2026-04-28T05:57:01Z",
  "labels": [
    "bug"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T05:57:01Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Symptom

`parseTOMLKeyPath` returns plain errors. The line-parser callers (`handleHeader`, `handleDottedKeyValue` in toml_line.go) wrap with `atLineCol`. Other call sites — notably `parseTOMLInlineTable` in toml_scalar.go — don't, so errors from key parsing inside an inline table surface without source position.

## Suspected fix

Audit all `parseTOMLKeyPath` call sites; ensure every one wraps the error with `atLineCol` (or the column-relative variant the caller has access to).

## Tests to add

- Inline table with a malformed key: `t = { "unterminated = 1 }` — error should carry a line number.
