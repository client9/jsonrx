# jsonrx
Convert alternative JSON formats back to spec so you can get on with your life.

## WIP -- Not quite ready!

The public API (e.g. `Decode` and `DecodeAppend`) are stable and works.  However a lot more work needs to be done on error cases.


## What does it do?

- [x] Remove trailing commas (JSON5, JWCC, HuJSON)
- [x] Remove leading commas ( cut-n-paste errors)
- [x] Convert hexadecimal literals to normal base10 integers (JSON5)
- [x] Ensure object key names are quoted (JSON5)
- [x] Convert single-quoted strings to JSON.
- [x] Remove single line '//' comments (C/Java/Go single line)(JSON5, JWCC, HuJSON)
- [x] Remove multiline '/* ... */' comments (JSON5, JWCC, HuJSON)
- [x] Remove '+' sign from integers (JSON5)
- [x] Remove leading zeros from numbers, e.g. "01"
- [x] Normalize floats 
- [ ] Convert NaN forms  ... (coming soon)
- [ ] Convert Infinity forms to ... (coming soon)
- [ ] Normalize string escape sequences (coming soon)
- [ ] Backtick strings
- [ ] Shell-style # single line comments

## Supported JSON Variants

- [HuJSON](https://github.com/tailscale/hujson)
- [JWCC](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html)
- [JSON5](https://json5.org)
- [JSONC #2](https://code.visualstudio.com/docs/languages/json#_json-with-comments)

## Maybe Supported JSON Variants

[HanSON](https://github.com/timjansen/hanson)
- Need to add backtick quoting

[SON](https://github.com/aleksandergurin/simple-object-notation)
- Need shell style single line comments

[JSONX](https://github.com/json-next)
- Need shell style single line comments
- Need backtick quoting

## Unsupported JSON Variants

- Yaml
- [HOCON](https://github.com/lightbend/config/blob/master/HOCON.md)
