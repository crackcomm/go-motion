package motion

//
// Character search motions (f, F, t, T, ; ,)
//
// These search for a specific character within the current line.
// Reference: searchc() in neovim's search.c:1567-1650, nv_csearch() in normal.c:4055
//
// All character searches:
//   - Stay within the current line
//   - Are multi-byte aware (operate on runes, not bytes)
//   - Support counts (e.g., "3fx" finds the 3rd occurrence)
//
// Neovim repeat state (for ; and ,) is not managed here — the caller
// is responsible for tracking the last search character and direction.
//

// FindNext implements "f{char}" — forward to the next occurrence of char
// on the current line. Returns -1 if not found.
//
// In neovim (searchc in search.c:1567):
//  1. Move to next character start (utfc_ptr2len)
//  2. Compare character at that position with target
//  3. Repeat count times
//  4. inclusive = (dir != BACKWARD) — forward finds are inclusive
//
// We operate on runes, so no multi-byte complexity.
func FindNext(text []rune, pos int, char rune) int {
	return findChar(text, pos, char, 1, false)
}

// FindPrev implements "F{char}" — backward to the previous occurrence
// of char on the current line. Returns -1 if not found.
func FindPrev(text []rune, pos int, char rune) int {
	return findChar(text, pos, char, -1, false)
}

// TillNext implements "t{char}" — forward to just before the next
// occurrence of char (lands one character left of it). Returns -1
// if not found.
func TillNext(text []rune, pos int, char rune) int {
	return findChar(text, pos, char, 1, true)
}

// TillPrev implements "T{char}" — backward to just after the previous
// occurrence of char (lands one character right of it). Returns -1
// if not found.
func TillPrev(text []rune, pos int, char rune) int {
	return findChar(text, pos, char, -1, true)
}

// findChar is the core character search implementation.
//
// Parameters:
//   - text: the text buffer
//   - pos: current cursor position
//   - char: the target character to find
//   - dir: 1 for forward, -1 for backward
//   - till: if true, land one char before/after the match (t/T behavior)
//
// Algorithm (from neovim's searchc):
//  1. Calculate line bounds (start/end of current line)
//  2. Move past current position in the given direction
//  3. For each character, check if it matches the target
//  4. If till=true: adjust final position by one in the reverse direction
//
// Returns -1 if the character is not found on the current line.
func findChar(text []rune, pos int, char rune, dir int, till bool) int {
	n := len(text)
	if n == 0 || pos < 0 || pos >= n {
		return -1
	}

	lineStart := LineStart(text, pos)
	lineEnd := lineEndExclusive(text, pos)

	if dir > 0 {
		// Forward search: start from pos+1
		i := pos + 1
		for i < lineEnd {
			if text[i] == char {
				if till {
					return i - 1
				}
				return i
			}
			i++
		}
	} else {
		// Backward search: start from pos-1
		i := pos - 1
		for i >= lineStart {
			if text[i] == char {
				if till {
					return i + 1
				}
				return i
			}
			i--
		}
	}

	return -1
}
