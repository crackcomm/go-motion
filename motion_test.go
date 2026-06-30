package motion

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		r    rune
		want Class
		desc string
	}{
		{' ', Space, "space"},
		{'\t', Space, "tab"},
		{'\n', Space, "newline"},
		{0xa0, Space, "non-breaking space"},
		{'a', Keyword, "lowercase letter"},
		{'Z', Keyword, "uppercase letter"},
		{'0', Keyword, "digit"},
		{'_', Keyword, "underscore"},
		{0xe9, Keyword, "e accented"},
		{0xf1, Keyword, "n accented"},
		{'.', NonKeyword, "period"},
		{',', NonKeyword, "comma"},
		{'/', NonKeyword, "slash"},
		{'-', NonKeyword, "hyphen"},
		{'(', NonKeyword, "open paren"},
		{')', NonKeyword, "close paren"},
		{0x4e00, Keyword, "CJK ideograph"},
		{0x3042, Keyword, "Hiragana"},
		{0xac00, Keyword, "Hangul"},
		{0x1f600, Emoji, "grinning face"},
		{0x2764, Emoji, "red heart"},
	}
	for _, tt := range tests {
		got := Classify(tt.r)
		if got != tt.want {
			t.Errorf("Classify(%q %s) = %d, want %d", tt.r, tt.desc, got, tt.want)
		}
	}
}

func TestClassifyBigword(t *testing.T) {
	if got := ClassifyBigword(' '); got != Space {
		t.Errorf("space should be Space, got %d", got)
	}
	if got := ClassifyBigword('a'); got != NonKeyword {
		t.Errorf("letter should be NonKeyword for bigword, got %d", got)
	}
	if got := ClassifyBigword('.'); got != NonKeyword {
		t.Errorf("punct should be NonKeyword for bigword, got %d", got)
	}
}

// --- Word motions ---

func TestNextWordStart_basic(t *testing.T) {
	text := []rune("hello world")
	cases := []struct{ pos, count, want int }{
		{0, 1, 6},  // 'h' -> through "hello" -> skip space -> 'w'
		{1, 1, 6},  // middle "hello" -> same
		{5, 1, 6},  // space -> 'w'
		{6, 1, 10}, // 'w': last word, through to end
	}
	for _, c := range cases {
		got := NextWordStart(text, c.pos, c.count)
		if got != c.want {
			t.Errorf("NextWordStart(text=%q, pos=%d, count=%d) = %d, want %d", string(text), c.pos, c.count, got, c.want)
		}
	}
}

func TestNextWordStart_punct(t *testing.T) {
	text := []rune("foo.bar")
	cases := []struct{ pos, want int }{
		{0, 3}, // 'f' -> 'f'=K, through "foo", land on '.'(NK)
		{3, 4}, // '.' -> '.'=NK, through '.', land on 'b'(K)
		{4, 6}, // 'b': last word, through to end
	}
	for _, c := range cases {
		got := NextWordStart(text, c.pos, 1)
		if got != c.want {
			t.Errorf("NextWordStart(%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

func TestNextWordStart_multi(t *testing.T) {
	text := []rune("x.y z")
	cases := []struct{ pos, count, want int }{
		{0, 1, 1}, // 1w: 'x'(K)->'.'(NK)
		{0, 2, 2}, // 2w: 'x'->'.'->'y'(K)
		{0, 3, 4}, // 3w: 'x'->'.'->'y'->skip space->'z'(K)
	}
	for _, c := range cases {
		got := NextWordStart(text, c.pos, c.count)
		if got != c.want {
			t.Errorf("NextWordStart(%q, pos=%d, count=%d) = %d, want %d", string(text), c.pos, c.count, got, c.want)
		}
	}
}

func TestNextWordStart_multiline(t *testing.T) {
	// h(0)e(1)l(2)l(3)o(4)\n(5)w(6)o(7)r(8)l(9)d(10)\n(11)f(12)o(13)o(14)
	text := []rune("hello\nworld\nfoo")
	cases := []struct{ pos, want int }{
		{0, 6},   // 'h' line1 -> through, skip \n -> 'w' line2
		{5, 6},   // \n after line1 -> 'w' line2
		{10, 12}, // 'd' line2 -> through, skip \n -> 'f' line3
	}
	for _, c := range cases {
		got := NextWordStart(text, c.pos, 1)
		if got != c.want {
			t.Errorf("NextWordStart(pos=%d) = %d, want %d", c.pos, got, c.want)
		}
	}
}

func TestNextWordStart_emptyLines(t *testing.T) {
	text := []rune("a\n\n\nb")
	cases := []struct{ pos, want int }{
		{0, 2}, // w from 'a' -> empty line (pos 2)
		{2, 3}, // w from empty line -> next empty line (pos 3)
		{3, 4}, // w from empty line -> 'b' (pos 4)
	}
	for _, c := range cases {
		got := NextWordStart(text, c.pos, 1)
		if got != c.want {
			t.Errorf("NextWordStart(%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

func TestPrevWordStart_basic(t *testing.T) {
	text := []rune("hello world foo")
	cases := []struct{ pos, count, want int }{
		{11, 1, 6}, // 'f' -> back through 'foo' -> skip space -> 'w'
		{12, 1, 6}, // 'o' in 'foo' -> same
		{12, 2, 0}, // 2b: 'f'->'w'->'h'
		{6, 1, 0},  // 'w' -> back -> 'h'
	}
	for _, c := range cases {
		got := PrevWordStart(text, c.pos, c.count)
		if got != c.want {
			t.Errorf("PrevWordStart(pos=%d, count=%d) = %d, want %d", c.pos, c.count, got, c.want)
		}
	}
}

func TestPrevWordStart_punct(t *testing.T) {
	text := []rune("foo.bar")
	cases := []struct{ pos, want int }{
		{7, 4}, // 'r': back through 'bar', prev word start = '.'(4... no, '.' is at 3)
		// Wait: text[7]='r', sclass=K. pos--=6('a'). while Space: 'a'!=Space. while same class back: 6('a'),5('r'),4('b').
		// pos=4, text[4]='b', Class='b'=K. text[3]='.'=NK != K, exit. Return 4?! That's not '.' at 3.
		// Hmm, actually the prev word from 'b' would be '.'. Let me think again.
	}
	for _, c := range cases {
		got := PrevWordStart(text, c.pos, 1)
		if got != c.want {
			t.Errorf("PrevWordStart(%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

func TestPrevWordStart_emptyLines(t *testing.T) {
	text := []rune("a\n\n\nb")
	cases := []struct{ pos, want int }{
		{4, 3}, // b from 'b' -> empty line (pos 3)
		{3, 2}, // b from empty line -> previous empty line (pos 2)
		{2, 0}, // b from empty line -> 'a' (pos 0)
	}
	for _, c := range cases {
		got := PrevWordStart(text, c.pos, 1)
		if got != c.want {
			t.Errorf("PrevWordStart(%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

func TestNextWordEnd_basic(t *testing.T) {
	text := []rune("hello world foo")
	cases := []struct{ pos, count, want int }{
		{0, 1, 4},  // e: end of "hello" -> 'o'
		{0, 2, 10}, // e e: end of "world" -> 'd'
		{0, 3, 14}, // e e e: end of "foo" -> 'o'
		{1, 1, 4},  // from middle 'e' -> end of "hello"
	}
	for _, c := range cases {
		got := NextWordEnd(text, c.pos, c.count)
		if got != c.want {
			t.Errorf("NextWordEnd(pos=%d, count=%d) = %d, want %d", c.pos, c.count, got, c.want)
		}
	}
}

func TestNextWordEnd_punct(t *testing.T) {
	text := []rune("foo.bar")
	cases := []struct{ pos, want int }{
		{0, 2}, // e from 'f': end of 'foo' = 'o'(2)
		{3, 6}, // e from '.': skip (no space), end of 'bar' = 'r'(6)
	}
	for _, c := range cases {
		got := NextWordEnd(text, c.pos, 1)
		if got != c.want {
			t.Errorf("NextWordEnd(%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

func TestNextWordEnd_emptyLines(t *testing.T) {
	text := []rune("a\n\n\nb")
	cases := []struct{ pos, want int }{
		{0, 2}, // e from 'a' -> empty line (pos 2)
		{2, 3}, // e from empty line -> next empty line (pos 3)
		{3, 4}, // e from empty line -> 'b' (pos 4)
	}
	for _, c := range cases {
		got := NextWordEnd(text, c.pos, 1)
		if got != c.want {
			t.Errorf("NextWordEnd(%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

func TestPrevWordEnd_basic(t *testing.T) {
	// h(0)e(1)l(2)l(3)o(4) (5)w(6)o(7)r(8)l(9)d(10) (11)f(12)o(13)o(14)
	text := []rune("hello world foo")
	cases := []struct{ pos, count, want int }{
		{14, 1, 10}, // ge from 'o'(14): back through 'foo' -> skip space -> end of 'world'='d'(10)
		{14, 2, 4},  // ge ge: ->10('d') -> skip space -> end of 'hello'='o'(4)
		{14, 3, 0},  // ge ge ge: ->10->4-> through 'hello' to start(0)
		{11, 1, 10}, // ge from space: skip -> 'd'(10)
	}
	for _, c := range cases {
		got := PrevWordEnd(text, c.pos, c.count)
		if got != c.want {
			t.Errorf("PrevWordEnd(pos=%d, count=%d) = %d, want %d", c.pos, c.count, got, c.want)
		}
	}
}

// --- Bigword motions ---

func TestNextBigwordStart(t *testing.T) {
	// "hello.world foo": 'hello.world' is one WORD (punctuation doesn't break)
	text := []rune("hello.world foo")
	cases := []struct{ pos, want int }{
		{0, 12}, // W from 'h': through entire WORD -> skip space -> 'f'
		{5, 12}, // W from '.': same
	}
	for _, c := range cases {
		got := NextBigwordStart(text, c.pos, 1)
		if got != c.want {
			t.Errorf("NextBigwordStart(pos=%d) = %d, want %d", c.pos, got, c.want)
		}
	}
}

func TestPrevBigwordStart(t *testing.T) {
	text := []rune("hello.world foo")
	// B from 14('o' in 'foo'): back through 'foo'(14..12) -> start of 'foo' at 12
	got := PrevBigwordStart(text, 14, 1)
	if got != 12 {
		t.Errorf("PrevBigwordStart(pos=14) = %d, want 12", got)
	}
}

func TestNextBigwordEnd(t *testing.T) {
	text := []rune("hello.world foo")
	if got := NextBigwordEnd(text, 0, 1); got != 10 {
		t.Errorf("NextBigwordEnd(pos=0) = %d, want 10 ('d' at end of WORD)", got)
	}
	if got := NextBigwordEnd(text, 12, 1); got != 14 {
		t.Errorf("NextBigwordEnd(pos=12) = %d, want 14 ('o' at end of 'foo')", got)
	}
}

func TestPrevBigwordEnd(t *testing.T) {
	text := []rune("hello.world foo")
	got := PrevBigwordEnd(text, 14, 1)
	if got != 10 {
		t.Errorf("PrevBigwordEnd(pos=14) = %d, want 10", got)
	}
}

// --- Line motions ---

func TestLineStart(t *testing.T) {
	// h(0)e(1)l(2)l(3)o(4)\n(5)w(6)o(7)r(8)l(9)d(10)
	text := []rune("hello\nworld")
	cases := []struct{ pos, want int }{
		{0, 0},
		{3, 0}, // middle "hello" -> start 0
		{5, 6}, // \n: this \n ENDS "hello", next line starts at 6
		{6, 6}, // first of "world" -> 6
		{8, 6}, // middle "world" -> 6
	}
	for _, c := range cases {
		got := LineStart(text, c.pos)
		if got != c.want {
			t.Errorf("LineStart(pos=%d) = %d, want %d", c.pos, got, c.want)
		}
	}
}

func TestLineEnd(t *testing.T) {
	text := []rune("hello\nworld")
	cases := []struct{ pos, want int }{
		{0, 4},   // $ from 'h': end of "hello" -> 'o'(4)
		{3, 4},   // $ from middle -> 'o'(4)
		{5, 4},   // $ from \n: end of "hello" -> 'o'(4)
		{6, 10},  // $ from 'w': end -> 'd'(10)
		{8, 10},  // $ from middle -> 'd'(10)
		{10, 10}, // $ from last -> itself
	}
	for _, c := range cases {
		got := LineEnd(text, c.pos)
		if got != c.want {
			t.Errorf("LineEnd(pos=%d) = %d, want %d", c.pos, got, c.want)
		}
	}
}

func TestLineEnd_emptyLine(t *testing.T) {
	// h(0)e(1)l(2)l(3)o(4)\n(5)\n(6)w(7)o(8)r(9)l(10)d(11)
	// Line 2 (between \n at 5 and \n at 6) is empty (0 chars).
	text := []rune("hello\n\nworld")
	cases := []struct{ pos, want int }{
		{5, 4},  // $ from \n after "hello": end of line 1 -> 'o'(4)
		{6, 6},  // $ from \n on empty line: stay on \n (0 chars)
		{7, 11}, // $ from 'w': end -> 'd'(11)
	}
	for _, c := range cases {
		got := LineEnd(text, c.pos)
		if got != c.want {
			t.Errorf("LineEnd(pos=%d) = %d, want %d", c.pos, got, c.want)
		}
	}
}

func TestFirstNonBlank(t *testing.T) {
	// "  hello\n  world"
	// Line1: (0) (1)h(2)e(3)l(4)l(5)o(6)\n(7)
	// Line2: (8) (9)w(10)o(11)r(12)l(13)d(14)
	text := []rune("  hello\n  world")
	cases := []struct{ pos, want int }{
		{0, 2},  // ^ from start: skip 2 spaces -> 'h'(2)
		{3, 2},  // ^ from 'e': go BACK to first non-blank -> 'h'(2)
		{7, 10}, // ^ from \n: end of line1, go to first non-blank of... hmm
		// Actually LineStart(7) on \n after "hello": walks back to 0, then skip spaces to 'h'(2).
		// But conceptually from \n, maybe user expects next line? Let's test what we get.
		{9, 10}, // ^ from ' ': skip -> 'w'(10)
	}
	for _, c := range cases {
		got := FirstNonBlank(text, c.pos)
		if got != c.want {
			t.Errorf("FirstNonBlank(text=%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

func TestFirstNonBlank_emptyLine(t *testing.T) {
	text := []rune("hello\n   \nworld")
	got := FirstNonBlank(text, 6) // pos 6 is space on the whitespace-only line
	if got != 6 {                 // no non-blank -> line start
		t.Errorf("FirstNonBlank from whitespace line = %d, want 6", got)
	}
}

func TestLastNonBlank(t *testing.T) {
	text := []rune("hello  \nworld  ")
	cases := []struct{ pos, want int }{
		{0, 4},  // g_ from 'h': last non-blank -> 'o'(4)
		{3, 4},  // from 'l': -> 'o'(4)
		{7, 12}, // g_ from 'w': last non-blank on line 2 -> 'd'(12)
		{8, 12}, // from 'o': -> 'd'(12)
	}
	for _, c := range cases {
		got := LastNonBlank(text, c.pos)
		if got != c.want {
			t.Errorf("LastNonBlank(text=%q, pos=%d) = %d, want %d", string(text), c.pos, got, c.want)
		}
	}
}

// --- Character search ---

func TestFindNext(t *testing.T) {
	text := []rune("hello world")
	cases := []struct {
		pos  int
		char rune
		want int
	}{
		{0, 'l', 2},
		{0, 'o', 4},
		{0, 'x', -1},
		{3, 'l', 9}, // from 'l'(3): next 'l' is at 9 (skips current)
	}
	for _, c := range cases {
		got := FindNext(text, c.pos, c.char)
		if got != c.want {
			t.Errorf("FindNext(pos=%d, char=%q) = %d, want %d", c.pos, c.char, got, c.want)
		}
	}
}

func TestFindPrev(t *testing.T) {
	text := []rune("hello world")
	if got := FindPrev(text, 4, 'l'); got != 3 {
		t.Errorf("FindPrev pos=4 'l' = %d, want 3", got)
	}
	if got := FindPrev(text, 4, 'h'); got != 0 {
		t.Errorf("FindPrev pos=4 'h' = %d, want 0", got)
	}
	if got := FindPrev(text, 4, 'x'); got != -1 {
		t.Errorf("FindPrev pos=4 'x' = %d, want -1", got)
	}
}

func TestTillNext(t *testing.T) {
	text := []rune("hello world")
	if got := TillNext(text, 0, 'l'); got != 1 {
		t.Errorf("TillNext pos=0 'l' = %d, want 1", got)
	}
	if got := TillNext(text, 0, 'x'); got != -1 {
		t.Errorf("TillNext pos=0 'x' = %d, want -1", got)
	}
}

func TestTillPrev(t *testing.T) {
	text := []rune("hello world")
	if got := TillPrev(text, 4, 'h'); got != 1 {
		t.Errorf("TillPrev pos=4 'h' = %d, want 1", got)
	}
	if got := TillPrev(text, 4, 'x'); got != -1 {
		t.Errorf("TillPrev pos=4 'x' = %d, want -1", got)
	}
}

func TestFindNext_staysOnLine(t *testing.T) {
	text := []rune("hello\nworld")
	if got := FindNext(text, 0, 'w'); got != -1 {
		t.Errorf("f across line: got %d, want -1", got)
	}
}

// --- Bracket matching ---

func TestMatchBracket_basic(t *testing.T) {
	cases := []struct {
		text string
		pos  int
		want int
	}{
		{"(hello)", 0, 6},
		{"(hello)", 6, 0},
		{"{hello}", 0, 6},
		{"[hello]", 0, 6},
		{"hello(world)", 5, 11},
	}
	for _, c := range cases {
		got := MatchBracket([]rune(c.text), c.pos)
		if got != c.want {
			t.Errorf("MatchBracket(%q, pos=%d) = %d, want %d", c.text, c.pos, got, c.want)
		}
	}
}

func TestMatchBracket_nested(t *testing.T) {
	// a(0)((1)b(2)((3)c(4))(5)d(6))(7)
	text := []rune("a(b(c)d)")
	if got := MatchBracket(text, 1); got != 7 {
		t.Errorf("MatchBracket outer '(' at 1: got %d, want 7", got)
	}
	if got := MatchBracket(text, 3); got != 5 {
		t.Errorf("MatchBracket inner '(' at 3: got %d, want 5", got)
	}
}

func TestMatchBracket_noMatch(t *testing.T) {
	if got := MatchBracket([]rune("hello world"), 0); got != -1 {
		t.Errorf("got %d, want -1", got)
	}
	if got := MatchBracket([]rune("("), 0); got != -1 {
		t.Errorf("unmatched: got %d, want -1", got)
	}
}

// --- Edge cases ---

func TestEmptyText(t *testing.T) {
	if got := NextWordStart(nil, 0, 1); got != 0 {
		t.Errorf("NextWordStart nil: %d", got)
	}
	if got := PrevWordStart(nil, 0, 1); got != 0 {
		t.Errorf("PrevWordStart nil: %d", got)
	}
	if got := NextWordEnd(nil, 0, 1); got != 0 {
		t.Errorf("NextWordEnd nil: %d", got)
	}
	if got := PrevWordEnd(nil, 0, 1); got != 0 {
		t.Errorf("PrevWordEnd nil: %d", got)
	}
	if got := LineEnd(nil, 0); got != 0 {
		t.Errorf("LineEnd nil: %d", got)
	}
	if got := FindNext(nil, 0, 'x'); got != -1 {
		t.Errorf("FindNext nil: %d", got)
	}
	if got := MatchBracket(nil, 0); got != -1 {
		t.Errorf("MatchBracket nil: %d", got)
	}
}

func TestSingleChar(t *testing.T) {
	text := []rune("a")
	tests := []func([]rune, int, int) int{
		NextWordStart, PrevWordStart, NextWordEnd, PrevWordEnd,
	}
	for _, fn := range tests {
		if got := fn(text, 0, 1); got != 0 {
			t.Errorf("%T from single char = %d, want 0", fn, got)
		}
	}
}
