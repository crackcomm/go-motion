package motion

// Range represents an interval [Start, End) for operator application.
// Start ≤ End always holds. For forward motions, Start is the cursor
// and End is the motion destination. For backward motions, Start is
// the motion destination and End is the cursor. For linewise operations
// (dd, cc, yy), the range covers the entire line.
type Range struct {
	Start, End int
}

// Op represents a text operation that can be composed with a motion.
type Op int

const (
	OpNone   Op = iota // no operation (navigation only)
	OpDelete           // d — delete range
	OpChange           // c — delete range and enter insert mode
	OpYank             // y — yank range to register
)

// TextObjKind distinguishes inner ("i") from a ("a") text objects.
type TextObjKind int

const (
	TextObjInner TextObjKind = iota
	TextObjA
)

// TextObjDelim identifies a text object by the character used to
// select it after i/a.
type TextObjDelim int

const (
	TextObjWord        TextObjDelim = iota // iw / aw
	TextObjBigWord                         // iW / aW
	TextObjParen                           // i( / a(  or  i) / a)
	TextObjBrace                           // i{ / a{  or  i} / a}
	TextObjBracket                         // i[ / a[  or  i] / a]
	TextObjAngle                           // i< / a<
	TextObjDoubleQuote                     // i" / a"
	TextObjSingleQuote                     // i' / a'
	TextObjBacktick                        // i` / a`
	TextObjTag                             // it / at
	TextObjParagraph                       // ip / ap
)

// ResultKind indicates what action the engine produced.
type ResultKind int

const (
	ResultNone     ResultKind = iota // no-op (digit, unknown key, empty range)
	ResultNavigate                   // just move cursor to new position
	ResultExecute                    // execute operator over range
	ResultCancel                     // cancel pending operation
)

// Result is the output of the engine after processing a key.
// When Kind is ResultNavigate, Cursor holds the new position.
// When Kind is ResultExecute, Op, Range, and Insert are set.
type Result struct {
	Kind   ResultKind
	Op     Op
	Range  Range
	Cursor int
	Insert bool
}

// Key represents a single key press for the composition engine.
// It's a rune but we use a separate type for clarity and to distinguish
// from raw runes in the engine's API.
type Key rune

// Standard keys used in vim composition.
const KeyEsc Key = 27
