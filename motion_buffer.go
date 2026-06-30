package motion

//
// Buffer motions (gg, G)
//
// gg moves to the start of the buffer (position 0).
// With a count (e.g., 5gg or 5G), moves to the start of line N.
//
// G without a count moves to the end of the last line.
// With a count, moves to the start of line N.
//
// Reference: nv_goto() in normal.c:4558, nv_gg() in normal.c:4484
//

// MoveToLine implements gg/G with count support.
//
// With count > 0: move to the start of line N (1-indexed).
//   - count=1 → position 0 (start of first line)
//   - count=2 → start of second line (after first \n)
//   - count=len(lines) → start of last line
//
// With count == 0: gg goes to position 0, G goes to the last line.
//
// Returns the position of the first character on the target line.
// If count exceeds the number of lines, moves to the last line.
func MoveToLine(text []rune, count int) int {
	n := len(text)
	if n == 0 {
		return 0
	}

	if count <= 0 {
		count = 1
	}

	// Walk through text, counting lines.
	line := 1
	for i, ch := range text {
		if line >= count {
			return i
		}
		if ch == '\n' {
			line++
		}
	}

	// count exceeds number of lines: return start of last line.
	// Walk backward from the end to find the last \n.
	if text[n-1] == '\n' {
		return n
	}
	for i := n - 1; i >= 0; i-- {
		if text[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// MoveToEndOfBuffer implements G without a count — go to the end of the
// last line.
func MoveToEndOfBuffer(text []rune) int {
	n := len(text)
	if n == 0 {
		return 0
	}
	return n - 1
}
