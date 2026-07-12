package text

import (
	"cursortab/assert"
	"testing"
)

func TestGroupChanges_SingleModification(t *testing.T) {
	changes := map[int]LineChange{
		1: {
			Type:       ChangeInlineDiff,
			Content:    "Hello there",
			OldContent: "Hello world",
			ColStart:   6,
			ColEnd:     11,
			Spans:      []InlineSpan{{OldStart: 6, OldEnd: 11, NewStart: 6, NewEnd: 11}},
		},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 1, len(groups), "number of groups")
	assert.Equal(t, "modification", groups[0].Type, "group type")
	assert.Equal(t, 1, groups[0].StartLine, "start line")
	assert.Equal(t, 1, groups[0].EndLine, "end line")
	assert.Equal(t, "inline_diff", groups[0].RenderHint, "render hint")
	assert.Equal(t, 6, groups[0].ColStart, "col start")
	assert.Equal(t, 11, groups[0].ColEnd, "col end")
	assert.Equal(t, 1, len(groups[0].Spans), "spans carried onto group")
	assert.Equal(t, InlineSpan{OldStart: 6, OldEnd: 11, NewStart: 6, NewEnd: 11}, groups[0].Spans[0], "span")
}

func TestGroupChanges_SingleAddition(t *testing.T) {
	changes := map[int]LineChange{
		2: {
			Type:    ChangeAddition,
			Content: "new line",
		},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 1, len(groups), "number of groups")
	assert.Equal(t, "addition", groups[0].Type, "group type")
	assert.Equal(t, 2, groups[0].StartLine, "start line")
	assert.Equal(t, 2, groups[0].EndLine, "end line")
	assert.Equal(t, "", groups[0].RenderHint, "render hint for addition")
}

func TestGroupChanges_ConsecutiveAdditions(t *testing.T) {
	changes := map[int]LineChange{
		2: {Type: ChangeAddition, Content: "line a"},
		3: {Type: ChangeAddition, Content: "line b"},
		4: {Type: ChangeAddition, Content: "line c"},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 1, len(groups), "should be grouped into one")
	assert.Equal(t, "addition", groups[0].Type, "group type")
	assert.Equal(t, 2, groups[0].StartLine, "start line")
	assert.Equal(t, 4, groups[0].EndLine, "end line")
	assert.Equal(t, 3, len(groups[0].Lines), "number of lines")
}

func TestGroupChanges_ConsecutiveModifications(t *testing.T) {
	changes := map[int]LineChange{
		2: {Type: ChangeModification, Content: "new 2", OldContent: "old 2"},
		3: {Type: ChangeModification, Content: "new 3", OldContent: "old 3"},
		4: {Type: ChangeModification, Content: "new 4", OldContent: "old 4"},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 1, len(groups), "should be grouped into one")
	assert.Equal(t, "modification", groups[0].Type, "group type")
	assert.Equal(t, 2, groups[0].StartLine, "start line")
	assert.Equal(t, 4, groups[0].EndLine, "end line")
	assert.Equal(t, 3, len(groups[0].Lines), "number of lines")
	assert.Equal(t, 3, len(groups[0].OldLines), "number of old lines")
	assert.Equal(t, "", groups[0].RenderHint, "no render hint for multi-line")
}

func TestGroupChanges_NonConsecutive(t *testing.T) {
	changes := map[int]LineChange{
		2: {Type: ChangeModification, Content: "new 2", OldContent: "old 2"},
		4: {Type: ChangeModification, Content: "new 4", OldContent: "old 4"},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 2, len(groups), "should be two groups")
	assert.Equal(t, 2, groups[0].StartLine, "first group start")
	assert.Equal(t, 4, groups[1].StartLine, "second group start")
}

func TestGroupChanges_MixedTypes(t *testing.T) {
	changes := map[int]LineChange{
		2: {Type: ChangeModification, Content: "mod", OldContent: "old"},
		3: {Type: ChangeAddition, Content: "add"},
	}

	groups := GroupChanges(changes)

	// Modification and addition are different types, so they should be separate groups
	assert.Equal(t, 2, len(groups), "should be two groups for different types")
}

func TestGroupChanges_DeletionsIncluded(t *testing.T) {
	changes := map[int]LineChange{
		2: {Type: ChangeDeletion, Content: "deleted"},
		3: {Type: ChangeAddition, Content: "added"},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 2, len(groups), "deletion and addition should be separate groups")
	assert.Equal(t, "deletion", groups[0].Type, "first group is deletion")
	assert.Equal(t, "addition", groups[1].Type, "second group is addition")
}

func TestGroupChanges_OnlyDeletions(t *testing.T) {
	changes := map[int]LineChange{
		2: {Type: ChangeDeletion, Content: "deleted 1"},
		3: {Type: ChangeDeletion, Content: "deleted 2"},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 1, len(groups), "one group for consecutive deletions")
	assert.Equal(t, "deletion", groups[0].Type, "group type is deletion")
}

func TestGroupChanges_Empty(t *testing.T) {
	groups := GroupChanges(nil)
	assert.Nil(t, groups, "no groups for empty changes")

	groups = GroupChanges(map[int]LineChange{})
	assert.Nil(t, groups, "no groups for empty map")
}

func TestGroupChanges_AppendCharsHint(t *testing.T) {
	changes := map[int]LineChange{
		1: {
			Type:       ChangeAppendChars,
			Content:    "hello world",
			OldContent: "hello",
			ColStart:   5,
			ColEnd:     11,
		},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 1, len(groups), "one group")
	assert.Equal(t, "append_chars", groups[0].RenderHint, "render hint")
	assert.Equal(t, 5, groups[0].ColStart, "col start")
	assert.Equal(t, 11, groups[0].ColEnd, "col end")
}

func TestGroupChanges_MultiLineWithDifferentColumns(t *testing.T) {
	// Changes with render hints are never merged - each gets its own group
	changes := map[int]LineChange{
		1: {Type: ChangeInlineDiff, Content: "a", OldContent: "b", ColStart: 0, ColEnd: 1},
		2: {Type: ChangeInlineDiff, Content: "c", OldContent: "d", ColStart: 5, ColEnd: 6},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 2, len(groups), "hinted changes stay separate")
	assert.Equal(t, "inline_diff", groups[0].RenderHint, "first group has hint")
	assert.Equal(t, 0, groups[0].ColStart, "first group col start")
	assert.Equal(t, 1, groups[0].ColEnd, "first group col end")
	assert.Equal(t, "inline_diff", groups[1].RenderHint, "second group has hint")
	assert.Equal(t, 5, groups[1].ColStart, "second group col start")
	assert.Equal(t, 6, groups[1].ColEnd, "second group col end")
}

func TestGroupChanges_MultiLineWithIdenticalColumns(t *testing.T) {
	// Changes with render hints are never merged - each gets its own single-line group
	// This allows the Lua renderer to apply char-level rendering to each line individually
	changes := map[int]LineChange{
		1: {Type: ChangeInlineDiff, Content: "application.route()", OldContent: "app.route()", ColStart: 3, ColEnd: 11},
		2: {Type: ChangeInlineDiff, Content: "application.route()", OldContent: "app.route()", ColStart: 3, ColEnd: 11},
		3: {Type: ChangeInlineDiff, Content: "application.route()", OldContent: "app.route()", ColStart: 3, ColEnd: 11},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 3, len(groups), "each hinted change gets its own group")
	for i, group := range groups {
		assert.Equal(t, "inline_diff", group.RenderHint, "group has hint")
		assert.Equal(t, 3, group.ColStart, "col start")
		assert.Equal(t, 11, group.ColEnd, "col end")
		assert.Equal(t, i+1, group.StartLine, "start line")
		assert.Equal(t, i+1, group.EndLine, "end line")
	}
}

func TestGroupChanges_MultiLineMixedHints(t *testing.T) {
	// Multi-line groups with different change types should clear RenderHint
	changes := map[int]LineChange{
		1: {Type: ChangeInlineDiff, Content: "a", OldContent: "b", ColStart: 0, ColEnd: 1},
		2: {Type: ChangeAppendChars, Content: "cd", OldContent: "c", ColStart: 1, ColEnd: 2},
	}

	groups := GroupChanges(changes)

	// Different change types = different group types = separate groups
	assert.Equal(t, 2, len(groups), "different types = separate groups")
}

// TestGroupChanges_ConsecutiveModificationsWithRenderHint verifies that consecutive
// modifications group together even when the first has a character-level render hint.
// A multi-line group cannot use a single-line hint, so it is cleared on merge.
func TestGroupChanges_ConsecutiveModificationsWithRenderHint(t *testing.T) {
	changes := map[int]LineChange{
		1: {
			Type: ChangeInlineDiff, Content: "def foo_new(data):", OldContent: "def foo_ol(data):",
			ColStart: 8, ColEnd: 15,
			Spans: []InlineSpan{{OldStart: 4, OldEnd: 10, NewStart: 4, NewEnd: 11}},
		},
		2: {Type: ChangeModification, Content: "    return x, y", OldContent: "    return x"},
	}

	groups := GroupChanges(changes)

	assert.Equal(t, 1, len(groups), "consecutive modifications should form 1 group")
	assert.Equal(t, "modification", groups[0].Type, "group type")
	assert.Equal(t, 1, groups[0].StartLine, "start line")
	assert.Equal(t, 2, groups[0].EndLine, "end line")
	assert.Equal(t, "", groups[0].RenderHint, "render hint cleared for multi-line group")
	assert.Nil(t, groups[0].Spans, "spans cleared for multi-line group")
	assert.Equal(t, 2, len(groups[0].Lines), "both new lines present")
	assert.Equal(t, 2, len(groups[0].OldLines), "both old lines present")
}

func TestCopyGroupsDeepCopiesSpans(t *testing.T) {
	original := []*Group{
		{
			Type:       "modification",
			StartLine:  1,
			EndLine:    1,
			RenderHint: "inline_diff",
			Lines:      []string{"Hello there"},
			OldLines:   []string{"Hello world"},
			Spans:      []InlineSpan{{OldStart: 6, OldEnd: 11, NewStart: 6, NewEnd: 11}},
		},
	}

	copied := CopyGroups(original)
	copied[0].Spans[0].OldStart = 99

	assert.Equal(t, 6, original[0].Spans[0].OldStart, "mutating the copy must not affect the original")
}

func TestCalculateCursorPosition_Modification(t *testing.T) {
	changes := map[int]LineChange{
		1: {Type: ChangeModification, Content: "modified line"},
	}
	newLines := []string{"modified line"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 1, line, "cursor line")
	assert.Equal(t, 13, col, "cursor col at end of line")
}

func TestCalculateCursorPosition_Addition(t *testing.T) {
	changes := map[int]LineChange{
		2: {Type: ChangeAddition, Content: "new line"},
	}
	newLines := []string{"line 1", "new line", "line 3"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 2, line, "cursor line")
	assert.Equal(t, 8, col, "cursor col at end of new line")
}

func TestCalculateCursorPosition_LatestLine(t *testing.T) {
	// Cursor goes to the latest (highest line number) non-deletion change
	changes := map[int]LineChange{
		1: {Type: ChangeModification, Content: "mod"},
		3: {Type: ChangeAddition, Content: "add"},
	}
	newLines := []string{"mod", "line 2", "add"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 3, line, "cursor at latest changed line")
	assert.Equal(t, 3, col, "cursor at end of line")
}

func TestCalculateCursorPosition_OnlyDeletions(t *testing.T) {
	changes := map[int]LineChange{
		1: {Type: ChangeDeletion, Content: "deleted"},
	}
	newLines := []string{}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, -1, line, "no cursor for deletions")
	assert.Equal(t, -1, col, "no cursor col for deletions")
}

func TestCalculateCursorPosition_Empty(t *testing.T) {
	line, col := CalculateCursorPosition(nil, nil)
	assert.Equal(t, -1, line, "no cursor for empty")
	assert.Equal(t, -1, col, "no cursor col for empty")

	line, col = CalculateCursorPosition(map[int]LineChange{}, nil)
	assert.Equal(t, -1, line, "no cursor for empty map")
	assert.Equal(t, -1, col, "no cursor col for empty map")
}

func TestCalculateCursorPosition_ClampToBuffer(t *testing.T) {
	// Cursor line should be clamped to buffer size
	changes := map[int]LineChange{
		10: {Type: ChangeAddition, Content: "line"},
	}
	newLines := []string{"only", "two", "lines"}

	line, _ := CalculateCursorPosition(changes, newLines)

	assert.True(t, line <= len(newLines), "cursor clamped to buffer size")
}

// Cursor should be placed at end of change, not end of line
func TestCalculateCursorPosition_AppendChars(t *testing.T) {
	// console.log(|); -> console.log("hello"|);
	// ColEnd marks where the appended text ends
	changes := map[int]LineChange{
		1: {
			Type:       ChangeAppendChars,
			Content:    `console.log("hello");`,
			OldContent: "console.log();",
			ColStart:   12, // after (
			ColEnd:     19, // after "hello"
		},
	}
	newLines := []string{`console.log("hello");`}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 1, line, "cursor line")
	assert.Equal(t, 19, col, "cursor at end of change, not end of line (21)")
}

func TestCalculateCursorPosition_InlineDiffReplacement(t *testing.T) {
	// app.route() -> application.route()
	// Cursor should be after "application", not at end of line
	changes := map[int]LineChange{
		1: {
			Type:       ChangeInlineDiff,
			Content:    "application.route()",
			OldContent: "app.route()",
			ColStart:   0,
			ColEnd:     11, // end of "application"
			Spans:      []InlineSpan{{OldStart: 0, OldEnd: 3, NewStart: 0, NewEnd: 11}},
		},
	}
	newLines := []string{"application.route()"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 1, line, "cursor line")
	assert.Equal(t, 11, col, "cursor at end of replacement, not end of line (19)")
}

func TestCalculateCursorPosition_InlineDiffDeletion(t *testing.T) {
	// "Hello world John" -> "Hello John"
	// Deleted "world " at position 6; the envelope is empty in new-line
	// coordinates, so ColEnd is the deletion point.
	changes := map[int]LineChange{
		1: {
			Type:       ChangeInlineDiff,
			Content:    "Hello John",
			OldContent: "Hello world John",
			ColStart:   6,
			ColEnd:     6,
			Spans:      []InlineSpan{{OldStart: 6, OldEnd: 12, NewStart: 6, NewEnd: 6}},
		},
	}
	newLines := []string{"Hello John"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 1, line, "cursor line")
	assert.Equal(t, 6, col, "cursor at deletion point")
}

func TestCalculateCursorPosition_AppendCharsAtLineEnd(t *testing.T) {
	// When ColEnd equals line length, behavior is same as before
	changes := map[int]LineChange{
		1: {
			Type:       ChangeAppendChars,
			Content:    "hello world",
			OldContent: "hello",
			ColStart:   5,
			ColEnd:     11, // equals len("hello world")
		},
	}
	newLines := []string{"hello world"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 1, line, "cursor line")
	assert.Equal(t, 11, col, "cursor at end of line (which is also end of change)")
}

func TestCalculateCursorPosition_CharLevelPriorityOverAddition(t *testing.T) {
	// AppendChars has lower priority than Modification, but higher than nothing
	// When we have AppendChars and Addition, AppendChars wins in priority order
	changes := map[int]LineChange{
		1: {
			Type:       ChangeAppendChars,
			Content:    `log("hi");`,
			OldContent: "log();",
			ColStart:   4,
			ColEnd:     8, // after "hi"
		},
		2: {Type: ChangeAddition, Content: "new line"},
	}
	newLines := []string{`log("hi");`, "new line"}

	line, col := CalculateCursorPosition(changes, newLines)

	// Addition has higher priority than AppendChars in current logic
	// But cursor col should still be at end of the selected line
	assert.Equal(t, 2, line, "cursor at addition line (higher priority)")
	assert.Equal(t, 8, col, "cursor at end of addition line")
}

func TestCalculateCursorPosition_MultipleAppendChars(t *testing.T) {
	// Multiple AppendChars changes: cursor at the last line's ColEnd
	changes := map[int]LineChange{
		1: {
			Type:       ChangeAppendChars,
			Content:    "first change",
			OldContent: "first",
			ColStart:   5,
			ColEnd:     12,
		},
		3: {
			Type:       ChangeAppendChars,
			Content:    "third change",
			OldContent: "third",
			ColStart:   5,
			ColEnd:     12,
		},
	}
	newLines := []string{"first change", "second", "third change"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 3, line, "cursor at last append chars line")
	assert.Equal(t, 12, col, "cursor at ColEnd of last append chars")
}

func TestCalculateCursorPosition_LatestLineWithCharLevel(t *testing.T) {
	// Cursor goes to the latest line; for char-level changes, uses ColEnd
	changes := map[int]LineChange{
		1: {Type: ChangeModification, Content: "modified line", OldContent: "old"},
		2: {
			Type:       ChangeAppendChars,
			Content:    "append here",
			OldContent: "append",
			ColStart:   6,
			ColEnd:     11,
		},
	}
	newLines := []string{"modified line", "append here"}

	line, col := CalculateCursorPosition(changes, newLines)

	assert.Equal(t, 2, line, "cursor at latest changed line")
	assert.Equal(t, 11, col, "cursor at ColEnd of append_chars")
}

// TestGroupsMustReflectActualBufferState verifies that groups computed from the
// actual buffer diff can differ from pre-computed groups when buffer state changes.
func TestGroupsMustReflectActualBufferState(t *testing.T) {
	// Scenario: completion expands 1 line to 2 lines, where line 1 is unchanged
	// First diff (1 old vs 2 new) sees: line 1 EQUAL, line 2 ADDITION
	oldLine := "const x = 1;"
	newLines := []string{"const x = 1;", "const y = 2;"}

	firstDiff := ComputeDiff(JoinLines([]string{oldLine}), JoinLines(newLines))

	assert.Equal(t, 1, len(firstDiff.Changes), "first diff: 1 change")
	assert.Equal(t, ChangeAddition, firstDiff.Changes[0].Type, "first diff: addition at line 2")

	firstGroups := GroupChanges(firstDiff.ChangesMap())
	assert.Equal(t, 1, len(firstGroups), "first diff: 1 group")
	assert.Equal(t, "addition", firstGroups[0].Type, "first diff group: addition")
	assert.Nil(t, firstGroups[0].OldLines, "addition has no old_lines")

	// But when applying to actual buffer, line 2 has different content
	// Second diff (actual buffer vs new content) sees: MODIFICATION
	actualBufferLine := "const y = 0;"
	newContent := "const y = 2;"

	secondDiff := ComputeDiff(JoinLines([]string{actualBufferLine}), JoinLines([]string{newContent}))

	assert.Equal(t, 1, len(secondDiff.Changes), "second diff: 1 change")
	change := secondDiff.Changes[0]
	isModification := change.Type == ChangeModification || change.Type == ChangeInlineDiff
	assert.True(t, isModification, "second diff: modification type")
	assert.True(t, change.OldContent != "", "modification has old content")

	secondGroups := GroupChanges(secondDiff.ChangesMap())
	assert.Equal(t, 1, len(secondGroups), "second diff: 1 group")
	assert.Equal(t, "modification", secondGroups[0].Type, "second diff group: modification")
	assert.NotNil(t, secondGroups[0].OldLines, "modification has old_lines")

	// Key assertion: the two diffs produce different group types
	assert.True(t, firstGroups[0].Type != secondGroups[0].Type,
		"groups from different buffer states should differ")
}

func TestValidateRenderHintsForCursor_DowngradesAppendCharsBeforeCursor(t *testing.T) {
	// Scenario: cursor is at column 5, but append_chars starts at column 2
	// This should be downgraded because the ghost text would appear before cursor
	groups := []*Group{
		{
			Type:       "modification",
			RenderHint: "append_chars",
			BufferLine: 10,
			ColStart:   2,
			ColEnd:     8,
		},
	}

	ValidateRenderHintsForCursor(groups, 10, 5) // cursor at row 10, col 5

	assert.Equal(t, "", groups[0].RenderHint, "should downgrade append_chars when ColStart < cursorCol")
}

func TestValidateRenderHintsForCursor_KeepsAppendCharsAtOrAfterCursor(t *testing.T) {
	// Scenario: cursor is at column 2, append_chars starts at column 4
	// This should NOT be downgraded because ghost text appears after cursor
	groups := []*Group{
		{
			Type:       "modification",
			RenderHint: "append_chars",
			BufferLine: 10,
			ColStart:   4,
			ColEnd:     8,
		},
	}

	ValidateRenderHintsForCursor(groups, 10, 2) // cursor at row 10, col 2

	assert.Equal(t, "append_chars", groups[0].RenderHint, "should keep append_chars when ColStart >= cursorCol")
}

func TestValidateRenderHintsForCursor_IgnoresDifferentLine(t *testing.T) {
	// Scenario: append_chars on a different line than cursor
	// Should NOT be affected even if ColStart < cursorCol
	groups := []*Group{
		{
			Type:       "modification",
			RenderHint: "append_chars",
			BufferLine: 15, // different line
			ColStart:   2,
			ColEnd:     8,
		},
	}

	ValidateRenderHintsForCursor(groups, 10, 5) // cursor at row 10, col 5

	assert.Equal(t, "append_chars", groups[0].RenderHint, "should not affect groups on different lines")
}

func TestValidateRenderHintsForCursor_NeverDowngradesInlineDiff(t *testing.T) {
	// inline_diff renders with plain extmarks that never hide the cursor, so
	// the hint is kept regardless of where the cursor sits on the line.
	for _, cursorCol := range []int{0, 5, 8, 20} {
		groups := []*Group{
			{
				Type:       "modification",
				RenderHint: "inline_diff",
				BufferLine: 10,
				ColStart:   3,
				ColEnd:     10,
				Spans:      []InlineSpan{{OldStart: 3, OldEnd: 8, NewStart: 3, NewEnd: 10}},
			},
		}

		ValidateRenderHintsForCursor(groups, 10, cursorCol)

		assert.Equal(t, "inline_diff", groups[0].RenderHint, "inline_diff kept at any cursor col")
	}
}

func TestValidateRenderHintsForCursor_AppendAtExactPosition(t *testing.T) {
	// At exact cursor position (col 5), append_chars is kept because the
	// cursor is at the change start, not past it.
	appendGroup := &Group{
		Type:       "modification",
		RenderHint: "append_chars",
		BufferLine: 10,
		ColStart:   5,
		ColEnd:     10,
	}

	ValidateRenderHintsForCursor([]*Group{appendGroup}, 10, 5)

	assert.Equal(t, "append_chars", appendGroup.RenderHint, "append_chars at exact cursor position should keep hint")
}

func TestFinalizeStageGroups_AppendCharsOnNonCursorLine(t *testing.T) {
	// Scenario: cursor is on line 5 (buffer), completion appends text on lines 5 and 7.
	// Line 5 (cursor line) gets append_chars validated against cursor position.
	// Line 7 (non-cursor line) should keep append_chars unconditionally.
	changes := map[int]LineChange{
		1: {
			Type:       ChangeAppendChars,
			Content:    "    fmt.Println(\"hello\")",
			OldContent: "    ",
			ColStart:   4,
			ColEnd:     23,
		},
		3: {
			Type:       ChangeAppendChars,
			Content:    "    return nil",
			OldContent: "    ",
			ColStart:   4,
			ColEnd:     14,
		},
	}
	newLines := []string{
		"    fmt.Println(\"hello\")",
		"",
		"    return nil",
	}

	ctx := &StageContext{
		BufferStart:         5,
		CursorRow:           5,
		CursorCol:           4,
		LineNumToBufferLine: map[int]int{1: 5, 3: 7},
	}
	groups, _, _ := FinalizeStageGroups(changes, newLines, ctx)

	assert.Equal(t, 2, len(groups), "should have 2 groups (hinted changes stay separate)")

	// Line 5 (cursor line): ColStart == cursorCol, so hint is kept
	assert.Equal(t, 5, groups[0].BufferLine, "first group on cursor line")
	assert.Equal(t, "append_chars", groups[0].RenderHint, "cursor line keeps hint when ColStart >= cursorCol")

	// Line 7 (non-cursor line): hint always preserved
	assert.Equal(t, 7, groups[1].BufferLine, "second group on non-cursor line")
	assert.Equal(t, "append_chars", groups[1].RenderHint, "non-cursor line keeps append_chars")
	assert.Equal(t, 4, groups[1].ColStart, "non-cursor line ColStart")
	assert.Equal(t, 14, groups[1].ColEnd, "non-cursor line ColEnd")
}

func TestFinalizeStageGroups_CursorLineDowngradedNonCursorLineKept(t *testing.T) {
	// Scenario: cursor is at column 8 on line 10, within existing content that
	// has an append starting at column 12. The cursor is before the append start
	// but within old content, so the hint should be kept. A separate case tests
	// downgrading when the append would overlay the cursor.
	changes := map[int]LineChange{
		1: {
			Type:       ChangeAppendChars,
			Content:    "    result := compute(x, y)",
			OldContent: "    result :=",
			ColStart:   13,
			ColEnd:     27,
		},
		2: {
			Type:       ChangeAppendChars,
			Content:    "    return nil",
			OldContent: "    return",
			ColStart:   10,
			ColEnd:     14,
		},
	}
	newLines := []string{
		"    result := compute(x, y)",
		"    return nil",
	}

	ctx := &StageContext{
		BufferStart:         10,
		CursorRow:           10,
		CursorCol:           8, // within old content but before ColStart=13
		LineNumToBufferLine: map[int]int{1: 10, 2: 11},
	}
	groups, _, _ := FinalizeStageGroups(changes, newLines, ctx)

	assert.Equal(t, 2, len(groups), "should have 2 groups")

	// Line 10 (cursor line): cursor at 8 is within old content (len 13),
	// so validation applies. ColStart(13) < cursorCol(8) is false, hint kept.
	assert.Equal(t, 10, groups[0].BufferLine, "first group on cursor line")
	assert.Equal(t, "append_chars", groups[0].RenderHint, "cursor line hint kept")

	// Line 11 (non-cursor line): keeps append_chars
	assert.Equal(t, 11, groups[1].BufferLine, "second group on non-cursor line")
	assert.Equal(t, "append_chars", groups[1].RenderHint, "non-cursor line keeps hint")
}

func TestToLuaFormat_PureInsertionCursorFromGroups(t *testing.T) {
	// When ToLuaFormat is called without stage.CursorLine set (e.g. from
	// PrepareCompletion), cursor should be derived from groups, not from
	// Changes which may contain a spurious deletion from the local diff.
	stage := &Stage{
		Changes: map[int]LineChange{
			1: {Type: ChangeDeletion, Content: "existing line content"},
		},
		Groups: []*Group{
			{Type: "addition", StartLine: 1, EndLine: 1, BufferLine: 7, Lines: []string{""}},
		},
		Lines: []string{""},
	}

	result := ToLuaFormat(stage, 7)

	assert.Equal(t, 1, result["cursor_line"], "cursor_line should be 1 from addition group")
	assert.Equal(t, 0, result["cursor_col"], "cursor_col should be 0 for empty line")
}

func TestToLuaFormat_InlineDiffEmitsSpans(t *testing.T) {
	stage := &Stage{
		Groups: []*Group{
			{
				Type: "modification", StartLine: 1, EndLine: 1, BufferLine: 4,
				Lines: []string{"Hello there"}, OldLines: []string{"Hello world"},
				RenderHint: "inline_diff", ColStart: 6, ColEnd: 11,
				Spans: []InlineSpan{{OldStart: 6, OldEnd: 11, NewStart: 6, NewEnd: 11}},
			},
		},
		Lines: []string{"Hello there"},
	}

	result := ToLuaFormat(stage, 4)

	group := result["groups"].([]map[string]any)[0]
	assert.Equal(t, "inline_diff", group["render_hint"], "render hint")

	_, hasColStart := group["col_start"]
	assert.False(t, hasColStart, "inline_diff omits group-level col_start")

	spans := group["spans"].([]map[string]any)
	assert.Equal(t, 1, len(spans), "span count")
	assert.Equal(t, 6, spans[0]["col_start"], "span col_start")
	assert.Equal(t, 11, spans[0]["col_end"], "span col_end")
	assert.Equal(t, 6, spans[0]["new_col_start"], "span new_col_start")
	assert.Equal(t, 11, spans[0]["new_col_end"], "span new_col_end")
}

func TestToLuaFormat_AppendCharsEmitsCols(t *testing.T) {
	stage := &Stage{
		Groups: []*Group{
			{
				Type: "modification", StartLine: 1, EndLine: 1, BufferLine: 2,
				Lines: []string{"hello world"}, OldLines: []string{"hello"},
				RenderHint: "append_chars", ColStart: 5, ColEnd: 11,
			},
		},
		Lines: []string{"hello world"},
	}

	result := ToLuaFormat(stage, 2)

	group := result["groups"].([]map[string]any)[0]
	assert.Equal(t, "append_chars", group["render_hint"], "render hint")
	assert.Equal(t, 5, group["col_start"], "col_start")
	assert.Equal(t, 11, group["col_end"], "col_end")

	_, hasSpans := group["spans"]
	assert.False(t, hasSpans, "append_chars has no spans")
}

func TestToLuaFormat_UsesPrecomputedCursor(t *testing.T) {
	// When stage has CursorLine set (from staging pipeline), use it directly.
	stage := &Stage{
		CursorLine: 3,
		CursorCol:  5,
		Groups:     []*Group{{Type: "addition", StartLine: 1, EndLine: 3, BufferLine: 10, Lines: []string{"a", "b", "c"}}},
		Lines:      []string{"a", "b", "c"},
	}

	result := ToLuaFormat(stage, 10)

	assert.Equal(t, 3, result["cursor_line"], "should use precomputed cursor_line")
	assert.Equal(t, 5, result["cursor_col"], "should use precomputed cursor_col")
}
