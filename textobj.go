package motion

//
// Text objects (iw, aw, iW, aW, i", a", i', a', i(, a(, i[, a[, i{, a{,
//                i<, a<, it, at, ip, ap)
//
// Reference: nv_object() in normal.c:5757, textobject.c
//
// Text objects are ranges within the buffer. They come in two variants:
//   - "inner" (i): selects the content without surrounding delimiters
//   - "a" (a): selects the content including surrounding delimiters
//
// Each text object function takes the text buffer and cursor position
// and returns the Range to operate on.
//

// ---------------------------------------------------------------------------
// Word text objects (iw, aw)
// ---------------------------------------------------------------------------

// WordInner returns the range of the word under the cursor (iw).
// The word is defined by Classify: consecutive characters of the same
// non-Space class. Trailing whitespace is NOT included.
//
// Note: count is not yet supported — only the first word is returned.
func WordInner(text []rune, pos int) Range {
	return wordTextObj(text, pos, Classify, false)
}

// WordA returns the range of the word under or adjacent to the cursor (aw).
// Includes trailing whitespace after the word. If the cursor is in
// whitespace, includes the preceding word and its trailing whitespace
// (which is the whitespace at the cursor, or the word before it).
//
// Note: count is not yet supported — only the first word is returned.
func WordA(text []rune, pos int) Range {
	return wordTextObj(text, pos, Classify, true)
}

// BigWordInner returns the range of the WORD under the cursor (iW).
func BigWordInner(text []rune, pos int) Range {
	return wordTextObj(text, pos, ClassifyBigword, false)
}

// BigWordA returns the range of the WORD under or adjacent to cursor (aW).
func BigWordA(text []rune, pos int) Range {
	return wordTextObj(text, pos, ClassifyBigword, true)
}

func wordTextObj(text []rune, pos int, cf func(rune) Class, aWord bool) Range {
	n := len(text)
	if n == 0 || pos < 0 || pos >= n {
		return Range{0, 0}
	}

	start := pos
	end := pos

	// If on whitespace, find the nearest word.
	// In neovim, iw from whitespace searches forward first, then backward.
	if pos < n && cf(text[pos]) == Space {
		// Search forward for a word.
		fwd := pos
		for fwd < n && cf(text[fwd]) == Space {
			fwd++
		}
		if fwd < n && cf(text[fwd]) != Space {
			start = fwd
			end = fwd
		} else {
			// No word found forward; search backward.
			bwd := pos
			for bwd > 0 && cf(text[bwd]) == Space {
				bwd--
			}
			if bwd >= 0 && cf(text[bwd]) != Space {
				start = bwd
				end = bwd
			}
		}
	}

	// Now both start and end point to the same position within the word.
	// Expand backward to word start.
	cc := cf(text[start])
	for start > 0 && cf(text[start-1]) == cc {
		start--
	}

	// Expand forward to word end.
	for end < n && cf(text[end]) == cc {
		end++
	}

	if aWord {
		// Include trailing whitespace up to next word.
		for end < n && cf(text[end]) == Space {
			end++
		}
	}

	return Range{start, end}
}

// ---------------------------------------------------------------------------
// Quote text objects (i", a", i', a', i`, a`)
// ---------------------------------------------------------------------------

// QuoteInner returns the range inside the nearest quote pair (i", i', i`).
// If the cursor is inside a quoted string, returns the content between
// the quotes. If the cursor is on a quote character, returns the content
// of the string that quote delimits.
func QuoteInner(text []rune, pos int, quote rune) Range {
	return quoteTextObj(text, pos, quote, false)
}

// QuoteA returns the range including the nearest quote pair (a", a', a`).
func QuoteA(text []rune, pos int, quote rune) Range {
	return quoteTextObj(text, pos, quote, true)
}

func quoteTextObj(text []rune, pos int, quote rune, aQuote bool) Range {
	n := len(text)
	if n == 0 || pos < 0 || pos >= n {
		return Range{0, 0}
	}

	// Find the nearest quote pair. We scan backward from pos to find
	// an opening quote, and forward from pos to find the closing quote.
	//
	// If the cursor is ON a quote, that quote is the opening one, and
	// we search forward for the matching closing quote.

	open := -1
	close := -1

	// If cursor is on a quote, treat it as the opening quote.
	if text[pos] == quote {
		open = pos
	} else {
		// Scan backward for the opening quote.
		for i := pos; i >= 0; i-- {
			if text[i] == quote {
				// Check if this quote has a matching closing quote.
				// Simple heuristic: a quote is an opening quote if
				// there's a matching closing quote after it.
				for j := i + 1; j < n; j++ {
					if text[j] == quote {
						open = i
						close = j
						break
					}
				}
				if open >= 0 {
					break
				}
			}
		}
	}

	if open < 0 {
		// No opening quote found.
		return Range{pos, pos}
	}

	// Find the closing quote after open.
	for j := open + 1; j < n; j++ {
		if text[j] == quote {
			close = j
			break
		}
	}

	if close < 0 {
		return Range{pos, pos}
	}

	if aQuote {
		return Range{open, close + 1}
	}
	return Range{open + 1, close}
}

// ---------------------------------------------------------------------------
// Bracket text objects (i(, a(, i[, a[, i{, a{, i<, a<)
// Also ib (inner block), iB (inner brace), ab, aB
// ---------------------------------------------------------------------------

// PairInner returns the range inside the nearest matching bracket pair
// (i(, i[, i{, i<). Uses the MatchBracket logic to find the pair.
func PairInner(text []rune, pos int, open, close rune) Range {
	return pairTextObj(text, pos, open, close, false)
}

// PairA returns the range including the nearest bracket pair (a(, a[, a{, a<).
func PairA(text []rune, pos int, open, close rune) Range {
	return pairTextObj(text, pos, open, close, true)
}

func pairTextObj(text []rune, pos int, open, close rune, aPair bool) Range {
	n := len(text)
	if n == 0 || pos < 0 || pos >= n {
		return Range{0, 0}
	}

	// If cursor is on an opening bracket, find its matching close.
	if text[pos] == open {
		cl := findMatchForward(text, pos, open, close)
		if cl < 0 {
			return Range{pos, pos}
		}
		if aPair {
			return Range{pos, cl + 1}
		}
		return Range{pos + 1, cl}
	}

	// If cursor is on a closing bracket, find its matching open.
	if text[pos] == close {
		op := findMatchBackward(text, pos, open, close)
		if op < 0 {
			return Range{pos, pos}
		}
		if aPair {
			return Range{op, pos + 1}
		}
		return Range{op + 1, pos}
	}

	// If cursor is inside a pair, find the innermost enclosing pair.
	// Scan backward for an opening bracket, then forward for its match.
	// If that pair doesn't enclose the cursor, scan further.
	//
	// Simplified: find the nearest opening bracket (open or close type)
	// and match across it.

	// Forward scan: find the first matching pair that encloses pos.
	for i := pos; i >= 0; i-- {
		if text[i] == open {
			cl := findMatchForward(text, i, open, close)
			if cl >= 0 && i <= pos && cl >= pos {
				// This pair encloses pos.
				if aPair {
					return Range{i, cl + 1}
				}
				return Range{i + 1, cl}
			}
		}
		if text[i] == close {
			op := findMatchBackward(text, i, open, close)
			if op >= 0 && op <= pos && i >= pos {
				if aPair {
					return Range{op, i + 1}
				}
				return Range{op + 1, i}
			}
		}
	}

	return Range{pos, pos}
}

// ---------------------------------------------------------------------------
// Tag text objects (it, at)
// ---------------------------------------------------------------------------

// TagInner returns the range inside the nearest HTML/XML tag pair (it).
func TagInner(text []rune, pos int) Range {
	return tagTextObj(text, pos, false)
}

// TagA returns the range including the tag pair and its content (at).
func TagA(text []rune, pos int) Range {
	return tagTextObj(text, pos, true)
}

func tagTextObj(text []rune, pos int, aTag bool) Range {
	n := len(text)
	if n == 0 || pos < 0 || pos >= n {
		return Range{0, 0}
	}

	// Find the nearest tag pair. We scan backward for a closing tag
	// </tagname> and forward for the matching opening tag <tagname>.

	// Step 1: find a closing tag </...> at or after pos.
	closeStart := -1
	closeEnd := -1
	tagName := ""

	for i := pos; i < n; i++ {
		if text[i] == '<' && i+1 < n && text[i+1] == '/' {
			// Found </ — find the matching >
			j := i + 2
			for j < n && text[j] != '>' {
				j++
			}
			if j < n && text[j] == '>' {
				closeStart = i
				closeEnd = j + 1
				tagName = string(text[i+2 : j])
				break
			}
		}
	}

	if closeStart < 0 || tagName == "" {
		return Range{pos, pos}
	}

	// Step 2: find the matching opening tag <tagname ...> before closeStart.
	openStart := -1
	openEnd := -1

	tagRunes := []rune(tagName)
	for i := closeStart; i >= 0; i-- {
		if text[i] == '<' && i+1 < n && text[i+1] != '/' {
			// Found < — check if it matches the tag name.
			j := i + 1
			k := 0
			match := true
			for j < n && text[j] != '>' && k < len(tagRunes) {
				if text[j] != tagRunes[k] {
					match = false
					break
				}
				j++
				k++
			}
			if match {
				// Skip past tag name to find >. Tag may have attributes.
				for j < n && text[j] != '>' {
					if text[j] == '/' && j+1 < n && text[j+1] == '>' {
						// Self-closing tag, skip.
						match = false
						break
					}
					j++
				}
				if match && j < n && text[j] == '>' {
					openStart = i
					openEnd = j + 1
					break
				}
			}
		}
	}

	if openStart < 0 {
		return Range{pos, pos}
	}

	if aTag {
		return Range{openStart, closeEnd}
	}
	// inner: content between > and <
	return Range{openEnd, closeStart}
}

// ---------------------------------------------------------------------------
// Paragraph text objects (ip, ap)
// ---------------------------------------------------------------------------

// ParagraphInner returns the range of the current paragraph (ip).
// The paragraph is delimited by blank lines (\n\n).
func ParagraphInner(text []rune, pos int) Range {
	return paragraphTextObj(text, pos, false)
}

// ParagraphA returns the range including surrounding blank lines (ap).
func ParagraphA(text []rune, pos int) Range {
	return paragraphTextObj(text, pos, true)
}

func paragraphTextObj(text []rune, pos int, aParagraph bool) Range {
	n := len(text)
	if n == 0 {
		return Range{0, 0}
	}

	// Find the start of the current paragraph.
	paraStart := pos
	if !isParagraphStart(text, pos) {
		for i := pos; i >= 0; i-- {
			if isParagraphStart(text, i) {
				paraStart = i
				break
			}
		}
	}

	// Find the end of the current paragraph.
	paraEnd := paraStart
	for paraEnd < n {
		if text[paraEnd] == '\n' && (paraEnd+1 >= n || text[paraEnd+1] == '\n') {
			break
		}
		paraEnd++
	}

	if aParagraph {
		// Include the blank lines after the paragraph.
		for paraEnd < n && text[paraEnd] == '\n' {
			paraEnd++
		}
		// Include blank lines before the paragraph.
		for paraStart > 0 && text[paraStart-1] == '\n' {
			paraStart--
		}
	}

	return Range{paraStart, paraEnd}
}
