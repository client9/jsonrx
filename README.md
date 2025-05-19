# jsonrx
Convert alternative JSON formats back to spec so you can get on with your life.

## WIP -- Not quite ready!

This works, but a lot more work needs to be done on error cases.

The public API (e.g. `Decode` and `DecodeAppend`) are stable.

## What does it do?

[x] Remove trailing commas (JSON5, JWCC, HuJSON)
[x] Remove leading commas ( cut-n-paste errors)
[x] Convert hexadecimal literals to normal base10 integers (JSON5)
[x] Ensure object key names are quoted (JSON5)
[x] Remove single line '//' comments (C/Java/Go single line)(JSON5, JWCC, HuJSON)
[x] Remove multiline '/* ... */' comments (JSON5, JWCC, HuJSON)
[x] Remove '+' sign from integers (JSON5)
[x] Remove leading zeros from numbers, e.g. "01"
[x] Normalize floats 
[ ] Convert NaN forms  ... (coming soon)
[ ] Convert Infinity forms to ... (coming soon)
[ ] Normalize string escape sequences
[x] Convert single-quoted strings to JSON.



