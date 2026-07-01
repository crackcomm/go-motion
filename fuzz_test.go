package motion

import (
	"math/rand"
	"strings"
	"testing"
)

// flatPos returns the flat buffer position for (line, col) in the joined text.
func flatPos(lines []string, line, col int) int {
	p := 0
	for i := 0; i < line; i++ {
		p += len(lines[i]) + 1 // +1 for \n
	}
	return p + col
}

func FuzzMotionConsistency(f *testing.F) {
	// Seed corpus: edge cases
	for _, s := range []string{
		"a",
		"a\n",
		"\n",
		"a\nb",
		"a\n\n",
		"a\nb\n",
		"a\n\nb",
		"\n\n",
		"\na",
		"ab\n\nxy",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// Split into lines (trailing \n produces a final empty line).
		// strings.Split("a\nb\n", "\n") → ["a", "b", ""] ✓
		// strings.Split("a\n", "\n")   → ["a", ""]    ✓
		// strings.Split("a", "\n")     → ["a"]        ✓
		// strings.Split("\n", "\n")    → ["", ""]     ✓
		lines := strings.Split(s, "\n")

		// Limit lines to 5, each line to 3 chars to keep tests focused.
		if len(lines) > 5 || len(lines) == 0 {
			return
		}
		for _, l := range lines {
			if len(l) > 3 {
				return
			}
		}

		text := []rune(strings.Join(lines, "\n"))

		// Test every possible cursor (line, col).
		for lineIdx, line := range lines {
			for col := 0; col <= len(line); col++ {
				pos := flatPos(lines, lineIdx, col)

				// j — move down
				if lineIdx+1 < len(lines) {
					next := lines[lineIdx+1]
					maxTargetCol := len(next)
					if maxTargetCol > 0 {
						maxTargetCol--
					}
					tcol := col
					if tcol > maxTargetCol {
						tcol = maxTargetCol
					}
					want := flatPos(lines, lineIdx+1, tcol)
					got := MoveDown(text, pos, 1)
					if got != want {
						t.Fatalf("j: lines=%q from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got, want)
					}
				}

				// k — move up
				if lineIdx > 0 {
					prev := lines[lineIdx-1]
					maxTargetCol := len(prev)
					if maxTargetCol > 0 {
						maxTargetCol--
					}
					tcol := col
					if tcol > maxTargetCol {
						tcol = maxTargetCol
					}
					want := flatPos(lines, lineIdx-1, tcol)
					got := MoveUp(text, pos, 1)
					if got != want {
						t.Fatalf("k: lines=%q from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got, want)
					}
				}

			}
		}

		// Test h and l through the Engine (only exported path).
		for lineIdx, line := range lines {
			for col := 0; col <= len(line); col++ {
				pos := flatPos(lines, lineIdx, col)
				if pos >= len(text) {
					continue
				}

				// h — left
				if col > 0 {
					want := flatPos(lines, lineIdx, col-1)
					var e Engine
					got := e.Process(text, pos, Key('h'))
					if got.Kind != ResultNavigate || got.Cursor != want {
						t.Fatalf("h: lines=%q from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, want)
					}
				} else {
					// h at line start: no-op
					var e Engine
					got := e.Process(text, pos, Key('h'))
					if got.Kind != ResultNavigate || got.Cursor != pos {
						t.Fatalf("h at start: lines=%q from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, pos)
					}
				}

				// l — right
				if col < len(line)-1 {
					want := flatPos(lines, lineIdx, col+1)
					var e Engine
					got := e.Process(text, pos, Key('l'))
					if got.Kind != ResultNavigate || got.Cursor != want {
						t.Fatalf("l: lines=%q from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, want)
					}
				} else {
					// l at last char or past end: no-op
					var e Engine
					got := e.Process(text, pos, Key('l'))
					if got.Kind != ResultNavigate || got.Cursor != pos {
						t.Fatalf("l at end: lines=%q from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, pos)
					}
				}
			}
		}
	})
}

func TestMotionConsistencySeeded(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 5000; i++ {
		nlines := rng.Intn(5) + 1 // 1-5 lines
		lines := make([]string, nlines)
		for j := range lines {
			llen := rng.Intn(4) // 0-3 chars
			buf := make([]byte, llen)
			for k := range buf {
				buf[k] = byte('a' + rng.Intn(26))
			}
			lines[j] = string(buf)
		}
		text := []rune(strings.Join(lines, "\n"))

		for lineIdx, line := range lines {
			for col := 0; col <= len(line); col++ {
				pos := flatPos(lines, lineIdx, col)
				if pos >= len(text) {
					continue
				}

				// j — move down
				if lineIdx+1 < len(lines) {
					next := lines[lineIdx+1]
					// The library clamps target column to the last
					// content column of the target line (LineEnd returns
					// the last content character, not the \n). So for a
					// non-empty line, max column is len-1; for an empty
					// line, max column is 0.
					maxTargetCol := len(next)
					if maxTargetCol > 0 {
						maxTargetCol--
					}
					tcol := col
					if tcol > maxTargetCol {
						tcol = maxTargetCol
					}
					want := flatPos(lines, lineIdx+1, tcol)
					got := MoveDown(text, pos, 1)
					if got != want {
						t.Fatalf("j: lines=%v from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got, want)
					}
				}

				// k — move up
				if lineIdx > 0 {
					prev := lines[lineIdx-1]
					maxTargetCol := len(prev)
					if maxTargetCol > 0 {
						maxTargetCol--
					}
					tcol := col
					if tcol > maxTargetCol {
						tcol = maxTargetCol
					}
					want := flatPos(lines, lineIdx-1, tcol)
					got := MoveUp(text, pos, 1)
					if got != want {
						t.Fatalf("k: lines=%v from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got, want)
					}
				}

				// h — left (via Engine)
				if col > 0 {
					want := flatPos(lines, lineIdx, col-1)
					var e Engine
					got := e.Process(text, pos, Key('h'))
					if got.Kind != ResultNavigate || got.Cursor != want {
						t.Fatalf("h: lines=%v from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, want)
					}
				} else {
					var e Engine
					got := e.Process(text, pos, Key('h'))
					if got.Kind != ResultNavigate || got.Cursor != pos {
						t.Fatalf("h at start: lines=%v from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, pos)
					}
				}

				// l — right (via Engine)
				if col < len(line)-1 {
					want := flatPos(lines, lineIdx, col+1)
					var e Engine
					got := e.Process(text, pos, Key('l'))
					if got.Kind != ResultNavigate || got.Cursor != want {
						t.Fatalf("l: lines=%v from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, want)
					}
				} else {
					var e Engine
					got := e.Process(text, pos, Key('l'))
					if got.Kind != ResultNavigate || got.Cursor != pos {
						t.Fatalf("l at end: lines=%v from=(%d,%d) pos=%d got=%d want=%d",
							lines, lineIdx, col, pos, got.Cursor, pos)
					}
				}
			}
		}
	}
}
