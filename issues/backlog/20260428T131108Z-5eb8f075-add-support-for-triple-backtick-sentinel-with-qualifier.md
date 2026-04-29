{
  "title": "Add support for triple backtick sentinel with qualifier",
  "id": "20260428T131108Z-5eb8f075",
  "state": "backlog",
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

It's usefull since the meta data is encoded in markdown code block and will render nicely.

The @docs/front-matter.md needs to be updated.

