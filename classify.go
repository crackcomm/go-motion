// Package motion implements vim cursor motions, matching neovim's behavior
// as closely as possible. The core abstraction is "character class" — every
// character belongs to one of four classes (Space, NonKeyword, Keyword, Emoji),
// and word motions (w, b, e, ge) operate on sequences of characters sharing
// the same class.
//
// Reference: neovim's src/nvim/textobject.c and src/nvim/mbyte.c
package motion

import "unicode"

// Class represents the vim "character class" used by word motions.
//
// Neovim's cls() (textobject.c:292) returns:
//
//	0 - white space
//	1 - punctuation
//	2 - keyword characters (letters, digits and underscore)
//	3 - emoji (via utf_class())
//
// Word boundaries change when the character class changes.
// For uppercase motions (W, B, E, gE), cls_bigword=true collapses all
// non-zero classes to 1 — only whitespace boundaries matter.
type Class int

const (
	Space      Class = iota // 0: whitespace — spaces, tabs, NUL, non-breaking space
	NonKeyword              // 1: punctuation — everything that isn't space or keyword
	Keyword                 // 2: keyword — letters, digits, underscore, CJK, etc.
	Emoji                   // 3: emoji — separate class so emoji act as their own "word"
)

// Classify returns the vim class of rune r.
//
// Latin-1 (< 256) follows neovim's default 'iskeyword' option value
// (which is "@,48-57,_,192-255"), see :help iskeyword:
//   - @     = alphabetic letters (a-z, A-Z)
//   - 48-57 = ASCII digits (0-9)
//   - _     = underscore
//   - 192-255 = Latin-1 supplement accented letters (e.g. é, ñ, ü)
//
// Unicode (≥ 256) uses Go's unicode package categories. This is an
// approximation of neovim's utf_class_tab() (mbyte.c:1227) which does a
// binary search over ~50 explicit Unicode range intervals. Our approach
// handles the common cases correctly; if edge cases matter, we can add
// the full interval table later.
//
// Reference: neovim's cls() in textobject.c:292, utf_class() in mbyte.c:1222
func Classify(r rune) Class {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return Space
	}

	if r < 256 {
		return classifyLatin1(byte(r))
	}

	if unicode.Is(unicode.Space, r) {
		return Space
	}

	if isEmoji(r) {
		return Emoji
	}

	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return Keyword
	}

	// CJK, Hiragana, Katakana, Hangul are treated as keyword characters
	// in neovim (the unicode table returns non-zero classes for these ranges).
	if isCJK(r) {
		return Keyword
	}

	return NonKeyword
}

// ClassifyBigword collapses all non-space classes to NonKeyword.
//
// This implements neovim's cls_bigword flag (textobject.c:285): when true,
// all non-zero classes are reported as 1, so only whitespace boundaries
// matter. Used by uppercase motions: W, B, E, gE.
//
// Reference: cls() in textobject.c:292-306
func ClassifyBigword(r rune) Class {
	c := Classify(r)
	if c == Space {
		return Space
	}
	return NonKeyword
}

// classifyLatin1 implements the Latin-1 character classification matching
// neovim's default 'iskeyword' option.
//
// In neovim, Latin-1 characters (c < 0x100) are checked via vim_iswordc_tab()
// which consults the 'iskeyword' option buffer. The default value is
// "@,48-57,_,192-255" meaning:
//   - class 0: space, tab, NUL, 0xa0 (non-breaking space)
//   - class 2: characters matching iskeyword (letters, digits, _, 0xc0-0xff)
//   - class 1: everything else (punctuation, symbols)
//
// Reference: utf_class_tab() in mbyte.c:1312-1319
func classifyLatin1(c byte) Class {
	if c == 0xa0 {
		return Space
	}

	if iskeyword(c) {
		return Keyword
	}

	return NonKeyword
}

// iskeyword returns true for characters matching neovim's default
// 'iskeyword' value "@,48-57,_,192-255".
//
// Neovim's 'iskeyword' option defines what characters make up a "keyword"
// (class 2). The default breaks down as:
//   - @  = all alphabetic characters via isalpha()
//   - 48-57 = ASCII digits 0-9
//   - _  = underscore
//   - 192-255 = Latin-1 supplement (accented characters like À-ÿ)
//
// Note: neovim also considers characters > 0x100 via utf_class_tab(),
// but for Latin-1 this is the complete check.
//
// Reference: vim_iswordc_tab() in charset.c, 'iskeyword' documentation
func iskeyword(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' ||
		(c >= 0xc0)
}

// isCJK matches the CJK / Hiragana / Katakana ranges from neovim's
// utf_class_tab() (mbyte.c:1235-1307).
//
// In neovim, these ranges are assigned non-zero classes (various keyword
// classes in the 0x3040-0x9fff range). The exact class number doesn't
// matter for word motion since we only compare equality — any non-zero
// class behaves like class 2 for boundary purposes.
//
// Ranges included:
//
//	0x3040-0x309f  Hiragana
//	0x30a0-0x30ff  Katakana
//	0x3300-0x9fff  CJK Ideographs (combined several neovim intervals)
//	0xac00-0xd7a3  Hangul Syllables
//	0xf900-0xfaff  CJK Ideographs
//	0x20000-0x2fa1f  CJK Extension B, C, D + Supplement
func isCJK(r rune) bool {
	return (r >= 0x3040 && r <= 0x309f) ||
		(r >= 0x30a0 && r <= 0x30ff) ||
		(r >= 0x3300 && r <= 0x9fff) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0x20000 && r <= 0x2a6df) ||
		(r >= 0x2a700 && r <= 0x2b73f) ||
		(r >= 0x2b740 && r <= 0x2b81f) ||
		(r >= 0x2f800 && r <= 0x2fa1f)
}

// isEmoji returns true for common emoji ranges.
//
// Neovim's utf_class_tab() (mbyte.c:1322-1326) uses utf8proc to detect
// emoji and returns class 3 for them. Emoji are treated as a separate
// class so that sequences like "hello😀world" have three words:
// "hello", "😀", "world".
//
// This list covers the major emoji blocks. The full set is defined by
// Unicode's Extended_Pictographic property (neovim uses utf8proc for this).
// We cover the most common ranges; edge cases can be added as needed.
//
// Reference: prop_is_emojilike() in neovim's mbyte.c
func isEmoji(r rune) bool {
	return (r >= 0x1f300 && r <= 0x1f9ff) ||
		(r >= 0x1fa00 && r <= 0x1fa6f) ||
		(r >= 0x1fa70 && r <= 0x1faff) ||
		r == 0x00a9 ||
		r == 0x00ae ||
		r == 0x203d ||
		r == 0x2049 ||
		r == 0x2122 ||
		r == 0x2139 ||
		r >= 0x2194 && r <= 0x2199 ||
		r >= 0x21a9 && r <= 0x21aa ||
		r >= 0x231a && r <= 0x231b ||
		r >= 0x2328 && r <= 0x2328 ||
		r >= 0x23cf && r <= 0x23cf ||
		r >= 0x23e9 && r <= 0x23f3 ||
		r >= 0x23f8 && r <= 0x23fa ||
		r >= 0x24c2 && r <= 0x24c2 ||
		r >= 0x25aa && r <= 0x25ab ||
		r >= 0x25b6 && r <= 0x25b6 ||
		r >= 0x25c0 && r <= 0x25c0 ||
		r >= 0x25fb && r <= 0x25fe ||
		r >= 0x2600 && r <= 0x27bf ||
		r >= 0x2934 && r <= 0x2935 ||
		r >= 0x2b05 && r <= 0x2b07 ||
		r >= 0x2b1b && r <= 0x2b1c ||
		r >= 0x2b50 && r <= 0x2b50 ||
		r >= 0x2b55 && r <= 0x2b55 ||
		r >= 0x3030 && r <= 0x3030 ||
		r >= 0x303d && r <= 0x303d ||
		r >= 0x3297 && r <= 0x3297 ||
		r >= 0x3299 && r <= 0x3299
}
