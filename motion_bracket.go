package motion

//
// Bracket matching motion (%)
//
// Reference: findmatchlimit() in neovim's search.c:1773-2341,
// nv_percent() in normal.c:4371
//
// The % motion finds the matching bracket. In neovim, it has two modes:
//   1. With a count (e.g., 50%): jump to percentage of file (linewise)
//   2. Without a count: find matching bracket under/after cursor
//
// We implement mode 2 only (the cursor-on-bracket case).
//
// Neovim's algorithm (simplified):
//   1. Look at character under cursor (or next char if cursor is at end)
//   2. Check for #if/#else/#endif at start of line
//   3. Check for /* or */ comment boundaries
//   4. Check matchpairs option (default: (:), {:}, [:])
//   5. Walk forward/backward, tracking nesting count
//
// We implement step 4 with the default matchpairs.
// String/comment awareness is not implemented (simplification).
//

// BracketPair defines an opening/closing bracket pair for % matching.
type BracketPair struct {
	Open  rune
	Close rune
}

// DefaultPairs are the default bracket pairs neovim uses (from matchpairs
// option, default: "(:),{:},[:]").
var DefaultPairs = []BracketPair{
	{Open: '(', Close: ')'},
	{Open: '{', Close: '}'},
	{Open: '[', Close: ']'},
}

// MatchBracket finds the matching bracket for % motion.
//
// Algorithm (simplified from neovim's findmatchlimit in search.c):
//  1. Find a bracket character at or after cursor
//  2. Look up its pair
//  3. Set direction: if opening bracket, go forward; if closing, go backward
//  4. Walk character by character, counting nesting:
//     On opening bracket: count++
//     On closing bracket: count--
//     When count reaches 0 and we hit a matching bracket: found it!
//  5. String/comment skipping not implemented (simplification)
//
// Returns -1 if no matching bracket is found.
//
// Reference: findmatchlimit() in search.c:1773, nv_percent() in normal.c:4371
func MatchBracket(text []rune, pos int) int {
	return MatchBracketWith(text, pos, DefaultPairs)
}

// MatchBracketWith is like MatchBracket but uses caller-specified pairs.
func MatchBracketWith(text []rune, pos int, pairs []BracketPair) int {
	n := len(text)
	if n == 0 {
		return -1
	}

	// Step 1: find a bracket at or after cursor.
	start := pos
	if start >= n {
		start = n - 1
	}

	for start < n {
		for _, p := range pairs {
			if text[start] == p.Open {
				return findMatchForward(text, start, p.Open, p.Close)
			}
			if text[start] == p.Close {
				return findMatchBackward(text, start, p.Open, p.Close)
			}
		}
		start++
	}

	return -1
}

// findMatchForward walks forward from pos, counting nesting, to find the
// matching closing bracket.
func findMatchForward(text []rune, pos int, open, close rune) int {
	n := len(text)
	count := 0

	for i := pos; i < n; i++ {
		ch := text[i]

		switch ch {
		case open:
			count++
		case close:
			count--
			if count == 0 {
				return i
			}
		}
	}

	return -1
}

// findMatchBackward walks backward from pos, counting nesting, to find the
// matching opening bracket.
func findMatchBackward(text []rune, pos int, open, close rune) int {
	count := 0

	for i := pos; i >= 0; i-- {
		ch := text[i]

		switch ch {
		case close:
			count++
		case open:
			count--
			if count == 0 {
				return i
			}
		}
	}

	return -1
}
