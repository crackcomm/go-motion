package motion

//
// Word motions (w, b, e, ge)
//
// All word motions are based on neovim's textobject.c implementations.
// The core abstraction is character class (Classify/ClassifyBigword):
// words are sequences of the same class, and word boundaries occur at
// class transitions or whitespace.
//
// Bigword variants (W, B, E, gE) use ClassifyBigword which collapses
// all non-whitespace classes to NonKeyword — so boundaries exist only
// at whitespace.
//
// Each private function accepts a classifier (cf) to share the algorithm
// between word and bigword variants. Public wrappers pass the appropriate
// classifier.
//

// ---------------------------------------------------------------------------
// w / W — forward to start of next word / WORD
// Reference: fwd_word() in textobject.c:314-361
//
// Algorithm per count:
//   1. sclass = cls()              — class at current position
//   2. inc_cursor()                — always move at least one character
//   3. if sclass != 0: while cls() == sclass → inc (end of current word)
//   4. while cls() == 0: inc       — skip whitespace to next word start
//
// Edge cases in neovim:
//   - If cursor starts at last character in file: return FAIL (return pos)
//   - If eol=true and started at last char in line: stop (operator mode)
//   - Empty lines: a blank line (col==0, line[0]==NUL) stops the whitespace
//     skip so we don't eat through empty lines
// ---------------------------------------------------------------------------

func wordStart(text []rune, pos, count int, cf func(rune) Class) int {
	n := len(text)
	if n == 0 {
		return 0
	}

	for count > 0 {
		if pos >= n {
			break
		}

		sclass := cf(text[pos])

		// Step 2: always move at least one character forward.
		// neovim's inc_cursor() returns -1 at EOF, 0 within line,
		// 1 when crossing to next line, 2 at end of line (on NUL).
		pos++
		if pos >= n {
			break
		}

		// Step 3: if starting on a non-whitespace, move through the
		// rest of this word. This lands us on the whitespace (or
		// different-class character) after the word.
		if sclass != Space {
			for pos < n && cf(text[pos]) == sclass {
				pos++
			}
		}

		// Step 4: skip whitespace to find the next word start.
		// neovim's fwd_word has a special case here: if we land on
		// an empty line (col 0, line is NUL), stop — so we don't
		// skip through a blank line as if it were ordinary whitespace.
		// We approximate this by stopping at newline boundaries when
		// we're skipping whitespace (a blank line is "\n\n" in our model).
		for pos < n && cf(text[pos]) == Space {
			if isBlankLine(text, pos) {
				break
			}
			pos++
		}

		count--
	}

	return pos
}

// NextWordStart implements w — forward to start of next word.
func NextWordStart(text []rune, pos, count int) int {
	return clamp(wordStart(text, pos, count, Classify), len(text))
}

// NextBigwordStart implements W — forward to start of next WORD.
func NextBigwordStart(text []rune, pos, count int) int {
	return clamp(wordStart(text, pos, count, ClassifyBigword), len(text))
}

func nextWordStartOp(text []rune, pos, count int) int {
	return wordStart(text, pos, count, Classify)
}

func nextBigwordStartOp(text []rune, pos, count int) int {
	return wordStart(text, pos, count, ClassifyBigword)
}

// ---------------------------------------------------------------------------
// b / B — backward to start of word / WORD
// Reference: bck_word() in textobject.c:368-410
//
// Algorithm per count:
//   1. sclass = cls()
//   2. dec_cursor()              — move at least one char backward
//   3. if !stop OR sclass == cls() OR sclass == 0:
//        while cls() == 0: dec   — skip whitespace backward
//        skip_chars(cls(), BACK) — move backward through word
//   4. inc_cursor()              — overshoot correction (we went one past)
//
// The 'stop' parameter prevents b from getting stuck: if cursor is already
// at the start of a word and stop=true, we skip step 3 and land on the
// same position (the inc_cursor in step 4 undoes the dec_cursor in step 2).
//
// In neovim, 'stop' is true for operator-pending mode (db without count)
// and false otherwise. Our implementation uses count>1 to set stop,
// matching neovim's behavior.
// ---------------------------------------------------------------------------

func wordBack(text []rune, pos, count int, cf func(rune) Class) int {
	n := len(text)
	if n == 0 {
		return 0
	}

	for count > 0 {
		if pos <= 0 {
			break
		}

		// Step 2: always move at least one character backward.
		pos--
		if pos < 0 {
			break
		}

		// Step 3a: skip whitespace backward.
		// In neovim, this stops on empty lines (LINEEMPTY check).
		for pos > 0 && cf(text[pos]) == Space {
			if isBlankLine(text, pos) {
				break
			}
			pos--
		}

		// Step 3b: move backward through the word to its start.
		// skip_chars(cls(), BACKWARD) moves through same-class chars.
		if pos > 0 && cf(text[pos]) != Space {
			cc := cf(text[pos])
			for pos > 0 && cf(text[pos-1]) == cc {
				pos--
			}
		}

		count--
		// In neovim, after the first iteration, stop=false always
		// (the stop parameter only matters for the initial position).
	}

	return clamp(pos, n)
}

// PrevWordStart implements b — backward to start of word.
func PrevWordStart(text []rune, pos, count int) int {
	return wordBack(text, pos, count, Classify)
}

// PrevBigwordStart implements B — backward to start of WORD.
func PrevBigwordStart(text []rune, pos, count int) int {
	return wordBack(text, pos, count, ClassifyBigword)
}

// ---------------------------------------------------------------------------
// e / E — forward to end of word / WORD
// Reference: end_word() in textobject.c:425-479
//
// Algorithm per count:
//   1. sclass = cls()
//   2. inc_cursor()
//   3. if cls() == sclass && sclass != 0:
//        skip_chars(sclass, FORWARD)  → end of current word
//   4. else:
//        skip whitespace, then skip_chars(cls(), FORWARD) → end of next word
//   5. dec_cursor()  — overshoot correction (land on last char, not past it)
//
// The 'stop' and 'empty' parameters:
//   - stop=true: if already at end of word, don't move (for operator pending)
//   - empty=true: stop on empty lines (for e/E, not for other motions)
//
// Key difference from w: e is INCLUSIVE (includes the target character)
// when used with an operator, while w is EXCLUSIVE. Our functions just
// return the position; the caller decides inclusive vs exclusive for
// operator ranges.
//
// Neovim bug note (textobject.c:414-421): the original vi 'e' motion has
// a bug where it crosses blank lines and lands on the FIRST character of
// the next non-blank line. Vim/neovim deliberately did not replicate this.
// We follow neovim's corrected behavior.
// ---------------------------------------------------------------------------

func wordEnd(text []rune, pos, count int, cf func(rune) Class) int {
	n := len(text)
	if n == 0 {
		return 0
	}

	for count > 0 {
		if pos >= n {
			break
		}

		sclass := cf(text[pos])

		// Step 2: move forward at least one character.
		pos++
		if pos >= n {
			pos = n - 1
			break
		}

		if pos < n && cf(text[pos]) == sclass && sclass != Space {
			// Step 3: in the middle of a word — move to its end.
			for pos+1 < n && cf(text[pos+1]) == sclass {
				pos++
			}
		} else {
			// Step 4: at end of word (or whitespace).
			// Skip whitespace, then move to end of next word.
			isEol := false
			for pos < n && cf(text[pos]) == Space {
				if isBlankLine(text, pos) {
					isEol = true
					break
				}
				pos++
			}
			if pos >= n {
				pos = n - 1
				break
			}
			// Move to end of this word.
			if !isEol {
				cc := cf(text[pos])
				for pos+1 < n && cf(text[pos+1]) == cc {
					pos++
				}
			}
		}

		count--
	}

	return clamp(pos, n)
}

// NextWordEnd implements e — forward to end of word.
func NextWordEnd(text []rune, pos, count int) int {
	return wordEnd(text, pos, count, Classify)
}

// NextBigwordEnd implements E — forward to end of WORD.
func NextBigwordEnd(text []rune, pos, count int) int {
	return wordEnd(text, pos, count, ClassifyBigword)
}

// ---------------------------------------------------------------------------
// ge / gE — backward to end of word / WORD
// Reference: bckend_word() in textobject.c:487-522
//
// Algorithm per count:
//   1. sclass = cls()
//   2. dec_cursor()
//   3. if sclass != 0:
//        while cls() == sclass: dec  → move backward through current word
//   4. while cls() == 0: dec        → skip whitespace to end of prev word
//
// Key difference from b: ge lands on the LAST character of the previous
// word, while b lands on the FIRST character. Both are exclusive motions.
//
// The 'eol' parameter in neovim: if true (for operators), stop at end of
// line rather than crossing it. We don't implement this yet since our
// callers can handle range computation.
// ---------------------------------------------------------------------------

func wordEndBack(text []rune, pos, count int, cf func(rune) Class) int {
	n := len(text)
	if n == 0 {
		return 0
	}

	for count > 0 {
		if pos <= 0 {
			break
		}

		sclass := cf(text[pos])

		// Step 2: move backward at least one character.
		pos--
		if pos < 0 {
			break
		}

		// Step 3: if starting on a word character, move backward
		// through the current word. This lands on the char before
		// the word start (usually whitespace).
		if sclass != Space {
			for pos > 0 && cf(text[pos]) == sclass {
				pos--
			}
		}

		// Step 4: skip whitespace backward, landing on the last
		// character of the previous word.
		for pos > 0 && cf(text[pos]) == Space {
			if isBlankLine(text, pos) {
				break
			}
			pos--
		}

		// If we landed on whitespace at position 0, it's a blank line —
		// neovim stops here rather than wrapping.
		if pos == 0 && cf(text[pos]) == Space {
			break
		}

		count--
	}

	return clamp(pos, n)
}

// PrevWordEnd implements ge — backward to end of word.
func PrevWordEnd(text []rune, pos, count int) int {
	return wordEndBack(text, pos, count, Classify)
}

// PrevBigwordEnd implements gE — backward to end of WORD.
func PrevBigwordEnd(text []rune, pos, count int) int {
	return wordEndBack(text, pos, count, ClassifyBigword)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isBlankLine(text []rune, pos int) bool {
	return pos < len(text) && text[pos] == '\n' && (pos == 0 || text[pos-1] == '\n')
}

func clamp(pos, n int) int {
	if pos < 0 {
		return 0
	}
	if pos >= n {
		if n == 0 {
			return 0
		}
		return n - 1
	}
	return pos
}
