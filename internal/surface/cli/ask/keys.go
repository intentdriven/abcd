package ask

import "unicode/utf8"

// KeyKind is what one key press is to the answer loop.
type KeyKind int

// The keys the arrow-key list acts on.
const (
	KeyRune KeyKind = iota // a printable character: Key.Rune
	KeyUp
	KeyDown
	KeyPageUp
	KeyPageDown
	KeyHome
	KeyEnd
	KeyEnter
	KeyBackspace
	KeyEscape
	KeyLeft
	KeyRight
	KeyTab
	KeyInterrupt // Ctrl-C, the byte 0x03 under raw mode
	KeySuspend   // Ctrl-Z, the byte 0x1a under raw mode
)

// Key is one key press.
type Key struct {
	Kind KeyKind
	Rune rune
}

// csiKeys are the CSI and SS3 sequences the list acts on, in both cursor-key
// modes, with the Home and End spellings terminals send.
var csiKeys = map[string]KeyKind{
	"\x1b[A": KeyUp, "\x1bOA": KeyUp,
	"\x1b[B": KeyDown, "\x1bOB": KeyDown,
	"\x1b[C": KeyRight, "\x1bOC": KeyRight,
	"\x1b[D": KeyLeft, "\x1bOD": KeyLeft,
	"\x1b[H": KeyHome, "\x1bOH": KeyHome, "\x1b[1~": KeyHome, "\x1b[7~": KeyHome,
	"\x1b[F": KeyEnd, "\x1bOF": KeyEnd, "\x1b[4~": KeyEnd, "\x1b[8~": KeyEnd,
	"\x1b[5~": KeyPageUp,
	"\x1b[6~": KeyPageDown,
	"\x1b[Z":  KeyLeft, // Shift-Tab: the previous part
}

// DecodeKeys decodes what one read from a raw terminal holds. A sequence is
// taken whole or not at all: an unknown CSI or SS3 sequence is dropped, a lone
// ESC at the end of the read is Escape (a terminal sends a sequence in one
// write), and a character split across reads is returned in rest for the
// next. Other control bytes are dropped.
func DecodeKeys(b []byte) (keys []Key, rest []byte) {
	for len(b) > 0 {
		c := b[0]
		switch {
		case c == 0x1b:
			n := sequenceLen(b)
			if n == 1 {
				keys = append(keys, Key{Kind: KeyEscape})
			} else if k, ok := csiKeys[string(b[:n])]; ok {
				keys = append(keys, Key{Kind: k})
			}
			b = b[n:]
			continue
		case c == '\r' || c == '\n':
			keys = append(keys, Key{Kind: KeyEnter})
		case c == 0x7f || c == 0x08:
			keys = append(keys, Key{Kind: KeyBackspace})
		case c == '\t':
			keys = append(keys, Key{Kind: KeyTab})
		case c == 0x03:
			keys = append(keys, Key{Kind: KeyInterrupt})
		case c == 0x1a:
			keys = append(keys, Key{Kind: KeySuspend})
		case c < 0x20:
		case c < utf8.RuneSelf:
			keys = append(keys, Key{Kind: KeyRune, Rune: rune(c)})
		default:
			if !utf8.FullRune(b) {
				return keys, b
			}
			r, n := utf8.DecodeRune(b)
			if r != utf8.RuneError && r >= 0xa0 {
				keys = append(keys, Key{Kind: KeyRune, Rune: r})
			}
			b = b[n:]
			continue
		}
		b = b[1:]
	}
	return keys, nil
}

// sequenceLen is the length of the escape sequence b starts with: 1 for a
// lone ESC (or ESC before anything that does not start a sequence), the whole
// CSI up to its final byte, or ESC O and one byte.
func sequenceLen(b []byte) int {
	if len(b) < 2 {
		return 1
	}
	switch b[1] {
	case '[':
		for i := 2; i < len(b); i++ {
			if b[i] >= 0x40 && b[i] <= 0x7e {
				return i + 1
			}
		}
		return len(b)
	case 'O':
		if len(b) >= 3 {
			return 3
		}
		return len(b)
	}
	return 1
}
