# tojson.FromJSONVariant

Convert alternative JSON formats back to spec so you can get on with your life.

## What does it do?

- [x] Remove trailing commas (JSON5, JWCC, HuJSON)
- [x] Remove leading commas ( cut-n-paste errors)
- [x] Convert hexadecimal literals to normal base10 integers (JSON5)
- [x] Ensure object key names are double quoted (JSON5)
- [x] Convert single-quoted strings to double quoted JSON.
- [x] Convert backtick string to double quoted JSON strings.
- [x] Convert multiline end-of-line escapes (backslash at end of line).
- [x] Remove single line '//' comments (C/Java/Go single line)(JSON5, JWCC, HuJSON)
- [x] Remove multiline '/* ... */' comments (JSON5, JWCC, HuJSON)
- [x] Remove shell-style # single line comments (SON, JSONX)
- [x] Remove '+' sign from integers (JSON5)
- [x] Remove leading zeros from numbers, e.g. "01"
- [x] Normalize floats ".5" to "0.5", "5." to "5"
- [x] Normalize string escape sequences
- [x] Convert "\r\n" to "\n"
- [x] Convert \x?? hex escapes
- [x] NaN and Infinity are errors (not representable in JSON)

## Supported JSON Variants

- [JSON](https://www.json.org/json-en.html) The original.
- [HuJSON](https://github.com/tailscale/hujson) (ending commas, C-style comments)
- [JWCC](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html) (ending commas, C-style comments)
- [JSON5](https://json5.org) (JSON as Javascript)
- [JSONC #2](https://code.visualstudio.com/docs/languages/json#_json-with-comments) (VS Code, ending commas, C-style coments)
- [SON](https://github.com/aleksandergurin/simple-object-notation) (ending commas, `#` comments)
- [HanSON](https://github.com/timjansen/hanson) (obsolete, unquoted keys, C-style comments, single or double quotes, ending commas)
- [JSONX](https://github.com/json-next) (similar to above)

## Additional Reading

* [Wikipedia](https://en.wikipedia.org/wiki/JSON)
* [Wuffs' Quirks Mode for JSON Parsing](https://github.com/google/wuffs/blob/3d6c609dc12de3c81e1b8079ceecf96370b086a2/std/json/decode_quirks.wuffs)
* [Awesome JSON](https://github.com/json-next/awesome-json-next)

