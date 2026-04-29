# Front Matter Guide

## YAML Formats

### Standard Triple Dash

```
---
fruit: "apple"
---
```

### Triple Dash YAML Qualifier

```
---yaml
fruit: "apple"
---
```

## TOML Formats

### Standard Triple Plus

```
+++
fruit = "apple"
+++
```

### Triple Dash TOML Qualifier

```
---toml
fruit = "apple"
---
```

## JSON Formats

### Object

Note: starting and ending`{` and `}` must be on a separate line.

```
{
    "friut": "apple"
}
```

### Triple Dash JSON Qualifier

```
---json
{
    "friut": "apple"
}
---
```

### Triple Dash Implicit

NOTE: this works since YAML is a superset of JSON.  It will be parsed with a YAML parser, but the result should be the same.

```
---
{
    "friut": "apple"
}
---
```

## Markdown Code Fence Formats

Using a triple-backtick code fence with an explicit language qualifier renders
as a syntax-highlighted block in Markdown and is recognised as front matter.
The qualifier is **required** — a bare ` ``` ` without a format is ignored.

### Code Fence YAML

```
```yaml
fruit: "apple"
```
```

### Code Fence TOML

```
```toml
fruit = "apple"
```
```

### Code Fence JSON

```
```json
{"fruit": "apple"}
```
```

## Not Supported

### Triple Dash Javascript Qualifier

```
---js
const var fruit = "apple";
---
```

### Triple SemiColon for JSON

As seen in [adrg/frontmatter](https://github.com/adrg/frontmatter) (Go):

```
;;;
{
	"fruit": "apple"
}
;;;
```

Easy to add if this is common.

