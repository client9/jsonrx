# String Escapes

Specs:

* [JSON RFC 7159](https://datatracker.ietf.org/doc/html/rfc7159#section-8)
* [Go rune literals](https://go.dev/ref/spec#Rune_literals)

| Char Escape  | Name            | Unicode | JSON | JS | Golang |
|--------------|-----------------|---------|---|---|---|
| \\0          | null            | U+0000  |   | X |   | 
| \\a          | bell            | U+0007  |   |   | X |
| \\b          | backspace       | U+0008  | X | X | X |
| \\f          | form feed       | U+000C  | X | X | X |
| \\n          | newline         | U+000A  | X | X | X |
| \\r          | carriage return | U+000D  | X |   | X |
| \\t          | tab             | U+0009  | X | X | X |
| \\v          | vertical tab    | U+000B  |   | X | X |
| \\\\         | backslash       | U+005C  | X | X | X |
| \\'          | single quote    | U+0027  |   | X |   |
| \\"          | double quote    | U+0022  | X | X | X |
| \\x?? hex    | hex escape      | U+00??  |   | X |   |
| \\u???? hex  | unicode escape  | U+????  | x | X |   |
| \\U????????  | unicode escape  |         |   | X |   |
| \\???        | octal escape    |         |   | X |   |
| \\[newline]  | end of line     | U+000A  |   | X |   |

