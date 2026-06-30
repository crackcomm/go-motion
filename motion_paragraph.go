package motion

//
// Paragraph motions ({, })
//
// Reference: findpar() in textobject.c:162-256, nv_findpar() in normal.c:4467
//
// In neovim, paragraphs are separated by empty lines. An "empty line"
// is a line whose first character is NUL (in our flat model: a line
// consisting only of a \n character, i.e., two consecutive \n).
//
// Paragraph motion algorithm (findpar with what=NUL):
//   1. For each count:
//      a. Set did_skip = false
//      b. Loop over lines in direction dir:
//         - If line is NOT empty: did_skip = true
//         - If !first_line && did_skip && startPS(line):
//           break (found paragraph boundary)
//         - Move to next line
//         - If out of bounds and count > 0: return FAIL
//   2. Cursor lands on first line of the paragraph.
//      If at end of file and going forward: cursor on last char, inclusive.
//
// Our flat model maps lines to ranges delimited by \n. A blank line is
// one where the character at line_start is \n (consecutive \n).
//
// startPS(lnum, NUL, false): returns true when the line is empty
// (starts with NUL), contains \f, or starts with the 'paragraphs'
// option. We simplify to: returns true when the line is empty.
//

// isParagraphStart returns true if position i is the start of a paragraph.
// A paragraph start is either position 0, or the first character of a
// content line that follows an empty line (i.e., preceded by \n\n).
func isParagraphStart(text []rune, i int) bool {
	if i <= 0 {
		return i == 0
	}
	if i >= len(text) {
		return false
	}
	// Must be preceded by a blank line: two consecutive \n ending at i-1.
	return text[i-1] == '\n' && text[i-2] == '\n'
}

// MoveParagraphForward implements } — move forward to the start of the
// next paragraph.
//
// Algorithm (adapted from findpar with dir=FORWARD, what=NUL):
//  1. For each count:
//     a. If on an empty line, skip forward past it.
//     b. Walk forward through non-empty lines (the current paragraph).
//     c. Stop at the next empty line (paragraph boundary).
//     d. Skip past the empty lines to the start of the next paragraph.
//  2. If no next paragraph exists, stay at the current position.
//
// Edge cases:
//   - Already at the last paragraph: no-op
//   - Inside blank lines: skip forward to the next paragraph
func MoveParagraphForward(text []rune, pos, count int) int {
	n := len(text)
	if n == 0 {
		return 0
	}

	for count > 0 {
		// Step 1: skip blank lines at the current position.
		wasBlank := false
		for pos < n && text[pos] == '\n' {
			pos++
			wasBlank = true
		}

		if !wasBlank {
			// We started in content. Walk forward through the paragraph
			// to find the blank-line boundary that ends it.
			for pos < n {
				if text[pos] == '\n' && (pos+1 >= n || text[pos+1] == '\n') {
					break
				}
				pos++
			}
			// Step 2: skip past the blank line to the next paragraph start.
			for pos < n && text[pos] == '\n' {
				pos++
			}
		}
		// If we were on a blank line, we already skipped to the start of
		// the next paragraph — that's the destination.

		count--
	}

	if pos > n {
		pos = n
	}

	return pos
}

// MoveParagraphBackward implements { — move backward to the start of the
// current or previous paragraph.
//
// Algorithm (adapted from findpar with dir=BACKWARD, what=NUL):
//  1. For each count:
//     a. If on an empty line, skip backward past it.
//     b. Walk backward through non-empty lines to find the paragraph
//     boundary before this paragraph.
//     c. Skip forward past the boundary to the paragraph start.
//  2. If no previous paragraph exists, go to position 0.
//
// If already at a paragraph start, moves to the start of the previous
// paragraph (if any).
func MoveParagraphBackward(text []rune, pos, count int) int {
	n := len(text)
	if n == 0 {
		return 0
	}
	if pos > n {
		pos = n
	}

	for count > 0 {
		// Step 1a: skip empty lines backward.
		for pos > 0 && text[pos-1] == '\n' {
			pos--
		}

		// If already at position 0, nowhere to go.
		if pos == 0 {
			break
		}

		// Check if we're at a paragraph start (first char after blank line).
		if isParagraphStart(text, pos) {
			// Already at a paragraph start. Move backward past the
			// blank line before it to find the previous paragraph.
			// Enter the blank line that precedes this paragraph.
			pos--
			for pos > 0 && text[pos] == '\n' {
				pos--
			}
		}

		// Step 1b: walk backward through this paragraph's content
		// to find the blank line boundary before it.
		for pos > 0 {
			if text[pos-1] == '\n' && pos > 1 && text[pos-2] == '\n' {
				// Found blank line at (pos-2, pos-1).
				// Step 1c: advance past it to paragraph start.
				break
			}
			if pos == 1 && text[0] == '\n' {
				// Position 1 is \n and position 0 is something
				// (or \n). This is a boundary at the very start.
				break
			}
			pos--
		}

		// Step 1c: advance past blank lines to paragraph start.
		for pos < n && text[pos] == '\n' {
			pos++
		}

		count--
	}

	if pos < 0 {
		pos = 0
	}

	// If already at the first paragraph, stay at start of text.
	if pos == n {
		pos = 0
	}

	return pos
}
