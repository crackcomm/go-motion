package motion

//
// Line motions (0, $, ^, g_)
//
// These operate within the current line of the text buffer. In neovim,
// lines are delimited by \n. We treat the text as a flat []rune where
// each \n separates lines.
//
// Reference: nv_beginline() in normal.c:6053, nv_dollar() in normal.c:3933,
// nv_g_underscore_cmd() in normal.c:5309
//

// LineStart returns the start (column 0) of the line containing pos.
// Implements neovim's "0" motion (beginline(0) in edit.c:2389).
//
// In neovim, beginline(0) simply sets col = 0. With the 'sol' option
// (startofline), it also moves to the first non-blank when not in
// combination with vertical motion — this is handled at the operator
// level, not the motion level, so we don't implement it here.
//
// When cursor is on a \n character, we treat it as being at the START
// of the next line (the line after the \n). This matches the visual
// display: \n in the text creates a line break, and the cursor on the
// \n is positioned at the break between visual lines.
//
// Edge cases:
//   - Empty text: returns 0
//   - Cursor past end: returns start of last line
//   - Consecutive \n (empty line): the \n ADJACENT to the empty line
//     is treated as its start; skip forward to the content after it.
func LineStart(text []rune, pos int) int {
	if len(text) == 0 {
		return 0
	}
	if pos >= len(text) {
		pos = len(text) - 1
	}
	if pos < 0 {
		pos = 0
	}

	// Walk backward until we hit the start of text or a newline.
	for pos > 0 && text[pos-1] != '\n' {
		pos--
	}
	return pos
}

// LineEnd returns the position of the last character (or NUL past end)
// on the line containing pos.
//
// Implements neovim's "$" motion (nv_dollar() in normal.c:3933).
// In neovim, $ is an INCLUSIVE motion (the character at the target is
// included in operator ranges). $ sets w_curswant = MAXCOL so that
// subsequent up/down movements don't snap to end-of-line.
//
// Edge cases:
//   - Empty line: returns pos (the \n position)
//   - Cursor past end: returns end of text
func LineEnd(text []rune, pos int) int {
	if pos < 0 {
		pos = 0
	}

	n := len(text)
	if n == 0 {
		return 0
	}

	// Walk forward until we hit end of text or a newline.
	for pos < n && text[pos] != '\n' {
		pos++
	}

	if pos >= n {
		return n - 1 // last line, end of text
	}

	// text[pos] == '\n' at this point.
	// Check if this \n represents an empty line: the line is empty when
	// the character before it is also a \n (consecutive \n = "\n\n"), or
	// when this \n is at position 0.
	//
	// Empty line: there are 0 characters on this line. Both 0 and $ stay
	// on the \n itself (it's both the start and end).
	if pos == 0 || (pos > 0 && text[pos-1] == '\n') {
		return pos
	}

	// Non-empty line: \n is the terminator after the last character.
	return pos - 1
}

// FirstNonBlank returns the first non-whitespace character on the line
// containing pos. Implements neovim's "^" motion.
//
// In neovim, beginline(BL_WHITE | BL_FIX) (edit.c:2389):
//   - BL_WHITE: skip past leading whitespace
//   - BL_FIX: don't stop on a trailing NUL in an empty line
//
// The difference from "0" is that "0" goes to column 0, while "^" goes
// to the first non-blank character.
//
// Edge cases:
//   - Empty line: returns line start (same as 0)
//   - Line with only whitespace: returns line start (first non-blank
//     doesn't exist, so we stay at start)
//   - Trailing whitespace: not affected (we only skip leading whitespace)
func FirstNonBlank(text []rune, pos int) int {
	start := LineStart(text, pos)
	end := lineEndExclusive(text, pos)
	i := start

	for i < end && Classify(text[i]) == Space {
		i++
	}

	// If we found a non-blank, return its position.
	// Otherwise return line start (same behavior as neovim).
	if i < end || end == npos(text) {
		return i
	}
	return start
}

// LastNonBlank returns the last non-whitespace character on the line
// containing pos. Implements neovim's "g_" motion.
//
// In neovim, nv_g_underscore_cmd() (normal.c:5309) works like $ but
// then walks LEFT from end-of-line past whitespace to the last
// non-blank character. Like $, g_ is an INCLUSIVE motion.
//
// Unlike LineEnd (which deliberately operates on the previous line when
// cursor is on \n), LastNonBlank is consistent with LineStart and
// operates on the NEXT line when cursor is on \n. This means g_ and 0
// agree on which line starts after \n, while $ still goes to the end
// of the line before \n.
//
// Algorithm:
//  1. Go to end of line
//  2. Walk left while the character is whitespace
//  3. Stop at the first non-whitespace (or line start)
//
// Edge cases:
//   - Empty line: returns line start
//   - Line with only whitespace: returns line start
//   - Line with trailing whitespace: returns last non-whitespace char
func LastNonBlank(text []rune, pos int) int {
	start := LineStart(text, pos)

	// Walk forward from start to find the end of the line (inclusive).
	// We compute this independently of LineEnd because LineEnd and
	// LineStart disagree on which line \n belongs to.
	end := start
	n := len(text)
	for end < n && text[end] != '\n' {
		end++
	}
	if end >= n {
		end = n - 1
	} else {
		// end is at \n; last content char is one before.
		if end > start {
			end--
		}
	}

	// Walk left from the end, skipping whitespace.
	for end > start && Classify(text[end]) == Space {
		end--
	}

	return end
}

// ---------------------------------------------------------------------------
// Helper: lineEndExclusive returns the position AFTER the last character
// on the line (i.e., the \n position or len(text)).
// ---------------------------------------------------------------------------

func lineEndExclusive(text []rune, pos int) int {
	if pos < 0 {
		pos = 0
	}

	// When cursor is on \n, skip forward past consecutive \n to the
	// start of the next content line, consistent with LineStart.
	if pos < len(text) && text[pos] == '\n' {
		for pos < len(text) && text[pos] == '\n' {
			pos++
		}
	}

	n := len(text)
	for pos < n && text[pos] != '\n' {
		pos++
	}
	return pos
}

func npos(text []rune) int {
	return len(text)
}
