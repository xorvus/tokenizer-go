package bpe

import (
	"github.com/dlclark/regexp2/v2"
	"github.com/dlclark/regexp2/v2/helpers"
	"github.com/dlclark/regexp2/v2/syntax"
	"strings"
	"unicode"
)

/*
Capture(index = 0, unindex = -1)
 Atomic
  Alternate
   Concatenate
    One(Ch = ')
    Atomic
     Alternate
      Set(Set = [STstſ])
      Concatenate
       Set(Set = [Rr])
       Set(Set = [Ee])
      Concatenate
       Set(Set = [Vv])
       Set(Set = [Ee])
      Set(Set = [Mm])
      SetloopAtomic(Set = [Ll])(Min = 2, Max = 2)
      Set(Set = [Dd])
   Concatenate
    Setloop(Set = [^\n\r\p{L}\p{N}])(Min = 0, Max = 1)
    SetloopAtomic(Set = [\p{L}])(Min = 1, Max = inf)
   SetloopAtomic(Set = [\p{N}])(Min = 1, Max = 3)
   Concatenate
    OneloopAtomic(Ch = \ )(Min = 0, Max = 1)
    SetloopAtomic(Set = [^\s\p{L}\p{N}])(Min = 1, Max = inf)
    SetloopAtomic(Set = [\n\r])(Min = 0, Max = inf)
   Concatenate
    Setloop(Set = [\s])(Min = 0, Max = inf)
    SetloopAtomic(Set = [\n\r])(Min = 1, Max = inf)
   Concatenate
    Setloop(Set = [\s])(Min = 1, Max = inf)
    NegLook
     Set(Set = [^\s])
   SetloopAtomic(Set = [\s])(Min = 1, Max = inf)
*/
// From command line
// Pattern: "(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\\r\\n\\p{L}\\p{N}]?\\p{L}+|\\p{N}{1,3}| ?[^\\s\\p{L}\\p{N}]+[\\r\\n]*|\\s*[\\r\\n]+|\\s+(?!\\S)|\\s+"
// Options: regexp2.None

func CL100KPattern_FindFirstChar(r *regexp2.Runner) bool {
	pos := r.Runtextpos
	// Empty matches aren't possible
	if pos < len(r.Runtext) {
		return true
	}

	// No match found
	r.Runtextpos = len(r.Runtext)
	return false
}

func CL100KPattern_Execute(r *regexp2.Runner) error {
	atomic_stackpos := 0
	alternation_starting_pos := 0
	var charloop_starting_pos, charloop_ending_pos = 0, 0
	iteration := 0
	iteration1 := 0
	iteration2 := 0
	iteration3 := 0
	var charloop_starting_pos1, charloop_ending_pos1 = 0, 0
	iteration4 := 0
	iteration5 := 0
	var charloop_starting_pos2, charloop_ending_pos2 = 0, 0
	iteration6 := 0
	negativelookahead_starting_pos := 0
	iteration7 := 0
	pos := r.Runtextpos
	matchStart := pos

	var slice = r.Runtext[pos:]

	// Node: Atomic
	// Atomic group.
	atomic_stackpos = r.Runstackpos

	// Node: Alternate
	// Match with 7 alternative expressions, atomically.
	alternation_starting_pos = pos

	// Branch 0
	// Node: Concatenate
	// Node: One(Ch = ')
	// Match '\”.
	if len(slice) == 0 || slice[0] != '\'' {
		goto AlternationBranch
	}

	// Node: Atomic
	// Node: Alternate
	// Match with 6 alternative expressions, atomically.
	if len(slice) < 2 {
		goto AlternationBranch
	}

	switch slice[1] {
	case 'S', 'T', 's', 't', 'ſ':
		pos += 2
		slice = r.Runtext[pos:]

	case 'R', 'r':
		// Node: Concatenate
		// Node: Empty

		// Node: Set(Set = [Ee])
		// Match [Ee].
		if len(slice) < 3 || (slice[2]|0x20 != 'e') {
			goto AlternationBranch
		}

		pos += 3
		slice = r.Runtext[pos:]

	case 'V', 'v':
		// Node: Concatenate
		// Node: Empty

		// Node: Set(Set = [Ee])
		// Match [Ee].
		if len(slice) < 3 || (slice[2]|0x20 != 'e') {
			goto AlternationBranch
		}

		pos += 3
		slice = r.Runtext[pos:]

	case 'M', 'm':
		pos += 2
		slice = r.Runtext[pos:]

	case 'L', 'l':
		// Node: SetloopAtomic(Set = [Ll])(Min = 2, Max = 2)
		// Match [Ll] exactly 2 times.
		if len(slice) < 3 ||
			(slice[1]|0x20 != 'l') ||
			(slice[2]|0x20 != 'l') {
			goto AlternationBranch
		}

		pos += 3
		slice = r.Runtext[pos:]

	case 'D', 'd':
		pos += 2
		slice = r.Runtext[pos:]

	default:
		goto AlternationBranch
	}

	goto AlternationMatch

AlternationBranch:
	pos = alternation_starting_pos
	slice = r.Runtext[pos:]

	// Branch 1
	// Node: Concatenate
	// Node: Setloop(Set = [^\n\r\p{L}\p{N}])(Min = 0, Max = 1)
	// Match [^\n\r\p{L}\p{N}] greedily, optionally.
	charloop_starting_pos = pos

	if len(slice) > 0 && ((slice[0] < 128 && helpers.IsInASCIIBitmap(slice[0], 0xfc00ffffffffdbff, 0x78000001f8000001)) || (slice[0] >= 128 && set_4a7765fc40e8e8c7561121585d1545f9a51b44bf010a50f8b1b086a02ec1f07f.CharIn(slice[0]))) {
		slice = slice[1:]
		pos++
	}

	charloop_ending_pos = pos
	goto CharLoopEnd

CharLoopBacktrack:

	if err := r.CheckTimeout(); err != nil {
		return err
	}
	if charloop_starting_pos >= charloop_ending_pos {
		goto AlternationBranch1
	}
	charloop_ending_pos--
	pos = charloop_ending_pos
	slice = r.Runtext[pos:]

CharLoopEnd:

	// Node: SetloopAtomic(Set = [\p{L}])(Min = 1, Max = inf)
	// Match [\p{L}] atomically at least once.
	iteration = 0
	for iteration < len(slice) && unicode.In(slice[iteration], unicode.L) {
		iteration++
	}

	if iteration == 0 {
		goto CharLoopBacktrack
	}

	slice = slice[iteration:]
	pos += iteration

	goto AlternationMatch

AlternationBranch1:
	pos = alternation_starting_pos
	slice = r.Runtext[pos:]

	// Branch 2
	// Node: SetloopAtomic(Set = [\p{N}])(Min = 1, Max = 3)
	// Match [\p{N}] atomically at least 1 and at most 3 times.
	iteration1 = 0
	for iteration1 < 3 && iteration1 < len(slice) && unicode.In(slice[iteration1], unicode.N) {
		iteration1++
	}

	if iteration1 == 0 {
		goto AlternationBranch2
	}

	slice = slice[iteration1:]
	pos += iteration1

	goto AlternationMatch

AlternationBranch2:
	pos = alternation_starting_pos
	slice = r.Runtext[pos:]

	// Branch 3
	// Node: Concatenate
	// Node: OneloopAtomic(Ch = \ )(Min = 0, Max = 1)
	// Match ' ' atomically, optionally.
	if len(slice) > 0 && slice[0] == ' ' {
		slice = slice[1:]
		pos++
	}

	// Node: SetloopAtomic(Set = [^\s\p{L}\p{N}])(Min = 1, Max = inf)
	// Match [^\s\p{L}\p{N}] atomically at least once.
	iteration2 = 0
	for iteration2 < len(slice) && ((slice[iteration2] < 128 && helpers.IsInASCIIBitmap(slice[iteration2], 0xfc00fffeffffc1ff, 0x78000001f8000001)) || (slice[iteration2] >= 128 && set_93b362dc942d41f83c0905ca6229c31c41f863d4ab4d2d9e17dae21b5b922113.CharIn(slice[iteration2]))) {
		iteration2++
	}

	if iteration2 == 0 {
		goto AlternationBranch3
	}

	slice = slice[iteration2:]
	pos += iteration2

	// Node: SetloopAtomic(Set = [\n\r])(Min = 0, Max = inf)
	// Match [\n\r] atomically any number of times.
	iteration3 = helpers.IndexOfAnyExcept2(slice, '\n', '\r')
	if iteration3 < 0 {
		iteration3 = len(slice)
	}

	slice = slice[iteration3:]
	pos += iteration3

	goto AlternationMatch

AlternationBranch3:
	pos = alternation_starting_pos
	slice = r.Runtext[pos:]

	// Branch 4
	// Node: Concatenate
	// Node: Setloop(Set = [\s])(Min = 0, Max = inf)
	// Match [\s] greedily any number of times.
	charloop_starting_pos1 = pos

	iteration4 = 0
	for iteration4 < len(slice) && unicode.IsSpace(slice[iteration4]) {
		iteration4++
	}

	slice = slice[iteration4:]
	pos += iteration4

	charloop_ending_pos1 = pos
	goto CharLoopEnd1

CharLoopBacktrack1:

	if err := r.CheckTimeout(); err != nil {
		return err
	}
	if charloop_starting_pos1 >= charloop_ending_pos1 {
		goto AlternationBranch4
	}
	charloop_ending_pos1 = helpers.IndexOfAny2(r.Runtext[charloop_starting_pos1:charloop_ending_pos1], '\n', '\r')
	if charloop_ending_pos1 < 0 { // miss
		goto AlternationBranch4
	}
	charloop_ending_pos1 += charloop_starting_pos1
	pos = charloop_ending_pos1
	slice = r.Runtext[pos:]

CharLoopEnd1:

	// Node: SetloopAtomic(Set = [\n\r])(Min = 1, Max = inf)
	// Match [\n\r] atomically at least once.
	iteration5 = helpers.IndexOfAnyExcept2(slice, '\n', '\r')
	if iteration5 < 0 {
		iteration5 = len(slice)
	}

	if iteration5 == 0 {
		goto CharLoopBacktrack1
	}

	slice = slice[iteration5:]
	pos += iteration5

	goto AlternationMatch

AlternationBranch4:
	pos = alternation_starting_pos
	slice = r.Runtext[pos:]

	// Branch 5
	// Node: Concatenate
	// Node: Setloop(Set = [\s])(Min = 1, Max = inf)
	// Match [\s] greedily at least once.
	charloop_starting_pos2 = pos

	iteration6 = 0
	for iteration6 < len(slice) && unicode.IsSpace(slice[iteration6]) {
		iteration6++
	}

	if iteration6 == 0 {
		goto AlternationBranch5
	}

	slice = slice[iteration6:]
	pos += iteration6

	charloop_ending_pos2 = pos
	charloop_starting_pos2++
	goto CharLoopEnd2

CharLoopBacktrack2:

	if err := r.CheckTimeout(); err != nil {
		return err
	}
	if charloop_starting_pos2 >= charloop_ending_pos2 {
		goto AlternationBranch5
	}
	charloop_ending_pos2--
	pos = charloop_ending_pos2
	slice = r.Runtext[pos:]

CharLoopEnd2:

	// Node: NegLook
	// Zero-width negative lookahead
	negativelookahead_starting_pos = pos

	if err := r.CheckTimeout(); err != nil {
		return err
	}
	// Node: Set(Set = [^\s])
	// Match [^\s].
	if len(slice) == 0 || unicode.IsSpace(slice[0]) {
		goto NegativeLookaroundMatch
	}

	goto CharLoopBacktrack2

NegativeLookaroundMatch:
	pos = negativelookahead_starting_pos
	slice = r.Runtext[pos:]

	goto AlternationMatch

AlternationBranch5:
	pos = alternation_starting_pos
	slice = r.Runtext[pos:]

	// Branch 6
	// Node: SetloopAtomic(Set = [\s])(Min = 1, Max = inf)
	// Match [\s] atomically at least once.
	iteration7 = 0
	for iteration7 < len(slice) && unicode.IsSpace(slice[iteration7]) {
		iteration7++
	}

	if iteration7 == 0 {
		return nil // The input didn't match.
	}

	slice = slice[iteration7:]
	pos += iteration7

AlternationMatch:
	;

	r.Runstackpos = atomic_stackpos

	// The input matched.
	r.Runtextpos = pos
	r.Capture(0, matchStart, pos)
	// just to prevent an unused var error in certain regex's
	var _ = slice
	return nil
}

// The set [^\n\r\p{L}\p{N}]
var set_4a7765fc40e8e8c7561121585d1545f9a51b44bf010a50f8b1b086a02ec1f07f = syntax.NewCharSetRuntime("\x01\x02\x00\x00\x00\x02\x00\x00\x00\n\n\r\r\x01L\x01N")

// The set [^\s\p{L}\p{N}]
var set_93b362dc942d41f83c0905ca6229c31c41f863d4ab4d2d9e17dae21b5b922113 = syntax.NewCharSetRuntime("\x01\x00\x00\x00\x00\x03\x00\x00\x00\x01 \x01L\x01N")

func init() {
	regexp2.RegisterEngine("(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\\r\\n\\p{L}\\p{N}]?\\p{L}+|\\p{N}{1,3}| ?[^\\s\\p{L}\\p{N}]+[\\r\\n]*|\\s*[\\r\\n]+|\\s+(?!\\S)|\\s+", regexp2.RuntimeEngineData{
		Caps:          nil,
		CapNames:      nil,
		CapsList:      nil,
		CapSize:       1,
		FindFirstChar: CL100KPattern_FindFirstChar,
		Execute:       CL100KPattern_Execute,
	}, regexp2.None)
	var _ = helpers.Min
	var _ = syntax.NewCharSetRuntime
	var _ = strings.Index
	var _ = unicode.IsDigit
}
