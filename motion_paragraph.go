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
// next paragraph (the next empty line).
//
// Algorithm (adapted from findpar with dir=FORWARD, what=NUL):
//  1. For each count:
//     a. If on an empty line, skip forward past consecutive empty lines.
//     b. Walk forward through non-empty lines (the current paragraph).
//     c. Stop at the next empty line (paragraph boundary).
//  2. If no next empty line exists, go to the end of the text.
//
// Edge cases:
//   - Already at the end of the text: no-op
//   - Inside blank lines: skip forward to the next paragraph's ending empty line
func MoveParagraphForward(text []rune, pos, count int) int {
	n := len(text)
	if n == 0 {
		return 0
	}

	for count > 0 {
		// Skip empty lines at the current position.
		for pos < n && text[pos] == '\n' && (pos == 0 || text[pos-1] == '\n') {
			pos++
		}

		// Walk forward through the paragraph to find the next empty line.
		for pos < n {
			if text[pos] == '\n' && (pos+1 >= n || text[pos+1] == '\n') {
				// Land on the empty line (after the newline of the content line)
				if pos+1 < n {
					pos++
				}
				break
			}
			pos++
		}

		count--
	}

	if pos > n {
		pos = n
	}

	return pos
}

// MoveParagraphBackward implements { — move backward to the start of the
// current or previous paragraph (the previous empty line).
//
// Algorithm (adapted from findpar with dir=BACKWARD, what=NUL):
//  1. For each count:
//     a. If on an empty line, skip backward past consecutive empty lines.
//     b. Walk backward through non-empty lines to find the paragraph
//     boundary before this paragraph.
//     c. Stop at the previous empty line.
//  2. If no previous empty line exists, go to position 0.
func MoveParagraphBackward(text []rune, pos, count int) int {
	n := len(text)
	if n == 0 {
		return 0
	}
	if pos > n {
		pos = n
	}

	for count > 0 {
		// Skip empty lines backward.
		for pos > 0 && text[pos] == '\n' && text[pos-1] == '\n' {
			pos--
		}

		// Walk backward through the paragraph to find the previous empty line.
		for pos > 0 {
			pos--
			if text[pos] == '\n' && (pos == 0 || text[pos-1] == '\n') {
				break
			}
		}

		count--
	}

	if pos < 0 {
		pos = 0
	}

	return pos
}
