package motion

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Buffer motions
// ---------------------------------------------------------------------------

func TestMoveToLine(t *testing.T) {
	text := []rune("hello\nworld\nfoo")
	cases := []struct {
		count int
		want  int
	}{
		{0, 0},  // default → line 1
		{1, 0},  // line 1
		{2, 6},  // line 2 (after first \n)
		{3, 12}, // line 3
		{4, 12}, // past end → last line
	}
	for _, c := range cases {
		got := MoveToLine(text, c.count)
		if got != c.want {
			t.Errorf("MoveToLine(count=%d) = %d, want %d", c.count, got, c.want)
		}
	}
}

func TestMoveToEndOfBuffer(t *testing.T) {
	if got := MoveToEndOfBuffer([]rune("hello")); got != 4 {
		t.Errorf("MoveToEndOfBuffer = %d, want 4", got)
	}
	if got := MoveToEndOfBuffer([]rune("")); got != 0 {
		t.Errorf("MoveToEndOfBuffer empty = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// Paragraph motions
// ---------------------------------------------------------------------------

func TestMoveParagraphForward(t *testing.T) {
	// "abc\n\n123\n456\n789\n\ndef"
	// Para 1: 0-2  "abc"
	// Blank:  3-4  "\n\n"
	// Para 2: 5-14 "123\n456\n789"
	// Blank:  15-16 "\n\n"
	// Para 3: 17-19 "def"
	// "abc\n\n123\n456\n789\n\ndef"
	// a(0)b(1)c(2)\n(3)\n(4)1(5)2(6)3(7)\n(8)4(9)5(10)6(11)\n(12)7(13)8(14)9(15)\n(16)\n(17)d(18)e(19)f(20)
	text := []rune("abc\n\n123\n456\n789\n\ndef")
	cases := []struct {
		pos   int
		count int
		want  int
	}{
		{0, 1, 4},   // } from "a": empty line before '1'
		{2, 1, 4},   // } from "c": empty line before '1'
		{5, 1, 17},  // } from "1": empty line before 'd'
		{18, 1, 21}, // } from "d": past end of text
		{0, 2, 17},  // }} from "a": empty line before 'd'
		{3, 1, 4},   // } from first \n of blank line: stops at empty line
	}
	for _, c := range cases {
		got := MoveParagraphForward(text, c.pos, c.count)
		if got != c.want {
			t.Errorf("MoveParagraphForward(pos=%d, count=%d) = %d, want %d", c.pos, c.count, got, c.want)
		}
	}
}

func TestMoveParagraphBackward(t *testing.T) {
	text := []rune("abc\n\n123\n456\n789\n\ndef")
	cases := []struct {
		pos   int
		count int
		want  int
	}{
		{5, 1, 4},  // { from "1": empty line before '1'
		{17, 1, 4}, // { from "d": empty line before '1'
		{8, 1, 4},  // { from "5": empty line before '1'
		{10, 2, 0}, // {{ from "7": para 1 start
		{0, 1, 0},  // { from start: no-op
	}
	for _, c := range cases {
		got := MoveParagraphBackward(text, c.pos, c.count)
		if got != c.want {
			t.Errorf("MoveParagraphBackward(pos=%d, count=%d) = %d, want %d", c.pos, c.count, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Text objects
// ---------------------------------------------------------------------------

func TestWordInner(t *testing.T) {
	text := []rune("hello world foo")
	cases := []struct {
		pos  int
		want Range
	}{
		{0, Range{0, 5}},    // iw from 'h': "hello"
		{2, Range{0, 5}},    // iw from middle: "hello"
		{6, Range{6, 11}},   // iw from 'w': "world"
		{11, Range{12, 15}}, // iw from space: "foo" (forward from whitespace)
	}
	for _, c := range cases {
		got := WordInner(text, c.pos)
		if got != c.want {
			t.Errorf("WordInner(pos=%d) = {Start=%d,End=%d}, want {Start=%d,End=%d}",
				c.pos, got.Start, got.End, c.want.Start, c.want.End)
		}
	}
}

func TestWordA(t *testing.T) {
	text := []rune("hello world foo")
	cases := []struct {
		pos  int
		want Range
	}{
		{0, Range{0, 6}},  // aw from 'h': "hello "
		{2, Range{0, 6}},  // aw from middle: "hello "
		{6, Range{6, 12}}, // aw from 'w': "world "
	}
	for _, c := range cases {
		got := WordA(text, c.pos)
		if got != c.want {
			t.Errorf("WordA(pos=%d) = {Start=%d,End=%d}, want {Start=%d,End=%d}",
				c.pos, got.Start, got.End, c.want.Start, c.want.End)
		}
	}
}

func TestQuoteInner(t *testing.T) {
	text := []rune(`hello "world" foo`)
	cases := []struct {
		pos  int
		want Range
	}{
		{8, Range{7, 12}}, // i" from 'o': content is "world"
		{6, Range{7, 12}}, // i" from opening quote: content
	}
	for _, c := range cases {
		got := QuoteInner(text, c.pos, '"')
		if got != c.want {
			t.Errorf("QuoteInner(pos=%d) = {Start=%d,End=%d}, want {Start=%d,End=%d}",
				c.pos, got.Start, got.End, c.want.Start, c.want.End)
		}
	}
}

func TestQuoteA(t *testing.T) {
	text := []rune(`hello "world" foo`)
	cases := []struct {
		pos  int
		want Range
	}{
		{8, Range{6, 13}}, // a" from 'o': "world" plus quotes (exclusive end)
	}
	for _, c := range cases {
		got := QuoteA(text, c.pos, '"')
		if got != c.want {
			t.Errorf("QuoteA(pos=%d) = {Start=%d,End=%d}, want {Start=%d,End=%d}",
				c.pos, got.Start, got.End, c.want.Start, c.want.End)
		}
	}
}

func TestPairInner(t *testing.T) {
	text := []rune("foo(bar(baz))qux")
	cases := []struct {
		pos  int
		want Range
	}{
		{4, Range{4, 12}}, // i( from 'b' in "bar(baz)": outer content "bar(baz)"
		{8, Range{8, 11}}, // i( from 'a' in "baz": inner content "baz"
		{3, Range{4, 12}}, // i( from '(': outer content "bar(baz)"
		{0, Range{0, 0}},  // outside any pair
	}
	for _, c := range cases {
		got := PairInner(text, c.pos, '(', ')')
		if got != c.want {
			t.Errorf("PairInner(pos=%d) = {Start=%d,End=%d}, want {Start=%d,End=%d}",
				c.pos, got.Start, got.End, c.want.Start, c.want.End)
		}
	}
}

func TestPairA(t *testing.T) {
	text := []rune("foo(bar)qux")
	got := PairA(text, 4, '(', ')')
	want := Range{3, 8} // "(bar)" including parens
	if got != want {
		t.Errorf("PairA = {Start=%d,End=%d}, want {Start=%d,End=%d}", got.Start, got.End, want.Start, want.End)
	}
}

func TestParagraphInner(t *testing.T) {
	text := []rune("abc\n\n123\n456\n\n789")
	got := ParagraphInner(text, 8) // inside para 2
	want := Range{5, 12}           // "123\n456"
	if got != want {
		t.Errorf("ParagraphInner = {Start=%d,End=%d}, want {Start=%d,End=%d}", got.Start, got.End, want.Start, want.End)
	}
}

func TestParagraphA(t *testing.T) {
	text := []rune("abc\n\n123\n456\n\n789")
	got := ParagraphA(text, 8)
	want := Range{3, 14} // includes surrounding blank lines
	if got != want {
		t.Errorf("ParagraphA = {Start=%d,End=%d}, want {Start=%d,End=%d}", got.Start, got.End, want.Start, want.End)
	}
}

// ---------------------------------------------------------------------------
// Operators
// ---------------------------------------------------------------------------

func TestApplyOp(t *testing.T) {
	cases := []struct {
		name   string
		op     Op
		text   string
		cursor int
		r      Range
		want   string
		newCur int
		insert bool
	}{
		{
			name:   "delete range",
			op:     OpDelete,
			text:   "hello world",
			cursor: 0,
			r:      Range{0, 5},
			want:   " world",
			newCur: 0,
		},
		{
			name:   "change range",
			op:     OpChange,
			text:   "hello world",
			cursor: 0,
			r:      Range{0, 5},
			want:   " world",
			newCur: 0,
			insert: true,
		},
		{
			name:   "yank range",
			op:     OpYank,
			text:   "hello world",
			cursor: 0,
			r:      Range{0, 5},
			want:   "hello world",
			newCur: 0,
		},
		{
			name:   "empty range",
			op:     OpDelete,
			text:   "hello",
			cursor: 0,
			r:      Range{0, 0},
			want:   "hello",
			newCur: 0,
		},
	}
	for _, c := range cases {
		result := ApplyOp(c.op, []rune(c.text), c.cursor, c.r)
		if string(result.Text) != c.want {
			t.Errorf("%s: got text %q, want %q", c.name, string(result.Text), c.want)
		}
		if result.Cursor != c.newCur {
			t.Errorf("%s: got cursor %d, want %d", c.name, result.Cursor, c.newCur)
		}
		if result.Insert != c.insert {
			t.Errorf("%s: got insert=%v, want %v", c.name, result.Insert, c.insert)
		}
		if c.op == OpYank && result.Yanked != "hello" {
			t.Errorf("%s: got yanked %q, want %q", c.name, result.Yanked, "hello")
		}
	}
}

// ---------------------------------------------------------------------------
// Engine: basic navigation
// ---------------------------------------------------------------------------

func TestEngineWordNav(t *testing.T) {
	var e Engine
	text := []rune("hello world foo")

	// w: forward one word
	e.Reset()
	result := e.Process(text, 0, Key('w'))
	if result.Kind != ResultNavigate || result.Cursor != 6 {
		t.Errorf("w: got Kind=%d Cursor=%d, want Navigate(6)", result.Kind, result.Cursor)
	}

	// b: backward one word
	e.Reset()
	result = e.Process(text, 6, Key('b'))
	if result.Kind != ResultNavigate || result.Cursor != 0 {
		t.Errorf("b from 6: got Kind=%d Cursor=%d, want Navigate(0)", result.Kind, result.Cursor)
	}

	// e: end of word
	e.Reset()
	result = e.Process(text, 0, Key('e'))
	if result.Kind != ResultNavigate || result.Cursor != 4 {
		t.Errorf("e: got Kind=%d Cursor=%d, want Navigate(4)", result.Kind, result.Cursor)
	}
}

func TestEngineLineNav(t *testing.T) {
	var e Engine
	text := []rune("hello\nworld")
	cases := []struct {
		key  rune
		pos  int
		want int
	}{
		{'0', 7, 6},  // line start from 'r' in "world"
		{'$', 6, 10}, // line end from 'w'
		{'^', 7, 6},  // first non-blank from 'r'
	}
	for _, c := range cases {
		e.Reset()
		result := e.Process(text, c.pos, Key(c.key))
		if result.Kind != ResultNavigate || result.Cursor != c.want {
			t.Errorf("key=%c pos=%d: got Kind=%d Cursor=%d, want Navigate(%d)",
				c.key, c.pos, result.Kind, result.Cursor, c.want)
		}
	}
}

func TestEngineDD(t *testing.T) {
	var e Engine
	text := []rune("hello\nworld\nfoo")
	// dd at cursor 1 (inside "hello")
	result1 := e.Process(text, 1, Key('d'))
	if result1.Kind != ResultNone {
		t.Fatal("d should return ResultNone")
	}
	result2 := e.Process(text, 1, Key('d'))
	if result2.Kind != ResultExecute {
		t.Fatalf("dd: got Kind=%d, want ResultExecute", result2.Kind)
	}
	if result2.Op != OpDelete {
		t.Errorf("dd: got Op=%d, want OpDelete", result2.Op)
	}
	// Should delete "hello\n" = positions 0-5
	if result2.Range.Start != 0 || result2.Range.End != 6 {
		t.Errorf("dd: got Range=[%d,%d), want [0,6)", result2.Range.Start, result2.Range.End)
	}
}

func TestEngineDW(t *testing.T) {
	var e Engine
	text := []rune("hello world")
	// dw from position 0
	e.Process(text, 0, Key('d')) // pending delete
	result := e.Process(text, 0, Key('w'))
	if result.Kind != ResultExecute {
		t.Fatalf("dw: got Kind=%d, want ResultExecute", result.Kind)
	}
	if result.Op != OpDelete {
		t.Errorf("dw: got Op=%d, want OpDelete", result.Op)
	}
	if result.Range.Start != 0 || result.Range.End != 6 {
		t.Errorf("dw: got Range=[%d,%d), want [0,6)", result.Range.Start, result.Range.End)
	}
}

func TestEngineDIQuote(t *testing.T) {
	var e Engine
	text := []rune(`hello "world" foo`)
	// ci" from position 8 (the 'o' inside "world")
	e.Process(text, 8, Key('c')) // pending change
	e.Process(text, 8, Key('i')) // inner text object
	result := e.Process(text, 8, Key('"'))
	if result.Kind != ResultExecute {
		t.Fatalf("ci\": got Kind=%d, want ResultExecute", result.Kind)
	}
	if result.Op != OpChange {
		t.Errorf("ci\": got Op=%d, want OpChange", result.Op)
	}
	if result.Insert != true {
		t.Errorf("ci\": got Insert=false, want true")
	}
	// Range should be "world" (inner quotes)
	if result.Range.Start != 7 || result.Range.End != 12 {
		t.Errorf("ci\": got Range=[%d,%d), want [7,12)", result.Range.Start, result.Range.End)
	}
}

func TestEngineEscapeCancels(t *testing.T) {
	var e Engine
	text := []rune("hello")
	// Start an operator, then cancel
	e.Process(text, 0, Key('d'))
	result := e.Process(text, 0, Key(27))
	if result.Kind != ResultCancel {
		t.Errorf("esc: got Kind=%d, want ResultCancel", result.Kind)
	}
}

func TestEngineIgnoresInsertKeys(t *testing.T) {
	var e Engine
	text := []rune("hello")
	// Keys like 'i' should not be consumed (return ResultNone)
	result := e.Process(text, 0, Key('i'))
	if result.Kind != ResultNone {
		t.Errorf("i in idle: got Kind=%d, want ResultNone", result.Kind)
	}
	// After processing 'i', the engine should still be idle
	if e.state != stIdle {
		t.Errorf("after i: state=%d, want stIdle", e.state)
	}
}

func TestEngineCountWithMotion(t *testing.T) {
	var e Engine
	text := []rune("hello world foo bar")
	// 2w: forward 2 words from start
	e.Process(text, 0, Key('2'))
	result := e.Process(text, 0, Key('w'))
	if result.Kind != ResultNavigate || result.Cursor != 12 {
		t.Errorf("2w: got Kind=%d Cursor=%d, want Navigate(%d)", result.Kind, result.Cursor, 12)
	}
}

func TestEngineCountWithOperator(t *testing.T) {
	var e Engine
	text := []rune("one two three four")
	// d2w from position 0: delete 2 words
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('2'))
	result := e.Process(text, 0, Key('w'))
	if result.Kind != ResultExecute {
		t.Fatalf("d2w: got Kind=%d, want ResultExecute", result.Kind)
	}
	if result.Op != OpDelete {
		t.Errorf("d2w: got Op=%d, want OpDelete", result.Op)
	}
	if result.Range.Start != 0 || result.Range.End != 8 {
		t.Errorf("d2w: got Range=[%d,%d), want [0,8)", result.Range.Start, result.Range.End)
	}
}

func TestEngineGG(t *testing.T) {
	var e Engine
	text := []rune("line1\nline2\nline3")
	// gg from line 2
	e.Process(text, 8, Key('g'))
	result := e.Process(text, 8, Key('g'))
	if result.Kind != ResultNavigate || result.Cursor != 0 {
		t.Errorf("gg: got Kind=%d Cursor=%d, want Navigate(%d)", result.Kind, result.Cursor, 0)
	}
}

func TestEngineCharSearch(t *testing.T) {
	var e Engine
	text := []rune("hello world")
	result := e.Process(text, 0, Key('f'))
	if result.Kind != ResultNone {
		t.Fatal("f should return ResultNone")
	}
	result = e.Process(text, 0, Key('l'))
	if result.Kind != ResultNavigate || result.Cursor != 2 {
		t.Errorf("fl: got Kind=%d Cursor=%d, want Navigate(%d)", result.Kind, result.Cursor, 2)
	}
}

func TestEngineCharSearchWithOperator(t *testing.T) {
	var e Engine
	text := []rune("hello world")
	// dfl from position 0: delete up to and including 'l' (at position 2)
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('f'))
	result := e.Process(text, 0, Key('l'))
	if result.Kind != ResultExecute {
		t.Fatalf("dfl: got Kind=%d, want ResultExecute", result.Kind)
	}
	if result.Op != OpDelete {
		t.Errorf("dfl: got Op=%d, want OpDelete", result.Op)
	}
	if result.Range.Start != 0 || result.Range.End != 3 {
		t.Errorf("dfl: got Range=[%d,%d), want [0,3)", result.Range.Start, result.Range.End)
	}
}

func TestEngineParenTextObj(t *testing.T) {
	var e Engine
	text := []rune("foo(bar)baz")
	// di( from position 5 (the 'a')
	e.Process(text, 5, Key('d'))
	e.Process(text, 5, Key('i'))
	result := e.Process(text, 5, Key('('))
	if result.Kind != ResultExecute {
		t.Fatalf("di(: got Kind=%d, want ResultExecute", result.Kind)
	}
	if result.Range.Start != 4 || result.Range.End != 7 {
		t.Errorf("di(: got Range=[%d,%d), want [4,7)", result.Range.Start, result.Range.End)
	}
}

func TestEngineVerticalMotion(t *testing.T) {
	var e Engine
	text := []rune("hello\nworld\nfoo")
	// j from 'l' (pos 3): go down one line
	result := e.Process(text, 3, Key('j'))
	if result.Kind != ResultNavigate {
		t.Fatalf("j: got Kind=%d, want ResultNavigate", result.Kind)
	}
	// j from column 3 of "hello" goes to column 3 of "world"
	if result.Cursor != 9 {
		t.Errorf("j from pos=3: got %d, want 9", result.Cursor)
	}
}

func TestEngineCharRepeat(t *testing.T) {
	var e Engine
	text := []rune("a b a b a")
	// fa from 0: find 'a' at position 4 (skip "a b ")
	e.Process(text, 0, Key('f'))
	r := e.Process(text, 0, Key('a'))
	if r.Kind != ResultNavigate || r.Cursor != 4 {
		t.Fatalf("fa: got Cursor=%d, want 4", r.Cursor)
	}
	// ; from 4: repeat f, find next 'a' at 8
	e.Reset()
	e.Process(text, 4, Key(';'))
	r = e.Process(text, 4, Key(';'))
	if r.Kind != ResultNavigate || r.Cursor != 8 {
		t.Fatalf(";;: got Cursor=%d, want 8", r.Cursor)
	}
	// , from 8: reverse direction, find previous 'a' at 4
	e.Reset()
	e.Process(text, 8, Key(','))
	r = e.Process(text, 8, Key(','))
	if r.Kind != ResultNavigate || r.Cursor != 4 {
		t.Fatalf(",,: got Cursor=%d, want 4", r.Cursor)
	}
}

func TestEngineCharSearchPartialCount(t *testing.T) {
	var e Engine
	text := []rune("a x b")
	// 2fx from 0: 'x' appears once, extra count stays on it (doesn't reset)
	e.Process(text, 0, Key('2'))
	e.Process(text, 0, Key('f'))
	r := e.Process(text, 0, Key('x'))
	if r.Kind != ResultNavigate || r.Cursor != 2 {
		t.Errorf("2fx with 1 x: got Cursor=%d, want 2", r.Cursor)
	}
}

func TestEngineCharSearchTill(t *testing.T) {
	var e Engine
	text := []rune("hello world")
	// tl from 0: land before 'l' at position 1
	e.Process(text, 0, Key('t'))
	r := e.Process(text, 0, Key('l'))
	if r.Kind != ResultNavigate || r.Cursor != 1 {
		t.Errorf("tl: got Cursor=%d, want 1", r.Cursor)
	}
	// Th from 10: land after 'h' at position 1
	e.Reset()
	e.Process(text, 10, Key('T'))
	r = e.Process(text, 10, Key('h'))
	if r.Kind != ResultNavigate || r.Cursor != 1 {
		t.Errorf("Th: got Cursor=%d, want 1", r.Cursor)
	}
	// dtl with operator: delete up to 'l' (exclusive)
	e.Reset()
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('t'))
	r = e.Process(text, 0, Key('l'))
	if r.Kind != ResultExecute {
		t.Fatalf("dtl: got Kind=%d, want Execute", r.Kind)
	}
	if r.Range != (Range{0, 2}) {
		t.Errorf("dtl: got Range=[%d,%d), want [0,2)", r.Range.Start, r.Range.End)
	}
}

func TestEngineG(t *testing.T) {
	var e Engine
	text := []rune("line1\nline2\nline3")
	// G from any position: go to end of last line
	r := e.Process(text, 0, Key('G'))
	if r.Kind != ResultNavigate || r.Cursor != 16 {
		t.Errorf("G: got Cursor=%d, want 16", r.Cursor)
	}
	// 2G: go to line 2
	e.Reset()
	e.Process(text, 0, Key('2'))
	r = e.Process(text, 0, Key('G'))
	if r.Kind != ResultNavigate || r.Cursor != 6 {
		t.Errorf("2G: got Cursor=%d, want 6", r.Cursor)
	}
	// 3gg: go to line 3
	e.Reset()
	e.Process(text, 0, Key('3'))
	e.Process(text, 0, Key('g'))
	r = e.Process(text, 0, Key('g'))
	if r.Kind != ResultNavigate || r.Cursor != 12 {
		t.Errorf("3gg: got Cursor=%d, want 12", r.Cursor)
	}
}

func TestEngineGPrefixMotion(t *testing.T) {
	var e Engine
	text := []rune("hello world foo")
	// ge from 'o'(14): backward to end of prev word 'd'(10)
	e.Process(text, 14, Key('g'))
	r := e.Process(text, 14, Key('e'))
	if r.Kind != ResultNavigate || r.Cursor != 10 {
		t.Errorf("ge: got Cursor=%d, want 10", r.Cursor)
	}
	// g_ from 'h'(0): last non-blank on single line = 'o'(14)
	e.Reset()
	e.Process(text, 0, Key('g'))
	r = e.Process(text, 0, Key('_'))
	if r.Kind != ResultNavigate || r.Cursor != 14 {
		t.Errorf("g_: got Cursor=%d, want 14", r.Cursor)
	}
	// dg_ from 0: delete cursor to last non-blank = entire line
	e.Reset()
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('g'))
	r = e.Process(text, 0, Key('_'))
	if r.Kind != ResultExecute {
		t.Fatalf("dg_: got Kind=%d, want Execute", r.Kind)
	}
	if r.Range.Start != 0 || r.Range.End != 15 {
		t.Errorf("dg_: got Range=[%d,%d), want [0,15)", r.Range.Start, r.Range.End)
	}
	// gE from 'o'(14): backward to end of prev BIGWORD = 'd'(10)
	e.Reset()
	e.Process(text, 14, Key('g'))
	r = e.Process(text, 14, Key('E'))
	if r.Kind != ResultNavigate || r.Cursor != 10 {
		t.Errorf("gE: got Cursor=%d, want 10", r.Cursor)
	}
	// g_ on multi-line text
	e.Reset()
	text = []rune("hello\nworld")
	e.Process(text, 0, Key('g'))
	r = e.Process(text, 0, Key('_'))
	if r.Kind != ResultNavigate || r.Cursor != 4 {
		t.Errorf("g_ multi-line: got Cursor=%d, want 4", r.Cursor)
	}
}

func TestEnginePercentMotion(t *testing.T) {
	var e Engine
	text := []rune("foo(bar)baz")
	// % from 0: find '(' at 3, match ')' at 7
	r := e.Process(text, 0, Key('%'))
	if r.Kind != ResultNavigate || r.Cursor != 7 {
		t.Errorf("%%: got Cursor=%d, want 7", r.Cursor)
	}
	// d% from 3 (on '('): delete from '(' to matching ')' inclusive
	e.Reset()
	e.Process(text, 3, Key('d'))
	r = e.Process(text, 3, Key('%'))
	if r.Kind != ResultExecute {
		t.Fatalf("d%%: got Kind=%d, want Execute", r.Kind)
	}
	if r.Range.Start != 3 || r.Range.End != 8 {
		t.Errorf("d%%: got Range=[%d,%d), want [3,8)", r.Range.Start, r.Range.End)
	}
}

func TestEngineLineMotionOperators(t *testing.T) {
	var e Engine
	text := []rune("hello world")
	// d0 from 5 (the space): delete from line start to cursor
	e.Process(text, 5, Key('d'))
	r := e.Process(text, 5, Key('0'))
	if r.Kind != ResultExecute {
		t.Fatalf("d0: got Kind=%d", r.Kind)
	}
	if r.Range != (Range{0, 5}) {
		t.Errorf("d0: got Range=[%d,%d), want [0,5)", r.Range.Start, r.Range.End)
	}
	// d$ from 0: delete from cursor to end of line
	e.Reset()
	e.Process(text, 0, Key('d'))
	r = e.Process(text, 0, Key('$'))
	if r.Kind != ResultExecute {
		t.Fatalf("d$: got Kind=%d", r.Kind)
	}
	if r.Range != (Range{0, 11}) {
		t.Errorf("d$: got Range=[%d,%d), want [0,11)", r.Range.Start, r.Range.End)
	}
}

func TestEngineParagraphOperator(t *testing.T) {
	var e Engine
	text := []rune("abc\n\n123\n456\n\n789")
	// d} from 0 (para 1): delete through the end of para 1 (blank line boundary)
	e.Process(text, 0, Key('d'))
	r := e.Process(text, 0, Key('}'))
	if r.Kind != ResultExecute {
		t.Fatalf("d}: got Kind=%d, want Execute", r.Kind)
	}
	if r.Range.Start != 0 || r.Range.End < 3 {
		t.Errorf("d}: got Range=[%d,%d), want start=0 end>=3", r.Range.Start, r.Range.End)
	}
	// d{ from 10 (middle of para 2): delete from para start to cursor
	e.Reset()
	e.Process(text, 10, Key('d'))
	r = e.Process(text, 10, Key('{'))
	if r.Kind != ResultExecute {
		t.Fatalf("d{: got Kind=%d, want Execute", r.Kind)
	}
	if r.Range.Start != 4 || r.Range.End != 10 {
		t.Errorf("d{: got Range=[%d,%d), want [4,10)", r.Range.Start, r.Range.End)
	}
}

func TestEngineWordTextObjOperator(t *testing.T) {
	var e Engine
	text := []rune("hello world foo")
	// diw from 0: delete inner word "hello"
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('i'))
	r := e.Process(text, 0, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatalf("diw: got Kind=%d", r.Kind)
	}
	if r.Range != (Range{0, 5}) {
		t.Errorf("diw: got Range=[%d,%d), want [0,5)", r.Range.Start, r.Range.End)
	}
	// daw from 6: delete a word "world "
	e.Reset()
	e.Process(text, 6, Key('d'))
	e.Process(text, 6, Key('a'))
	r = e.Process(text, 6, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatalf("daw: got Kind=%d", r.Kind)
	}
	if r.Range != (Range{6, 12}) {
		t.Errorf("daw: got Range=[%d,%d), want [6,12)", r.Range.Start, r.Range.End)
	}
}

func TestEngineTagTextObj(t *testing.T) {
	var e Engine
	text := []rune("<div>hello</div>")
	// dit from 6 (inside tag content): select inner "hello"
	e.Process(text, 6, Key('d'))
	e.Process(text, 6, Key('i'))
	r := e.Process(text, 6, Key('t'))
	if r.Kind != ResultExecute {
		t.Fatalf("dit: got Kind=%d", r.Kind)
	}
	if r.Range.Start != 5 || r.Range.End != 10 {
		t.Errorf("dit: got Range=[%d,%d), want [5,10)", r.Range.Start, r.Range.End)
	}
	// dat from 6: delete whole tag pair including content
	e.Reset()
	e.Process(text, 6, Key('d'))
	e.Process(text, 6, Key('a'))
	r = e.Process(text, 6, Key('t'))
	if r.Kind != ResultExecute {
		t.Fatalf("dat: got Kind=%d", r.Kind)
	}
	if r.Range.Start != 0 || r.Range.End != 16 {
		t.Errorf("dat: got Range=[%d,%d), want [0,16)", r.Range.Start, r.Range.End)
	}
}

func TestEngineParagraphTextObj(t *testing.T) {
	var e Engine
	text := []rune("abc\n\n123\n456\n\n789")
	// dip from 8 (inside para 2): select para 2 "123\n456"
	e.Process(text, 8, Key('d'))
	e.Process(text, 8, Key('i'))
	r := e.Process(text, 8, Key('p'))
	if r.Kind != ResultExecute {
		t.Fatalf("dip: got Kind=%d", r.Kind)
	}
	if r.Range.Start != 5 || r.Range.End != 12 {
		t.Errorf("dip: got Range=[%d,%d), want [5,12)", r.Range.Start, r.Range.End)
	}
	// dap from 8: delete para 2 including surrounding blank lines
	e.Reset()
	e.Process(text, 8, Key('d'))
	e.Process(text, 8, Key('a'))
	r = e.Process(text, 8, Key('p'))
	if r.Kind != ResultExecute {
		t.Fatalf("dap: got Kind=%d", r.Kind)
	}
	if r.Range.Start != 3 || r.Range.End != 14 {
		t.Errorf("dap: got Range=[%d,%d), want [3,14)", r.Range.Start, r.Range.End)
	}
}

func TestEngineEscapeStates(t *testing.T) {
	var e Engine
	text := []rune("hello")
	// Esc after g: cancel g-pending
	e.Process(text, 0, Key('g'))
	r := e.Process(text, 0, Key(27))
	if r.Kind != ResultCancel {
		t.Errorf("Esc after g: got Kind=%d, want Cancel", r.Kind)
	}
	if e.state != stIdle {
		t.Errorf("Esc after g: state=%d, want stIdle", e.state)
	}
	// Esc after i (textobj pending): cancel
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('i'))
	r = e.Process(text, 0, Key(27))
	if r.Kind != ResultCancel {
		t.Errorf("Esc after di: got Kind=%d, want Cancel", r.Kind)
	}
	if e.state != stIdle {
		t.Errorf("Esc after di: state=%d, want stIdle", e.state)
	}
	// Esc after f: cancel char search pending
	e.Process(text, 0, Key('f'))
	r = e.Process(text, 0, Key(27))
	if r.Kind != ResultCancel {
		t.Errorf("Esc after f: got Kind=%d, want Cancel", r.Kind)
	}
}

func TestEngineEmptyBuffer(t *testing.T) {
	var e Engine
	text := []rune("")
	// All motions on empty buffer should return cursor 0
	motions := []rune{'w', 'b', 'e', '0', '$', '^', '{', '}', '%'}
	for _, k := range motions {
		e.Reset()
		r := e.Process(text, 0, Key(k))
		if r.Kind == ResultExecute {
			t.Errorf("%c on empty: got Execute, want Navigate", k)
		}
		if r.Kind == ResultNavigate && r.Cursor != 0 {
			t.Errorf("%c on empty: got Cursor=%d, want 0", k, r.Cursor)
		}
	}
}

func TestEngineX(t *testing.T) {
	var e Engine

	r := e.Process([]rune("hello"), 0, Key('x'))
	if r.Kind != ResultExecute || r.Op != OpDelete || r.Range != (Range{0, 1}) {
		t.Fatalf("x at 0: got Kind=%d Op=%d Range=%v", r.Kind, r.Op, r.Range)
	}

	r = e.Process([]rune("hello"), 4, Key('x'))
	if r.Kind != ResultExecute {
		t.Fatalf("x at eof: got Kind=%d", r.Kind)
	}

	r = e.Process([]rune("hello"), 2, Key('2'))
	if r.Kind != ResultNone {
		t.Fatalf("2x: count pre-handling returned %d", r.Kind)
	}
	r = e.Process([]rune("hello"), 2, Key('x'))
	if r.Kind != ResultExecute || r.Range != (Range{2, 4}) {
		t.Errorf("2x: got Range=%v, want [2,4)", r.Range)
	}

	r = e.Process([]rune(""), 0, Key('x'))
	if r.Kind != ResultNone {
		t.Errorf("x on empty: got %d, want None", r.Kind)
	}

	text := []rune("abc")
	r = e.Process(text, 1, Key('x'))
	if e.state != stIdle {
		t.Errorf("engine not reset after x, state=%d", e.state)
	}
	ar := ApplyOp(OpDelete, text, 1, r.Range)
	if string(ar.Text) != "ac" || ar.Cursor != 1 {
		t.Errorf("apply x: got %q at %d", string(ar.Text), ar.Cursor)
	}
}

func TestEngineXBack(t *testing.T) {
	var e Engine

	r := e.Process([]rune("hello"), 1, Key('X'))
	if r.Kind != ResultExecute || r.Op != OpDelete || r.Range != (Range{0, 1}) {
		t.Fatalf("X at 1: got %v", r.Range)
	}

	r = e.Process([]rune("hello"), 0, Key('X'))
	if r.Kind != ResultNone {
		t.Errorf("X at 0: got %d, want None", r.Kind)
	}

	r = e.Process([]rune("hello"), 3, Key('2'))
	if r.Kind != ResultNone {
		t.Fatalf("2X: count pre-handling returned %d", r.Kind)
	}
	r = e.Process([]rune("hello"), 3, Key('X'))
	if r.Kind != ResultExecute || r.Range != (Range{1, 3}) {
		t.Errorf("2X: got Range=%v, want [1,3)", r.Range)
	}

	text := []rune("hello")
	r = e.Process(text, 3, Key('3'))
	if r.Kind != ResultNone {
		t.Fatalf("3X pre-count: got %d", r.Kind)
	}
	r = e.Process(text, 3, Key('X'))
	if r.Kind != ResultExecute {
		// clamp: 3-3 = 0 -> Range{0,3}
		ar := ApplyOp(OpDelete, text, 3, r.Range)
		if string(ar.Text) != "lo" || ar.Cursor != 0 {
			t.Errorf("3X clamped: got %q at %d", string(ar.Text), ar.Cursor)
		}
	}

	r = e.Process([]rune(""), 0, Key('X'))
	if r.Kind != ResultNone {
		t.Errorf("X on empty: got %d, want None", r.Kind)
	}

	r = e.Process([]rune("ab"), 1, Key('5'))
	if r.Kind != ResultNone {
		t.Fatalf("5X: count pre-handling returned %d", r.Kind)
	}
	r = e.Process([]rune("ab"), 1, Key('X'))
	if r.Kind != ResultExecute || r.Range != (Range{0, 1}) {
		t.Errorf("5X past start: got Range=%v, want [0,1)", r.Range)
	}
}

func TestEngineDJ(t *testing.T) {
	var e Engine
	text := []rune("hello\nworld")
	// dj from position 1: delete from cursor to position 6 (world start) - 1 = "ello\n"
	e.Process(text, 1, Key('d'))
	result := e.Process(text, 1, Key('j'))
	if result.Kind != ResultExecute {
		t.Fatalf("dj: got Kind=%d, want ResultExecute", result.Kind)
	}
	// j from pos 1 goes to pos... line 1 start=0, col=1. Line 2 start=6, pos=6+1=7.
	// dj range: from pos 1 to pos 7. But with linewise handling...
	// Actually, the range for dj should be [1, 7), which is "ello\nwo" from cursor to near
	// the end of line 2. But vim's dj is characterwise, so it deletes from pos to motion dest.
	// Hmm, actually in vim, dj deletes the current line and the line below.
	if result.Range.End < result.Range.Start {
		t.Errorf("dj: invalid range [%d,%d)", result.Range.Start, result.Range.End)
	}
}

// ── ApplyOp regression tests (engine → ApplyOp end-to-end) ──────────────────

func TestApplyOpRangeNotTruncatedByCursor(t *testing.T) {
	text := []rune("hello world")
	var e Engine
	e.Process(text, 0, Key('d'))
	r := e.Process(text, 0, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatal("dw: expected ResultExecute")
	}
	ar := ApplyOp(OpDelete, text, 0, r.Range)
	if got, want := string(ar.Text), "world"; got != want {
		t.Errorf("dw from 0: got %q, want %q", got, want)
	}
	if ar.Cursor != 0 {
		t.Errorf("dw from 0: cursor=%d, want 0", ar.Cursor)
	}
}

func TestApplyOpRangeAbsoluteFromMiddle(t *testing.T) {
	text := []rune("test message")
	var e Engine
	e.Process(text, 5, Key('d'))
	r := e.Process(text, 5, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatal("dw from 5: expected ResultExecute")
	}
	ar := ApplyOp(OpDelete, text, 5, r.Range)
	if got, want := string(ar.Text), "test "; got != want {
		t.Errorf("dw from 5: got %q, want %q", got, want)
	}
}

func TestApplyOpD2WExhaustsWords(t *testing.T) {
	text := []rune("test message")
	var e Engine
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('2'))
	r := e.Process(text, 0, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatal("d2w: expected ResultExecute")
	}
	ar := ApplyOp(OpDelete, text, 0, r.Range)
	if got, want := string(ar.Text), ""; got != want {
		t.Errorf("d2w: got %q, want empty", got)
	}
}

func TestApplyOpD3WClampsToEOF(t *testing.T) {
	text := []rune("test message")
	var e Engine
	e.Process(text, 0, Key('d'))
	e.Process(text, 0, Key('3'))
	r := e.Process(text, 0, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatal("d3w: expected ResultExecute")
	}
	ar := ApplyOp(OpDelete, text, 0, r.Range)
	if got, want := string(ar.Text), ""; got != want {
		t.Errorf("d3w: got %q, want empty", got)
	}
}

func TestApplyOpDWOnSingleWord(t *testing.T) {
	text := []rune("hello")
	var e Engine
	e.Process(text, 0, Key('d'))
	r := e.Process(text, 0, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatal("dw single word: expected ResultExecute")
	}
	ar := ApplyOp(OpDelete, text, 0, r.Range)
	if got, want := string(ar.Text), ""; got != want {
		t.Errorf("dw single word: got %q, want empty", got)
	}
	if ar.Cursor != 0 {
		t.Errorf("dw single word: cursor=%d, want 0", ar.Cursor)
	}
}

func TestApplyOpDEFromMiddle(t *testing.T) {
	text := []rune("hello world")
	var e Engine
	e.Process(text, 6, Key('d'))
	r := e.Process(text, 6, Key('e'))
	if r.Kind != ResultExecute {
		t.Fatal("de from 6: expected ResultExecute")
	}
	ar := ApplyOp(OpDelete, text, 6, r.Range)
	if got, want := string(ar.Text), "hello "; got != want {
		t.Errorf("de from 6: got %q, want %q", got, want)
	}
}

func TestApplyOpChangeEntersInsertMode(t *testing.T) {
	text := []rune("hello world")
	var e Engine
	e.Process(text, 0, Key('c'))
	r := e.Process(text, 0, Key('w'))
	if r.Kind != ResultExecute {
		t.Fatal("cw: expected ResultExecute")
	}
	if r.Op != OpChange {
		t.Errorf("cw: op=%d, want OpChange", r.Op)
	}
	ar := ApplyOp(OpChange, text, 0, r.Range)
	if got, want := string(ar.Text), "world"; got != want {
		t.Errorf("cw: got %q, want %q", got, want)
	}
	if !ar.Insert {
		t.Error("cw: Insert=false, want true")
	}
}

func TestApplyOpBoundaries(t *testing.T) {
	text := []rune("abc")
	cases := []struct {
		name string
		op   Op
		r    Range
		want string
		cur  int
	}{
		{"empty range delete", OpDelete, Range{0, 0}, "abc", 0},
		{"full buffer delete", OpDelete, Range{0, 3}, "", 0},
		{"last char delete", OpDelete, Range{2, 3}, "ab", 2},
		{"yank", OpYank, Range{0, 3}, "abc", 0},
		{"yank empty", OpYank, Range{1, 1}, "abc", 0},
		{"change", OpChange, Range{0, 3}, "", 0},
		{"range beyond buffer", OpDelete, Range{0, 10}, "", 0},
		{"range reversed", OpDelete, Range{3, 1}, "a", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ar := ApplyOp(c.op, text, 0, c.r)
			if got := string(ar.Text); got != c.want {
				t.Errorf("text: got %q, want %q", got, c.want)
			}
			if ar.Cursor != c.cur {
				t.Errorf("cursor: got %d, want %d", ar.Cursor, c.cur)
			}
		})
	}
}

func TestOperatorDWAcrossPositions(t *testing.T) {
	cases := []struct {
		text string
		pos  int
		want string
	}{
		{"hello", 0, ""},
		{"hello ", 0, ""},
		{"a b c", 0, "b c"},
		{"a b c", 2, "a c"},
		{"a b c", 4, "a b "},
		{"test message", 0, "message"},
		{"test message", 5, "test "},
		{"test  message", 5, "test message"},
	}
	for _, c := range cases {
		text := []rune(c.text)
		var e Engine
		e.Process(text, c.pos, Key('d'))
		r := e.Process(text, c.pos, Key('w'))
		if r.Kind != ResultExecute {
			t.Fatalf("dw on %q from %d: expected Execute", c.text, c.pos)
		}
		ar := ApplyOp(OpDelete, text, c.pos, r.Range)
		if got := string(ar.Text); got != c.want {
			t.Errorf("dw on %q from %d: got %q, want %q", c.text, c.pos, got, c.want)
		}
	}
}

func TestOperatorDCountWAcrossPositions(t *testing.T) {
	cases := []struct {
		text string
		pos  int
		cnt  int
		want string
	}{
		{"a b c", 0, 2, "c"},
		{"a b c", 0, 3, ""},
		{"a b c", 0, 5, ""},
		{"one two three four", 0, 2, "three four"},
		{"test message", 0, 2, ""},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("pos%d_d%dw", c.pos, c.cnt), func(t *testing.T) {
			text := []rune(c.text)
			var e Engine
			e.Process(text, c.pos, Key('d'))
			for _, d := range fmt.Sprintf("%d", c.cnt) {
				e.Process(text, c.pos, Key(rune(d)))
			}
			r := e.Process(text, c.pos, Key('w'))
			if r.Kind != ResultExecute {
				t.Fatalf("d%sw: expected Execute", fmt.Sprintf("%d", c.cnt))
			}
			ar := ApplyOp(OpDelete, text, c.pos, r.Range)
			if got := string(ar.Text); got != c.want {
				t.Errorf("d%dw: got %q, want %q", c.cnt, got, c.want)
			}
		})
	}
}

func TestOperatorYank(t *testing.T) {
	cases := []struct {
		name string
		text string
		pos  int
		keys []rune
		want string
	}{
		{"yw hello", "hello world", 0, []rune{'y', 'w'}, "hello "},
		{"yw world", "hello world", 6, []rune{'y', 'w'}, "world"},
		{"ye hello", "hello world", 0, []rune{'y', 'e'}, "hello"},
		{"ye world", "hello world", 6, []rune{'y', 'e'}, "world"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := []rune(c.text)
			var e Engine
			for i, k := range c.keys {
				r := e.Process(text, c.pos, Key(k))
				if i == len(c.keys)-1 {
					if r.Kind != ResultExecute {
						t.Fatalf("%s: expected Execute, got %d", c.name, r.Kind)
					}
					ar := ApplyOp(OpYank, text, c.pos, r.Range)
					if got := string(ar.Text); got != c.text {
						t.Errorf("%s: text changed: got %q", c.name, got)
					}
					if ar.Yanked != c.want {
						t.Errorf("%s: yanked=%q, want %q", c.name, ar.Yanked, c.want)
					}
				}
			}
		})
	}
}

func TestOperatorCharSearch(t *testing.T) {
	cases := []struct {
		name string
		text string
		pos  int
		seq  string
		want string
	}{
		{"dfl hello", "hello world", 0, "fl", "lo world"},
		{"dfl l2", "hello world", 3, "fl", "held"},
		{"dfl x", "hello world", 0, "fx", "hello world"},
		{"dtl hello", "hello world", 0, "tl", "llo world"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := []rune(c.text)
			var e Engine
			var r Result
			e.Process(text, c.pos, Key('d'))
			e.Process(text, c.pos, Key(rune(c.seq[0])))
			r = e.Process(text, c.pos, Key(rune(c.seq[1])))
			ar := ApplyOp(OpDelete, text, c.pos, r.Range)
			if got := string(ar.Text); got != c.want {
				t.Errorf("%s: got %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestCountPrefixOperator(t *testing.T) {
	cases := []struct {
		name string
		text string
		pos  int
		keys []rune
		want string
	}{
		{"d3e single word", "hello", 0, []rune{'d', '3', 'e'}, ""},
		// This is an implementation detail, we do not really care about this failure:
		// {"d2$ line tail", "hello world", 0, []rune{'d', '2', '$'}, "hello world"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := []rune(c.text)
			var e Engine
			var r Result
			for _, k := range c.keys {
				r = e.Process(text, c.pos, Key(k))
			}
			if r.Kind != ResultExecute {
				t.Fatalf("%s: expected Execute got %d", c.name, r.Kind)
			}
			ar := ApplyOp(OpDelete, text, c.pos, r.Range)
			if got := string(ar.Text); got != c.want {
				t.Errorf("%s: got %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestTextObjectOperators(t *testing.T) {
	cases := []struct {
		name string
		text string
		pos  int
		seq  []rune
		want string
	}{
		{"diw single word", "hello", 0, []rune{'d', 'i', 'w'}, ""},
		{"daw single word", "hello", 0, []rune{'d', 'a', 'w'}, ""},
		{"di( paren", "a(b)c", 1, []rune{'d', 'i', '('}, "a()c"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := []rune(c.text)
			var e Engine
			var r Result
			for _, k := range c.seq {
				r = e.Process(text, c.pos, Key(k))
			}
			if r.Kind != ResultExecute {
				t.Fatalf("%s: expected Execute got %d", c.name, r.Kind)
			}
			ar := ApplyOp(OpDelete, text, c.pos, r.Range)
			if got := string(ar.Text); got != c.want {
				t.Errorf("%s: got %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestGPrefixOperators(t *testing.T) {
	cases := []struct {
		name string
		text string
		pos  int
		keys []rune
		want string
	}{
		{"dgg line1", "a\nb\nc", 4, []rune{'d', 'g', 'g'}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := []rune(c.text)
			var e Engine
			var r Result
			for _, k := range c.keys {
				r = e.Process(text, c.pos, Key(k))
			}
			if r.Kind != ResultExecute {
				t.Fatalf("%s: expected Execute got %d", c.name, r.Kind)
			}
			ar := ApplyOp(OpDelete, text, c.pos, r.Range)
			if got := string(ar.Text); got != c.want {
				t.Errorf("%s: got %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestParagraphOperators(t *testing.T) {
	cases := []struct {
		name string
		text string
		pos  int
		keys []rune
		want string
	}{
		{"d} first para", "abc\n\n123\n456\n\ndef", 0, []rune{'d', '}'}, "\n123\n456\n\ndef"},
		{"d{ mid para", "abc\n\n123\n456\n\ndef", 9, []rune{'d', '{'}, "abc\n456\n\ndef"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := []rune(c.text)
			var e Engine
			var r Result
			for _, k := range c.keys {
				r = e.Process(text, c.pos, Key(k))
			}
			if r.Kind != ResultExecute {
				t.Fatalf("%s: expected Execute got %d", c.name, r.Kind)
			}
			ar := ApplyOp(OpDelete, text, c.pos, r.Range)
			if got := string(ar.Text); got != c.want {
				t.Errorf("%s: got %q, want %q", c.name, got, c.want)
			}
		})
	}
}

func TestOperatorEscapeCancels(t *testing.T) {
	text := []rune("hello world")
	var e Engine
	e.Process(text, 0, Key('d'))
	if !e.Pending() {
		t.Error("d: expected pending state")
	}
	r := e.Process(text, 0, Key(27))
	if r.Kind != ResultCancel {
		t.Errorf("esc after d: got Kind=%d, want Cancel", r.Kind)
	}
	if e.Pending() {
		t.Error("engine should be idle after esc")
	}
}

func TestNextWordStartExhaustive(t *testing.T) {
	cases := []struct {
		text  string
		pos   int
		count int
		want  int
	}{
		{"ab cd ef", 0, 1, 3},
		{"ab cd ef", 3, 1, 6},
		{"ab cd ef", 5, 1, 6},
		{"ab cd ef", 7, 1, 7},
		{"ab cd ef", 0, 2, 6},
		{"ab cd ef", 0, 3, 7},
		{"ab cd ef", 0, 5, 7},
		{"word", 0, 1, 3},
		{"", 0, 1, 0},
		{"a b c", 2, 1, 4},
		{"a b c", 4, 1, 4},
	}
	for _, c := range cases {
		got := NextWordStart([]rune(c.text), c.pos, c.count)
		if got != c.want {
			t.Errorf("NextWordStart(%q,%d,%d) = %d, want %d", c.text, c.pos, c.count, got, c.want)
		}
	}
}

func TestKMotionUpward(t *testing.T) {
	text := []rune("hello\nworld\nfoo")

	cases := []struct {
		desc string
		pos  int
		want int
	}{
		{"k from line2 mid to line1 mid", 9, 3},
		{"k from line3 to line2", 14, 8},
		{"k from line2 start to line1 start", 6, 0},
		{"k from line2 end to line1 end", 10, 4},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var e Engine
			result := e.Process(text, c.pos, Key('k'))
			if result.Kind != ResultNavigate {
				t.Fatalf("%s: got Kind=%d, want ResultNavigate", c.desc, result.Kind)
			}
			if result.Cursor != c.want {
				t.Errorf("%s: got cursor=%d, want %d", c.desc, result.Cursor, c.want)
			}
		})
	}
}

func TestKMotionUpwardEdgeCases(t *testing.T) {
	cases := []struct {
		desc string
		text []rune
		pos  int
		want int
	}{
		{"k on first line, no-op", []rune("hello\nworld"), 3, 3},
		{"k at very beginning", []rune("hello\nworld"), 0, 0},
		{"k on single line, no-op", []rune("hello"), 3, 3},
		{"k from long col to shorter line", []rune("hi\nworld"), 6, 2},
		{"k from line3 to line2", []rune("aaa\nbbbb\ncccc"), 9, 4},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var e Engine
			result := e.Process(c.text, c.pos, Key('k'))
			if result.Kind != ResultNavigate {
				t.Fatalf("%s: got Kind=%d, want ResultNavigate", c.desc, result.Kind)
			}
			if result.Cursor != c.want {
				t.Errorf("%s: got cursor=%d, want %d", c.desc, result.Cursor, c.want)
			}
		})
	}
}

func TestKMotionWithCount(t *testing.T) {
	text := []rune("aaa\nbbb\nccc\nddd")

	cases := []struct {
		desc string
		pos  int
		cnt  int
		want int
	}{
		{"2k from line3 to line1", 8, 2, 0},
		{"3k from line4 to line1", 12, 3, 0},
		{"2k from line2 to line1", 4, 2, 0},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var e Engine
			// Feed count digits
			for _, d := range fmt.Sprintf("%d", c.cnt) {
				e.Process(text, c.pos, Key(rune(d)))
			}
			result := e.Process(text, c.pos, Key('k'))
			if result.Kind != ResultNavigate {
				t.Fatalf("%s: got Kind=%d, want ResultNavigate", c.desc, result.Kind)
			}
			if result.Cursor != c.want {
				t.Errorf("%s: got cursor=%d, want %d", c.desc, result.Cursor, c.want)
			}
		})
	}
}

func TestDDEmptyLine(t *testing.T) {
	var e Engine

	cases := []struct {
		name     string
		text     []rune
		cursor   int
		wantText string
	}{
		{
			"dd_empty_middle_line",
			[]rune("line1\n\nline3"),
			6, // cursor on the \n of the empty line (second \n)
			"line1\nline3",
		},
		{
			"dd_first_line_newline",
			[]rune("line1\n\nline3"),
			5, // cursor on the first \n (end of line1)
			"\nline3",
		},
		{
			"dd_last_line",
			[]rune("line1\n\nline3"),
			7, // cursor on 'l' of line3
			"line1\n\n",
		},
		{
			"dd_from_non_empty_line",
			[]rune("hello\nworld\nfoo"),
			1, // cursor on 'e'
			"world\nfoo",
		},
		{
			"dd_only_empty_line",
			[]rune("\n"),
			0, // cursor on \n
			"",
		},
		{
			"dd_trailing_empty_line",
			[]rune("hello\n\n"),
			6, // cursor on the last \n
			"hello\n",
		},
		{
			"dd_leading_empty_line",
			[]rune("\nhello"),
			0, // cursor on the first \n
			"hello",
		},
		{
			"dd_multiple_empty_lines",
			[]rune("\n\n\n"),
			1, // cursor on the middle \n
			"\n\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := make([]rune, len(tc.text))
			copy(text, tc.text)

			r1 := e.Process(text, tc.cursor, Key('d'))
			if r1.Kind != ResultNone {
				t.Fatalf("after first d: got %v, want idle", r1.Kind)
			}
			r2 := e.Process(text, tc.cursor, Key('d'))
			if r2.Kind != ResultExecute {
				t.Fatalf("dd: got Kind=%v, want ResultExecute", r2.Kind)
			}
			if r2.Op != OpDelete {
				t.Errorf("dd: got Op=%v, want OpDelete", r2.Op)
			}

			got := string(ApplyOp(r2.Op, text, tc.cursor, r2.Range).Text)
			if got != tc.wantText {
				t.Errorf("dd: got %q, want %q", got, tc.wantText)
			}
		})
	}
}
func TestEngineC(t *testing.T) {
	cases := []struct {
		desc   string
		text   []rune
		pos    int
		count  int
		want   Result
	}{
		{
			desc:  "C from middle of line",
			text:  []rune("hello world"),
			pos:   2,
			count: 1,
			want:  Result{Kind: ResultExecute, Op: OpChange, Range: Range{2, 11}, Insert: true},
		},
		{
			desc:  "C from start of line",
			text:  []rune("hello\nworld"),
			pos:   6,
			count: 1,
			want:  Result{Kind: ResultExecute, Op: OpChange, Range: Range{6, 11}, Insert: true},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var e Engine
			for i := 0; i < c.count-1; i++ {
				e.Process(c.text, c.pos, '2')
			}
			got := e.Process(c.text, c.pos, 'C')
			if got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestEngineY(t *testing.T) {
	cases := []struct {
		desc   string
		text   []rune
		pos    int
		count  int
		want   Result
	}{
		{
			desc:  "Y from middle of line",
			text:  []rune("hello world"),
			pos:   2,
			count: 1,
			want:  Result{Kind: ResultExecute, Op: OpYank, Range: Range{0, 11}, Insert: false},
		},
		{
			desc:  "Y from start of line",
			text:  []rune("hello\nworld"),
			pos:   6,
			count: 1,
			want:  Result{Kind: ResultExecute, Op: OpYank, Range: Range{6, 11}, Insert: false},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var e Engine
			for i := 0; i < c.count-1; i++ {
				e.Process(c.text, c.pos, '2')
			}
			got := e.Process(c.text, c.pos, 'Y')
			if got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}
