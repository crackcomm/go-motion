package motion

//
// Operator application
//
// ApplyOp modifies the text buffer by applying an operator over a range.
// It returns the new text, new cursor position, and whether to enter
// insert mode (for OpChange).
//
// The range is always [Start, End) where Start ≤ End.
//
// After deletion, the cursor is placed at the range Start.
// After change, the cursor is placed at the range Start and insert mode
// is requested.
// After yank, the text is returned in the YankText field of the result,
// and cursor stays at its original position.
//

// ApplyResult describes the result of applying an operator.
type ApplyResult struct {
	Text   []rune // modified text
	Cursor int    // new cursor position
	Insert bool   // true if caller should enter insert mode
	Yanked string // yanked text (for OpYank)
	Empty  bool   // true if the operation was a no-op (empty range)
}

// ApplyOp applies operator op over range [r.Start, r.End) in text,
// starting from cursor position pos.
//
// The range must satisfy 0 ≤ r.Start ≤ r.End ≤ len(text).
func ApplyOp(op Op, text []rune, pos int, r Range) ApplyResult {
	n := len(text)

	// Clamp range to valid bounds.
	start := r.Start
	end := r.End
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	if start > end {
		start, end = end, start
	}

	if start == end {
		if op == OpChange {
			return ApplyResult{Text: text, Cursor: start, Insert: true}
		}
		return ApplyResult{Text: text, Cursor: pos, Empty: true}
	}

	switch op {
	case OpDelete:
		newText := make([]rune, 0, n-(end-start))
		newText = append(newText, text[:start]...)
		newText = append(newText, text[end:]...)
		newCursor := min(start, len(newText))
		return ApplyResult{Text: newText, Cursor: newCursor, Insert: false}

	case OpChange:
		newText := make([]rune, 0, n-(end-start))
		newText = append(newText, text[:start]...)
		newText = append(newText, text[end:]...)
		newCursor := min(start, len(newText))
		return ApplyResult{Text: newText, Cursor: newCursor, Insert: true}

	case OpYank:
		yanked := string(text[start:end])
		return ApplyResult{Text: text, Cursor: pos, Insert: false, Yanked: yanked}

	default:
		return ApplyResult{Text: text, Cursor: pos, Empty: true}
	}
}
