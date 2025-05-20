# jsonrx
Convert alternative JSON formats back to spec so you can get on with your life.

## WIP -- Not quite ready!

The public API of `Decode` and `DecodeAppend` are stable and works.

* However a lot more work needs to be done on error cases.  Row/Collum numbers on errors are likely wrong

* Conversion of string escape sequences in double/single/back needs work.

## What does it do?

- [x] Remove trailing commas (JSON5, JWCC, HuJSON)
- [x] Remove leading commas ( cut-n-paste errors)
- [x] Convert hexadecimal literals to normal base10 integers (JSON5)
- [x] Ensure object key names are double quoted (JSON5)
- [x] Convert single-quoted strings to double quoted JSON.
- [x] Convert backtick string to double quoted JSON strings.
- [x] Remove single line '//' comments (C/Java/Go single line)(JSON5, JWCC, HuJSON)
- [x] Remove multiline '/* ... */' comments (JSON5, JWCC, HuJSON)
- [x] Remove shell-style # single line comments
- [x] Remove '+' sign from integers (JSON5)
- [x] Remove leading zeros from numbers, e.g. "01"
- [x] Normalize floats ".5" to "0.5"
- [ ] Convert NaN forms  ... (coming soon)
- [ ] Convert Infinity forms to ... (coming soon)
- [ ] Normalize string escape sequences (coming soon)

## Supported JSON Variants

- [HuJSON](https://github.com/tailscale/hujson)
- [JWCC](https://nigeltao.github.io/blog/2021/json-with-commas-comments.html)
- [JSON5](https://json5.org)
- [JSONC #2](https://code.visualstudio.com/docs/languages/json#_json-with-comments)
- [SON](https://github.com/aleksandergurin/simple-object-notation)
* [HanSON](https://github.com/timjansen/hanson)
* [JSONX](https://github.com/json-next)

## Additional Reading

* [Wuffs' Quirks Mode for JSON Parsing](https://github.com/google/wuffs/blob/3d6c609dc12de3c81e1b8079ceecf96370b086a2/std/json/decode_quirks.wuffs)
* [Awesome JSON](https://github.com/json-next/awesome-json-next)

