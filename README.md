# go-motion

A Vim motion library for Go. Pure functions on `[]rune` + cursor position — faithful to neovim behavior.

## Usage

```go
import "github.com/crackcomm/go-motion"

// Use individual motions directly:
pos := motion.MoveParagraphForward(text, cursor, 1)
pos := motion.NextWordStart(text, cursor, 2)
rng := motion.WordInner(text, cursor)

// Or use the Engine for key sequence composition:
var eng motion.Engine
for _, key := range "d2w" {
    result := eng.Process(text, cursor, motion.Key(key))
}
// result.Kind == motion.ResultExecute
// result.Op == motion.OpDelete
// result.Range covers the next 2 words
```

## Package Structure

| File                  | Contents                                                                       |
| --------------------- | ------------------------------------------------------------------------------ |
| `types.go`            | Core types: `Range`, `Op`, `Result`, `Key`, `TextObjKind`, `TextObjDelim`      |
| `classify.go`         | Character classification (Keyword/NonKeyword/Space/Emoji)                      |
| `motion_word.go`      | Word motions: `w`, `b`, `e`, `ge` and bigword `W`, `B`, `E`, `gE`              |
| `motion_line.go`      | Line motions: `0`, `$`, `^`, `g_`                                              |
| `motion_search.go`    | Char search: `f`, `F`, `t`, `T`                                                |
| `motion_bracket.go`   | Bracket matching `%`                                                           |
| `motion_paragraph.go` | Paragraph motions `{`, `}`                                                     |
| `motion_buffer.go`    | Buffer motions: `gg`, `G`                                                      |
| `textobj.go`          | Text objects: `iw`/`aw`, `iW`/`aW`, `i"`/`a"`, `i(`/`a(`, `it`/`at`, `ip`/`ap` |
| `operators.go`        | Operators: `OpDelete`, `OpChange`, `OpYank` + `ApplyOp`                        |
| `engine.go`           | Composition state machine                                                      |

## Motion Functions

All motions follow: `func(text []rune, pos int, count ...int) int`

| Function                | Key             | Description                                  |
| ----------------------- | --------------- | -------------------------------------------- |
| `NextWordStart`         | `w`             | Forward to start of next word                |
| `PrevWordStart`         | `b`             | Backward to start of word                    |
| `NextWordEnd`           | `e`             | Forward to end of word                       |
| `PrevWordEnd`           | `ge`            | Backward to end of previous word             |
| `NextBigwordStart`      | `W`             | Forward to next bigword                      |
| `PrevBigwordStart`      | `B`             | Backward to bigword start                    |
| `NextBigwordEnd`        | `E`             | Forward to bigword end                       |
| `PrevBigwordEnd`        | `gE`            | Backward to bigword end                      |
| `LineStart`             | `0`             | First column of current line                 |
| `LineEnd`               | `$`             | Last column of current line                  |
| `FirstNonBlank`         | `^`             | First non-whitespace of current line         |
| `LastNonBlank`          | `g_`            | Last non-whitespace of current line          |
| `FindNext`              | `f`             | Next char forward on current line            |
| `FindPrev`              | `F`             | Previous char backward on current line       |
| `TillNext`              | `t`             | Before next char forward on current line     |
| `TillPrev`              | `T`             | After previous char backward on current line |
| `MatchBracket`          | `%`             | Matching bracket under cursor                |
| `MoveParagraphForward`  | `}`             | Forward to next paragraph                    |
| `MoveParagraphBackward` | `{`             | Backward to paragraph start                  |
| `MoveToLine`            | `gg`/`{count}G` | Go to line (1-indexed)                       |
| `MoveToEndOfBuffer`     | `G`             | Go to end of buffer                          |

## Text Objects

Text object functions return `Range{start, end}` (exclusive end).

| Function         | Keys                   | Description                                 |
| ---------------- | ---------------------- | ------------------------------------------- |
| `WordInner`      | `iw`                   | Inner word                                  |
| `WordA`          | `aw`                   | A word (includes trailing whitespace)       |
| `BigWordInner`   | `iW`                   | Inner bigword                               |
| `BigWordA`       | `aW`                   | A bigword                                   |
| `QuoteInner`     | `i"`, `i'`, `` i` ``   | Inside quotes                               |
| `QuoteA`         | `a"`, `a'`, `` a` ``   | Including quotes                            |
| `PairInner`      | `i(`, `i[`, `i{`, `i<` | Inside pair                                 |
| `PairA`          | `a(`, `a[`, `a{`, `a<` | Including delimiters                        |
| `TagInner`       | `it`                   | Inside HTML/XML tag                         |
| `TagA`           | `at`                   | Including tag                               |
| `ParagraphInner` | `ip`                   | Current paragraph (no blank lines)          |
| `ParagraphA`     | `ap`                   | Current paragraph (with surrounding blanks) |

## Operators

```go
func ApplyOp(op Op, text []rune, cursor int, r Range) ApplyResult
```

| Op         | Behavior                   | Cursor After   |
| ---------- | -------------------------- | -------------- |
| `OpDelete` | Remove range from text     | At range start |
| `OpChange` | Remove range, enter insert | At range start |
| `OpYank`   | Copy range to Yanked field | Unchanged      |

`ApplyResult` fields: `Text []rune`, `Cursor int`, `Insert bool`, `Yanked []rune`. The caller is responsible for managing mode and yank register.

## Engine

The `Engine` type manages the Vim command composition state machine:

- **Idle** — waiting for a command
- **Operator pending** — received `d`, `c`, or `y`, waiting for motion/textobj
- **Text object pending** — received `i` or `a`, waiting for delimiter
- **Char search pending** — received `f`, `F`, `t`, `T`, waiting for target char
- **G prefix** — received `g`, waiting for next key

### States

| Input           | Idle                     | Op Pending                   | G Pending             | TextObj Pending  | Char Pending     |
| --------------- | ------------------------ | ---------------------------- | --------------------- | ---------------- | ---------------- |
| `d`,`c`,`y`     | → Op                     | execute double‑op (dd/cc/yy) | → mark                | → mark           | → mark           |
| `i`,`a`         | pass‑through             | → TextObj                    | → TextObj             | —                | —                |
| `G`             | Navigate/Execute         | Execute                      | Navigate/Execute      | —                | —                |
| `f`,`F`,`t`,`T` | → Char                   | → Char                       | → Char                | —                | —                |
| `g`             | → G                      | → G                          | —                     | —                | —                |
| `gg`            | execute gg               | execute gg                   | —                     | —                | —                |
| motion key      | Navigate                 | Execute(op+motion)           | Execute(op+ge/g\_/gE) | —                | —                |
| text obj delim  | —                        | Execute(op+textobj)          | —                     | Navigate/Execute | —                |
| char            | Navigate                 | Execute(op+char)             | —                     | —                | Navigate/Execute |
| `1-9`           | start count              | append count                 | —                     | —                | —                |
| `0`             | LineStart / append count | append count                 | —                     | —                | —                |
| Esc             | Cancel                   | Cancel                       | Cancel                | Cancel           | Cancel           |

### Pending State

```go
eng.Process(text, cursor, motion.Key('d')) // starts an operator
if eng.Pending() {
    // Engine is waiting for a motion or text object.
    // Route subsequent keys to the engine.
}
```

Applications should check `Pending()` before routing a key to the engine — if the engine is idle, the key might be an insert/navigation command that the application should handle directly.

For status-line display (like neovim's `showcmd`), call `Status()`:

```go
eng.Process(text, cursor, motion.Key('d'))
eng.Process(text, cursor, motion.Key('4'))
fmt.Print(eng.Status()) // prints "d4"
```

### Repeat (;, ,)

`;` repeats the last `f`/`F`/`t`/`T` in the same direction. `,` repeats in the opposite direction.

## Character Classification

| Class | Constant     | Examples                              |
| ----- | ------------ | ------------------------------------- |
| 0     | `Space`      | space, tab, newline                   |
| 1     | `NonKeyword` | `.,!?()[]{}` etc.                     |
| 2     | `Keyword`    | letters, digits, `_`, Unicode letters |
| 3     | `Emoji`      | Emoji codepoints                      |

## Unimplemented / Partial

These are tracked as known limitations. Contributions welcome.

| Feature                   | Expected Keys                               | Status                       |
| ------------------------- | ------------------------------------------- | ---------------------------- |
| Text object count         | `d2iw`, `c3aw`                              | Not implemented              |
| Case change operators     | `gu{motion}`, `gU{motion}`                  | Cancel instead (placeholder) |
| Repeat last change        | `.`                                         | Not implemented              |
| Join lines                | `J`                                         | Not implemented              |
| Shift / indent            | `>`, `<`                                    | Not implemented              |
| Format                    | `=`                                         | Not implemented              |
| Filter                    | `!`                                         | Not implemented              |
| Replace                   | `r`, `R`                                    | Not implemented              |
| Single-char delete/change | `x`, `X`, `s`, `S`                          | Not implemented              |
| Search                    | `/`, `?`, `n`, `N`, `*`, `#`                | Not implemented              |
| Screen-position motions   | `H`, `M`, `L`                               | Not implemented              |
| Scroll                    | `Ctrl-D`, `Ctrl-U`, `Ctrl-F`, `Ctrl-B`, `z` | Not implemented              |
| Sentence motions          | `(`, `)`                                    | Not implemented              |
| Visual mode               | `v`, `V`, `Ctrl-V`                          | Not implemented              |
| Marks                     | `m`, `'`, `` ` ``                           | Not implemented              |
| Registers                 | `"`, `0-9`, `a-z`                           | Not implemented              |
| Macros                    | `q`, `@`                                    | Not implemented              |

## Design Notes

- All functions operate on `[]rune` for correct multi-byte handling.
- No global state. The `Engine` is the only stateful type and is safe for single‑threaded use.
- Line motions on `\n`: `LineStart` skips to the next line, `LineEnd` stays on the previous line.
- Paragraphs are separated by blank lines, matching neovim's `findpar()`.
- Bracket matching `%` supports `()`, `[]`, `{}` with nesting tracking.
- `g`-prefixed motions (`ge`, `gE`, `g_`) compose with operators in both Idle and G-pending states.
- `G` in Idle and Op-pending states navigates/executes directly (no `g` prefix needed).
