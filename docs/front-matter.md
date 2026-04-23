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

## Not Supported

### Triple Dash Javascript Qualifier

```
---js
const var fruit = "apple";
---
```

### Triple SemiColor for JSON

As seen in [adrg/frontmatter](https://github.com/adrg/frontmatter) (Go):

```
;;;
{
	"fruit": "apple"
}
;;;
```

Easy to add if this is common.

