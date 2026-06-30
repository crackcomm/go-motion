package motion

// record appends a key to the pending status buffer.
func (e *Engine) record(r rune) {
	e.pending = append(e.pending, r)
}

//
// Composition engine
//
// The engine is a state machine that processes vim normal-mode key
// sequences and produces results. It handles:
//
//   - Simple motions: w, b, e, ge, 0, $, ^, g_, {, }, gg, G, %, f, t, F, T
//   - Operators: d, c, y — composed with motions and text objects
//   - Double-key operators: dd, cc, yy (linewise)
//   - Text objects: iw, aw, iW, aW, i", a", i(, a(, i[, a[, i{, a{, i<, a<,
//                   it, at, ip, ap
//   - Counts: 3w, 5dw, 2di"
//   - G-prefixed: gg, g_, gE, ge
//   - Char search: f, t, F, T, ;, ,
//   - Esc to cancel pending operations
//
// The engine evaluates motions against the text buffer immediately,
// so Process() always returns a complete result or ResultNone.
//
// Usage:
//
//	var e motion.Engine
//	for each keypress {
//	    result := e.Process(text, cursor, key)
//	    switch result.Kind {
//	    case motion.ResultNavigate:
//	        cursor = result.Cursor
//	    case motion.ResultExecute:
//	        ar := motion.ApplyOp(result.Op, text, cursor, result.Range)
//	        text, cursor = ar.Text, ar.Cursor
//	        if result.Insert { enter insert mode }
//	    case motion.ResultCancel:
//	        // clear pending state
//	    }
//	}
//

// motionType identifies a specific vim motion for the engine to resolve.
type motionType int

const (
	motionH       motionType = iota // h — left
	motionJ                         // j — down
	motionK                         // k — up
	motionL                         // l — right
	motionW                         // w — next word start
	motionB                         // b — prev word start
	motionE                         // e — next word end
	motionGe                        // ge — prev word end
	motionBigW                      // W — next WORD start
	motionBigB                      // B — prev WORD start
	motionBigE                      // E — next WORD end
	motionBigGe                     // gE — prev WORD end
	motionZero                      // 0 — line start
	motionDollar                    // $ — line end
	motionCaret                     // ^ — first non-blank
	motionG_                        // g_ — last non-blank
	motionOBrace                    // { — paragraph backward
	motionCBrace                    // } — paragraph forward
	motionGG                        // gg — line 1
	motionG                         // G — last line or line N
	motionPercent                   // % — matching bracket
	motionF                         // f — find char forward
	motionT                         // t — till char forward
	motionBwdF                      // F — find char backward
	motionBwdT                      // T — till char backward
)

// engineState tracks the current position in a key sequence.
type engineState int

const (
	stIdle    engineState = iota // waiting for first key
	stOp                         // operator (d/c/y) pending, waiting for motion/textobj
	stG                          // g prefix pending
	stTextObj                    // i or a pressed after operator, waiting for delimiter
	stChar                       // f/t/F/T pressed, waiting for target character
)

// Engine is a vim key sequence composition state machine.
//
// It is stateful across calls: call Process() for each keypress.
// The zero value is ready to use.
type Engine struct {
	state engineState
	op    Op
	count int
	kind  TextObjKind // inner or a (for text object)

	// Char search repeat state.
	lastChar rune
	lastDir  int // 1 or -1
	lastTill bool

	// Pending key buffer for Status() display.
	pending []rune
}

// Pending returns true if the engine is in the middle of a key sequence
// (operator pending, g-prefix, text object, or char search).
// Applications can check this to decide whether to route the next key
// to the engine or handle it directly.
func (e *Engine) Pending() bool {
	return e.state != stIdle
}

// Status returns the pending key sequence as typed, for status-line display
// (like neovim's showcmd). Returns empty string when idle.
// Buffer is reused across calls — result is invalidated on next Process call.
func (e *Engine) Status() string {
	return string(e.pending)
}

// Reset clears all pending state, returning the engine to idle.
// Call this when the application handles a non-motion key (like i, a, /).
func (e *Engine) Reset() {
	e.state = stIdle
	e.op = OpNone
	e.count = 0
	e.kind = TextObjInner
	e.pending = e.pending[:0]
}

// Process handles a single keypress and returns the result.
//
// The text buffer and cursor position are needed so the engine can
// evaluate motion ranges immediately. If the sequence is incomplete,
// Result.Kind is ResultNone.
//
// If the engine doesn't understand the key (e.g., 'i' for insert mode),
// it returns ResultNone with no state change. Callers should ignore
// the result and handle the key themselves, or call Reset() first.
//
// Consecutive calls without intermediate Reset() form a single key
// sequence (e.g., "d" + "i" + "\"").
func (e *Engine) Process(text []rune, cursor int, key Key) Result {
	r := rune(key)

	// Handle escape: always cancel.
	if key == KeyEsc {
		return e.cancel()
	}

	// Count digits: 1-9 start a count, 0 continues one.
	if r >= '0' && r <= '9' {
		if r == '0' && e.count == 0 && e.state != stOp && e.state != stG {
			// 0 at the start (no count yet) is a motion, not a digit.
			// But after an operator, 0 is a digit for d0, c0, y0.
			// We handle it below as motionZero.
		} else {
			if e.count > 0 || r != '0' {
				e.count = e.count*10 + int(r-'0')
				e.record(r)
				return Result{Kind: ResultNone}
			}
		}
	}

	cnt := e.count
	if cnt == 0 {
		cnt = 1
	}

	switch e.state {
	case stIdle:
		return e.handleIdle(text, cursor, r, cnt)

	case stOp:
		return e.handleOp(text, cursor, r, cnt)

	case stG:
		return e.handleG(text, cursor, r, cnt)

	case stTextObj:
		return e.handleTextObj(text, cursor, r)

	case stChar:
		return e.handleChar(text, cursor, r, cnt)
	}

	return Result{Kind: ResultNone}
}

// cancel clears state and returns a Cancel result.
func (e *Engine) cancel() Result {
	e.Reset()
	return Result{Kind: ResultCancel}
}

// ---------------------------------------------------------------------------
// State: idle — waiting for the first key of a sequence
// ---------------------------------------------------------------------------

func (e *Engine) handleIdle(text []rune, cursor int, r rune, cnt int) Result {
	switch r {
	case 'h':
		return e.navigateMotion(text, cursor, motionH, cnt)
	case 'j':
		return e.navigateMotion(text, cursor, motionJ, cnt)
	case 'k':
		return e.navigateMotion(text, cursor, motionK, cnt)
	case 'l':
		return e.navigateMotion(text, cursor, motionL, cnt)

	case 'w':
		return e.navigateMotion(text, cursor, motionW, cnt)
	case 'b':
		return e.navigateMotion(text, cursor, motionB, cnt)
	case 'e':
		return e.navigateMotion(text, cursor, motionE, cnt)
	case 'W':
		return e.navigateMotion(text, cursor, motionBigW, cnt)
	case 'B':
		return e.navigateMotion(text, cursor, motionBigB, cnt)
	case 'E':
		return e.navigateMotion(text, cursor, motionBigE, cnt)

	case '0':
		return e.navigateMotion(text, cursor, motionZero, 0)
	case '$':
		return e.navigateMotion(text, cursor, motionDollar, cnt)
	case '^':
		return e.navigateMotion(text, cursor, motionCaret, cnt)
	case '{':
		return e.navigateMotion(text, cursor, motionOBrace, cnt)
	case '}':
		return e.navigateMotion(text, cursor, motionCBrace, cnt)
	case '%':
		return e.navigateMotion(text, cursor, motionPercent, cnt)

	case 'f':
		e.record('f')
		e.state = stChar
		e.lastDir = 1
		e.lastTill = false
		return Result{Kind: ResultNone}
	case 't':
		e.record('t')
		e.state = stChar
		e.lastDir = 1
		e.lastTill = true
		return Result{Kind: ResultNone}
	case 'F':
		e.record('F')
		e.state = stChar
		e.lastDir = -1
		e.lastTill = false
		return Result{Kind: ResultNone}
	case 'T':
		e.record('T')
		e.state = stChar
		e.lastDir = -1
		e.lastTill = true
		return Result{Kind: ResultNone}

	case ';':
		if e.lastChar == 0 {
			return Result{Kind: ResultNone}
		}
		return e.navigateChar(text, cursor, e.lastChar, e.lastDir, e.lastTill, cnt)

	case ',':
		if e.lastChar == 0 {
			return Result{Kind: ResultNone}
		}
		return e.navigateChar(text, cursor, e.lastChar, -e.lastDir, e.lastTill, cnt)

		// Operators.
	case 'd':
		e.record('d')
		e.state = stOp
		e.op = OpDelete
		return Result{Kind: ResultNone}
	case 'c':
		e.record('c')
		e.state = stOp
		e.op = OpChange
		return Result{Kind: ResultNone}
	case 'y':
		e.record('y')
		e.state = stOp
		e.op = OpYank
		return Result{Kind: ResultNone}

	case 'G':
		// G without count: go to end of buffer.
		// With count: go to that line number.
		if e.count == 0 {
			return e.navigateMotion(text, cursor, motionG, 0)
		}
		return e.navigateMotion(text, cursor, motionGG, e.count)

	// G prefix.
	case 'g':
		e.record('g')
		e.state = stG
		return Result{Kind: ResultNone}
	}

	return Result{Kind: ResultNone}
}

// ---------------------------------------------------------------------------
// State: operator pending — waiting for motion or text object
// ---------------------------------------------------------------------------

func (e *Engine) handleOp(text []rune, cursor int, r rune, cnt int) Result {
	switch {
	// Double-key: dd, cc, yy — linewise operation.
	case r == 'd' && e.op == OpDelete:
		return e.lineOp(text, cursor, OpDelete)

	case r == 'c' && e.op == OpChange:
		return e.lineOp(text, cursor, OpChange)

	case r == 'y' && e.op == OpYank:
		return e.lineOp(text, cursor, OpYank)

		// Text objects: i/a after operator.
	case r == 'i':
		e.record('i')
		e.state = stTextObj
		e.kind = TextObjInner
		return Result{Kind: ResultNone}
	case r == 'a':
		e.record('a')
		e.state = stTextObj
		e.kind = TextObjA
		return Result{Kind: ResultNone}

		// Motions.
	case r == 'h':
		return e.execute(text, cursor, motionH, cnt)
	case r == 'j':
		return e.execute(text, cursor, motionJ, cnt)
	case r == 'k':
		return e.execute(text, cursor, motionK, cnt)
	case r == 'l':
		return e.execute(text, cursor, motionL, cnt)
	case r == 'w':
		return e.execute(text, cursor, motionW, cnt)
	case r == 'b':
		return e.execute(text, cursor, motionB, cnt)
	case r == 'e':
		return e.execute(text, cursor, motionE, cnt)
	case r == 'W':
		return e.execute(text, cursor, motionBigW, cnt)
	case r == 'B':
		return e.execute(text, cursor, motionBigB, cnt)
	case r == 'E':
		return e.execute(text, cursor, motionBigE, cnt)

	case r == '0':
		return e.execute(text, cursor, motionZero, 0)
	case r == '$':
		return e.execute(text, cursor, motionDollar, cnt)
	case r == '^':
		return e.execute(text, cursor, motionCaret, cnt)
	case r == '{':
		return e.execute(text, cursor, motionOBrace, cnt)
	case r == '}':
		return e.execute(text, cursor, motionCBrace, cnt)
	case r == '%':
		return e.execute(text, cursor, motionPercent, cnt)

	case r == 'G':
		if e.count == 0 {
			return e.execute(text, cursor, motionG, 0)
		}
		return e.execute(text, cursor, motionGG, e.count)

		// Char search with operator.
	case r == 'f':
		e.record('f')
		e.state = stChar
		e.lastDir = 1
		e.lastTill = false
		return Result{Kind: ResultNone}
	case r == 't':
		e.record('t')
		e.state = stChar
		e.lastDir = 1
		e.lastTill = true
		return Result{Kind: ResultNone}
	case r == 'F':
		e.record('F')
		e.state = stChar
		e.lastDir = -1
		e.lastTill = false
		return Result{Kind: ResultNone}
	case r == 'T':
		e.record('T')
		e.state = stChar
		e.lastDir = -1
		e.lastTill = true
		return Result{Kind: ResultNone}

	case r == ';':
		if e.lastChar == 0 {
			return Result{Kind: ResultNone}
		}
		return e.executeChar(text, cursor, e.lastChar, e.lastDir, e.lastTill, cnt)

	case r == ',':
		if e.lastChar == 0 {
			return Result{Kind: ResultNone}
		}
		return e.executeChar(text, cursor, e.lastChar, -e.lastDir, e.lastTill, cnt)

	case r == 'g':
		e.record('g')
		e.state = stG
		return Result{Kind: ResultNone}
	}

	return e.cancel()
}

// lineOp executes a linewise operator (dd, cc, yy).
func (e *Engine) lineOp(text []rune, cursor int, op Op) Result {
	start := LineStart(text, cursor)
	end := LineEnd(text, cursor)
	if end < npos(text)-1 {
		end++ // include the \n
	}
	if end < npos(text) {
		end++ // include the \n terminator
	}
	e.Reset()
	return Result{
		Kind:   ResultExecute,
		Op:     op,
		Range:  Range{start, end},
		Insert: op == OpChange,
	}
}

// ---------------------------------------------------------------------------
// State: g prefix — waiting for the second key of a g-sequence
// ---------------------------------------------------------------------------

func (e *Engine) handleG(text []rune, cursor int, r rune, cnt int) Result {
	switch r {
	case 'g':
		// gg: go to line N (or line 1 if no count).
		line := e.count
		if line == 0 {
			line = 1
		}
		pos := MoveToLine(text, line)
		e.Reset()
		return Result{Kind: ResultNavigate, Cursor: pos}

	case 'G':
		// G with explicit count: go to line N.
		// G without count: go to last line.
		var pos int
		if e.count == 0 {
			pos = MoveToEndOfBuffer(text)
		} else {
			pos = MoveToLine(text, e.count)
		}
		e.Reset()
		return Result{Kind: ResultNavigate, Cursor: pos}

	case '_':
		if e.op != OpNone {
			return e.execute(text, cursor, motionG_, cnt)
		}
		return e.navigateMotion(text, cursor, motionG_, cnt)
	case 'E':
		if e.op != OpNone {
			return e.execute(text, cursor, motionBigGe, cnt)
		}
		return e.navigateMotion(text, cursor, motionBigGe, cnt)
	case 'e':
		if e.op != OpNone {
			return e.execute(text, cursor, motionGe, cnt)
		}
		return e.navigateMotion(text, cursor, motionGe, cnt)

		// g-prefixed operators: gu / gU.
		// Not yet implemented — return the operator to the caller
		// so it can handle it, rather than silently doing the wrong thing.
	case 'u':
	case 'U':
		return e.cancel()
	}

	return e.cancel()
}

// ---------------------------------------------------------------------------
// State: text object — waiting for delimiter after i/a
// ---------------------------------------------------------------------------

var textObjDelims = map[rune]TextObjDelim{
	'w':  TextObjWord,
	'W':  TextObjBigWord,
	'p':  TextObjParagraph,
	'"':  TextObjDoubleQuote,
	'\'': TextObjSingleQuote,
	'`':  TextObjBacktick,
	'(':  TextObjParen,
	')':  TextObjParen,
	'[':  TextObjBracket,
	']':  TextObjBracket,
	'{':  TextObjBrace,
	'}':  TextObjBrace,
	'<':  TextObjAngle,
	'>':  TextObjAngle,
	't':  TextObjTag,
	'b':  TextObjParen, // ib = inner paren block
	'B':  TextObjBrace, // iB = inner brace block
}

func (e *Engine) handleTextObj(text []rune, cursor int, r rune) Result {
	delim, ok := textObjDelims[r]
	if !ok {
		return e.cancel()
	}

	rng := resolveTextObjRange(text, cursor, e.kind, delim)
	op := e.op
	e.Reset()
	if rng.Start == rng.End {
		// Empty range — no-op.
		return Result{Kind: ResultNone}
	}
	return Result{
		Kind:   ResultExecute,
		Op:     op,
		Range:  rng,
		Insert: op == OpChange,
	}
}

// resolveTextObjRange computes the Range for a text object.
func resolveTextObjRange(text []rune, cursor int, kind TextObjKind, delim TextObjDelim) Range {
	switch delim {
	case TextObjWord:
		if kind == TextObjInner {
			return WordInner(text, cursor)
		}
		return WordA(text, cursor)

	case TextObjBigWord:
		if kind == TextObjInner {
			return BigWordInner(text, cursor)
		}
		return BigWordA(text, cursor)

	case TextObjParagraph:
		if kind == TextObjInner {
			return ParagraphInner(text, cursor)
		}
		return ParagraphA(text, cursor)

	case TextObjDoubleQuote:
		if kind == TextObjInner {
			return QuoteInner(text, cursor, '"')
		}
		return QuoteA(text, cursor, '"')

	case TextObjSingleQuote:
		if kind == TextObjInner {
			return QuoteInner(text, cursor, '\'')
		}
		return QuoteA(text, cursor, '\'')

	case TextObjBacktick:
		if kind == TextObjInner {
			return QuoteInner(text, cursor, '`')
		}
		return QuoteA(text, cursor, '`')

	case TextObjParen:
		if kind == TextObjInner {
			return PairInner(text, cursor, '(', ')')
		}
		return PairA(text, cursor, '(', ')')

	case TextObjBrace:
		if kind == TextObjInner {
			return PairInner(text, cursor, '{', '}')
		}
		return PairA(text, cursor, '{', '}')

	case TextObjBracket:
		if kind == TextObjInner {
			return PairInner(text, cursor, '[', ']')
		}
		return PairA(text, cursor, '[', ']')

	case TextObjAngle:
		if kind == TextObjInner {
			return PairInner(text, cursor, '<', '>')
		}
		return PairA(text, cursor, '<', '>')

	case TextObjTag:
		if kind == TextObjInner {
			return TagInner(text, cursor)
		}
		return TagA(text, cursor)
	}

	return Range{cursor, cursor}
}

// ---------------------------------------------------------------------------
// State: char search — waiting for target character
// ---------------------------------------------------------------------------

func (e *Engine) handleChar(text []rune, cursor int, r rune, cnt int) Result {
	ch := r
	e.lastChar = ch

	hasOp := e.op != OpNone

	if hasOp {
		op := e.op
		e.op = OpNone
		rng := resolveCharRange(text, cursor, ch, e.lastDir, e.lastTill, cnt)
		e.Reset()
		return Result{
			Kind:   ResultExecute,
			Op:     op,
			Range:  rng,
			Insert: op == OpChange,
		}
	}

	e.state = stIdle
	return e.navigateChar(text, cursor, ch, e.lastDir, e.lastTill, cnt)
}

// ---------------------------------------------------------------------------
// Navigation result constructors
// ---------------------------------------------------------------------------

func (e *Engine) navigateMotion(text []rune, cursor int, mt motionType, cnt int) Result {
	pos := resolveMotionPos(text, cursor, mt, cnt)
	e.Reset()
	return Result{
		Kind:   ResultNavigate,
		Cursor: pos,
	}
}

func (e *Engine) navigateChar(text []rune, cursor int, ch rune, dir int, till bool, count int) Result {
	pos := resolveCharPos(text, cursor, ch, dir, till, count)
	e.Reset()
	if pos < 0 {
		return Result{Kind: ResultNone}
	}
	return Result{
		Kind:   ResultNavigate,
		Cursor: pos,
	}
}

// ---------------------------------------------------------------------------
// Execute result constructors
// ---------------------------------------------------------------------------

func (e *Engine) execute(text []rune, cursor int, mt motionType, count int) Result {
	rng := resolveMotionRange(text, cursor, mt, count)
	op := e.op
	e.Reset()
	if rng.Start == rng.End {
		return Result{Kind: ResultNone}
	}
	return Result{
		Kind:   ResultExecute,
		Op:     op,
		Range:  rng,
		Insert: op == OpChange,
	}
}

func (e *Engine) executeChar(text []rune, cursor int, ch rune, dir int, till bool, count int) Result {
	rng := resolveCharRange(text, cursor, ch, dir, till, count)
	op := e.op
	e.Reset()
	if rng.Start == rng.End {
		return Result{Kind: ResultNone}
	}
	return Result{
		Kind:   ResultExecute,
		Op:     op,
		Range:  rng,
		Insert: op == OpChange,
	}
}

// ---------------------------------------------------------------------------
// Motion position resolution (for navigation)
// ---------------------------------------------------------------------------

func resolveMotionPos(text []rune, cursor int, mt motionType, count int) int {
	switch mt {
	case motionH:
		pos := max(cursor-count, 0)
		return pos
	case motionL:
		pos := cursor + count
		n := len(text)
		if pos >= n {
			pos = n - 1
		}
		return pos
	case motionJ:
		return moveVertical(text, cursor, count)
	case motionK:
		return moveVertical(text, cursor, -count)
	case motionW:
		return NextWordStart(text, cursor, count)
	case motionB:
		return PrevWordStart(text, cursor, count)
	case motionE:
		return NextWordEnd(text, cursor, count)
	case motionGe:
		return PrevWordEnd(text, cursor, count)
	case motionBigW:
		return NextBigwordStart(text, cursor, count)
	case motionBigB:
		return PrevBigwordStart(text, cursor, count)
	case motionBigE:
		return NextBigwordEnd(text, cursor, count)
	case motionBigGe:
		return PrevBigwordEnd(text, cursor, count)
	case motionZero:
		return LineStart(text, cursor)
	case motionDollar:
		return LineEnd(text, cursor)
	case motionCaret:
		return FirstNonBlank(text, cursor)
	case motionG_:
		return LastNonBlank(text, cursor)
	case motionGG:
		return MoveToLine(text, count)
	case motionG:
		return MoveToEndOfBuffer(text)
	case motionOBrace:
		return MoveParagraphBackward(text, cursor, count)
	case motionCBrace:
		return MoveParagraphForward(text, cursor, count)
	case motionPercent:
		pos := MatchBracket(text, cursor)
		if pos < 0 {
			return cursor
		}
		return pos
	}
	return cursor
}

// moveVertical moves the cursor up (negative) or down (positive) by count
// lines, preserving horizontal column as best as possible (same as neovim's
// curs_up/curs_down behavior).
//
// In our flat buffer, lines are separated by \n. Moving up/down means:
//   - Down: find the \n after the current line start, move past it
//   - Up:   find the \n before the current line start, move past it
//
// Column is preserved but clamped to the target line's length.
func moveVertical(text []rune, cursor int, count int) int {
	n := len(text)
	if n == 0 {
		return 0
	}
	if count == 0 {
		return cursor
	}

	lineStart := LineStart(text, cursor)
	col := max(cursor-lineStart, 0)

	if count > 0 {
		for range count {
			// Find the \n at end of the current line.
			end := lineStart
			for end < n && text[end] != '\n' {
				end++
			}
			if end >= n {
				break // already at last line
			}
			// Move past \n to start of next line.
			lineStart = end + 1
			if lineStart >= n {
				lineStart = n - 1
				break
			}
		}
	} else {
		for range -count {
			if lineStart <= 0 {
				break
			}
			// Walk backward from lineStart-1 to find the \n that
			// terminates the previous line.
			prevEnd := lineStart - 1
			for prevEnd >= 0 && text[prevEnd] != '\n' {
				prevEnd--
			}
			// The target line starts at prevEnd + 1.
			lineStart = prevEnd + 1
		}
	}

	// Clamp column to the target line.
	lineEnd := lineStart
	for lineEnd < n && text[lineEnd] != '\n' {
		lineEnd++
	}
	maxCol := lineEnd - lineStart
	if col > maxCol {
		col = maxCol
	}

	pos := lineStart + col
	if pos >= n {
		pos = n - 1
	}
	return pos
}

// ---------------------------------------------------------------------------
// Char search position resolution (for navigation)
// ---------------------------------------------------------------------------

func resolveCharPos(text []rune, cursor int, ch rune, dir int, till bool, count int) int {
	pos := cursor
	for range count {
		var next int
		if dir > 0 {
			if till {
				next = TillNext(text, pos, ch)
			} else {
				next = FindNext(text, pos, ch)
			}
		} else {
			if till {
				next = TillPrev(text, pos, ch)
			} else {
				next = FindPrev(text, pos, ch)
			}
		}
		if next < 0 {
			return pos // not found, stay at last successful position
		}
		pos = next
	}
	return pos
}

// ---------------------------------------------------------------------------
// Motion range resolution (for operator + motion)
// ---------------------------------------------------------------------------

func resolveMotionRange(text []rune, cursor int, mt motionType, count int) Range {
	dest := resolveMotionPos(text, cursor, mt, count)

	switch mt {
	case motionGG, motionG, motionJ, motionK:
		// Linewise: from LineStart(cursor) to end of destination line.
		start := LineStart(text, cursor)
		end := dest
		if end < start {
			end, start = start, end
		}
		// Extend end to include the full line.
		lineEndPos := end
		for lineEndPos < len(text) && text[lineEndPos] != '\n' {
			lineEndPos++
		}
		if lineEndPos < len(text) {
			lineEndPos++ // include the \n
		}
		return Range{start, lineEndPos}

	case motionH, motionL:
		return rangeFromCursor(text, cursor, dest, false)

	case motionW, motionB, motionBigW, motionBigB, motionGe, motionBigGe:
		return rangeFromCursor(text, cursor, dest, false)

	case motionE, motionBigE:
		return rangeFromCursor(text, cursor, dest, true)

	case motionZero:
		// d0: delete from line start to cursor, exclusive of cursor.
		start := LineStart(text, cursor)
		return Range{start, cursor}

	case motionCaret:
		// d^: delete from first non-blank to cursor, exclusive of cursor.
		start := FirstNonBlank(text, cursor)
		return Range{start, cursor}

	case motionDollar:
		// d$: delete from cursor to end of line, inclusive.
		end := LineEnd(text, cursor)
		if end < len(text) && text[end] != '\n' {
			end++ // include the last char
		}
		return Range{cursor, end}

	case motionG_:
		// dg_: delete from cursor to last non-blank, inclusive.
		end := LastNonBlank(text, cursor)
		if end < len(text) && text[end] != '\n' {
			end++
		}
		return Range{cursor, end}

	case motionOBrace:
		dest := MoveParagraphBackward(text, cursor, count)
		return Range{dest, cursor}

	case motionCBrace:
		dest := MoveParagraphForward(text, cursor, count)
		if dest < len(text) {
			dest++
		}
		return Range{cursor, dest}

	case motionPercent:
		return rangeFromCursor(text, cursor, dest, true)

	case motionF, motionBwdF:
		// f/F: exclusive of the found char? Actually in vim, df{char} deletes
		// UP TO AND INCLUDING the found char. So it's inclusive.
		return rangeFromCursor(text, cursor, dest, true)

	case motionT, motionBwdT:
		// t/T: exclusive of the found char (land before it).
		return rangeFromCursor(text, cursor, dest, false)
	}

	return rangeFromCursor(text, cursor, dest, false)
}

// rangeFromCursor returns the operator range for a motion from cursor to dest.
// If inclusive, the range extends one past dest to include it.
func rangeFromCursor(text []rune, cursor int, dest int, inclusive bool) Range {
	if dest == cursor {
		return Range{cursor, cursor}
	}

	start := cursor
	end := dest
	if dest < cursor {
		start, end = dest, cursor
	}

	if inclusive {
		if end < len(text) {
			end++
		}
	}

	return Range{start, end}
}

// ---------------------------------------------------------------------------
// Char search range resolution (for operator + f/t/F/T)
// ---------------------------------------------------------------------------

func resolveCharRange(text []rune, cursor int, ch rune, dir int, till bool, count int) Range {
	dest := resolveCharPos(text, cursor, ch, dir, till, count)
	if dest < 0 {
		return Range{cursor, cursor}
	}
	// f/F: inclusive (includes the target char).
	// t/T: exclusive (lands before/after, doesn't include target).
	inclusive := !till
	return rangeFromCursor(text, cursor, dest, inclusive)
}

// ---------------------------------------------------------------------------
// Exported helpers for vim up/down motion (approximate)
// ---------------------------------------------------------------------------

// MoveDown moves the cursor down count lines while preserving column.
func MoveDown(text []rune, cursor, count int) int {
	return moveVertical(text, cursor, count)
}

// MoveUp moves the cursor up count lines while preserving column.
func MoveUp(text []rune, cursor, count int) int {
	return moveVertical(text, cursor, -count)
}
