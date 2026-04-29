{
  "title": "frontmatter: Add support for triple backtick sentinel with qualifier",
  "id": "20260428T131108Z-5eb8f075",
  "state": "done",
  "created": "2026-04-28T13:11:08Z",
  "labels": [],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T13:11:08Z",
      "type": "filed",
      "to": "backlog"
    },
    {
      "ts": "2026-04-29T05:44:06Z",
      "type": "moved",
      "from": "backlog",
      "to": "active"
    },
    {
      "ts": "2026-04-29T05:46:02Z",
      "type": "moved",
      "from": "active",
      "to": "done"
    }
  ]
}

Another front matter marker is reusing the existing markdown code block with qualifier:

```
\`\`\`json  (or yaml or toml)
{
...
}
\`\`\`

```

It's useful since the metadata is encoded in a markdown code block and will render nicely.

## Design notes

- **Qualifier is required.** Unqualified ` ``` ` (no format) is NOT supported — it's too common in Markdown and would cause false positives. Only ` ```json `, ` ```yaml `, and ` ```toml ` are recognised.
- **Unknown backtick qualifiers** (e.g. ` ```js `) should return an error, matching the behavior of `---<unknown>` qualifiers.
- Closing sentinel is ` ``` ` (trimmed of trailing whitespace, same leniency as other sentinels).
- `docs/front-matter.md` needs to be updated.

## Resolution

Implemented as designed, with no deviations.

What landed:
- `frontmatter.go`: added ` ```yaml `, ` ```toml `, ` ```json ` entries to `frontMatterFormats`; extended unknown-qualifier check in `detectFrontMatterFormat` to also catch ` ```<unknown> ` with an error
- `frontmatter_test.go`: 11 new test cases covering yaml/toml/json basics, empty blocks, EOF-without-newline, trailing whitespace, unclosed, parse errors, unqualified backtick (no front matter), and bogus qualifiers
- `docs/front-matter.md`: new "Markdown Code Fence Formats" section documenting all three variants and the required-qualifier rule

Follow-ups: none
