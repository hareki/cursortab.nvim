package text

import "sort"

// Group represents consecutive changes of the same type for rendering
type Group struct {
	Type      string   // "modification", "addition", "deletion"
	StartLine int      // 1-indexed, relative to stage content
	EndLine   int      // 1-indexed, inclusive
	Lines     []string // New content
	OldLines  []string // Old content (modifications only)

	// BufferLine is the 1-indexed absolute buffer position for rendering.
	// Computed by staging/grouping using LineMapping.GetBufferLine for correct coordinate mapping.
	BufferLine int

	// Character-level rendering hints (single-line only)
	RenderHint string       // "", "append_chars", "inline_diff", "stacked"
	ColStart   int          // For character-level changes
	ColEnd     int          // For character-level changes
	Spans      []InlineSpan // Word-level edit spans (inline_diff only)
}

// GroupChanges groups consecutive same-type changes for efficient rendering.
// Returns groups sorted by StartLine. Deletions are not grouped (they don't render as content).
// Group content is populated from change.Content and change.OldContent fields.
func GroupChanges(changes map[int]LineChange) []*Group {
	if len(changes) == 0 {
		return nil
	}

	var lineNums []int
	for lineNum := range changes {
		lineNums = append(lineNums, lineNum)
	}

	if len(lineNums) == 0 {
		return nil
	}

	sort.Ints(lineNums)

	var groups []*Group
	var currentGroup *Group

	for _, lineNum := range lineNums {
		change := changes[lineNum]
		groupType := change.Type.GroupType()
		changeHasHint := change.Type.RenderHint() != ""

		// Determine if we should extend current group or start new one.
		// Render hints are single-line only: a hinted change can only extend a
		// modification group if the incoming change itself has NO hint (the group's
		// existing hint is cleared on extend). Two consecutive hinted modifications
		// stay separate so each gets its own char-level rendering.
		isModification := groupType == "modification"
		shouldStartNew := currentGroup == nil ||
			currentGroup.Type != groupType ||
			lineNum != currentGroup.EndLine+1 ||
			(!isModification && (currentGroup.RenderHint != "" || changeHasHint)) ||
			(isModification && changeHasHint)

		if shouldStartNew {
			// Flush current group and start new
			if currentGroup != nil {
				groups = append(groups, currentGroup)
			}
			currentGroup = &Group{
				Type:      groupType,
				StartLine: lineNum,
				EndLine:   lineNum,
				Lines:     []string{change.Content},
			}
			if isModification {
				currentGroup.OldLines = []string{change.OldContent}
			}
			// Set RenderHint for this single-line group
			setRenderHint(currentGroup, change)
		} else {
			// Extend current group
			currentGroup.EndLine = lineNum
			currentGroup.Lines = append(currentGroup.Lines, change.Content)
			if isModification {
				currentGroup.OldLines = append(currentGroup.OldLines, change.OldContent)
				// Multi-line modification groups cannot use a single-line render hint
				currentGroup.RenderHint = ""
				currentGroup.ColStart = 0
				currentGroup.ColEnd = 0
				currentGroup.Spans = nil
			}
		}
	}

	// Flush final group
	if currentGroup != nil {
		groups = append(groups, currentGroup)
	}

	return groups
}

// CopyGroups returns a deep copy of the given groups slice.
// Each Group struct is copied so that mutations to the copy don't affect the original.
func CopyGroups(groups []*Group) []*Group {
	if groups == nil {
		return nil
	}
	result := make([]*Group, len(groups))
	for i, g := range groups {
		cp := *g
		if g.Lines != nil {
			cp.Lines = make([]string, len(g.Lines))
			copy(cp.Lines, g.Lines)
		}
		if g.OldLines != nil {
			cp.OldLines = make([]string, len(g.OldLines))
			copy(cp.OldLines, g.OldLines)
		}
		if g.Spans != nil {
			cp.Spans = make([]InlineSpan, len(g.Spans))
			copy(cp.Spans, g.Spans)
		}
		result[i] = &cp
	}
	return result
}

// setRenderHint sets the render hint for character-level optimizations
func setRenderHint(group *Group, change LineChange) {
	group.RenderHint = change.Type.RenderHint()
	if group.RenderHint != "" {
		group.ColStart = change.ColStart
		group.ColEnd = change.ColEnd
		group.Spans = change.Spans
	} else {
		group.ColStart = 0
		group.ColEnd = 0
		group.Spans = nil
	}
}

// ValidateRenderHintsForCursor downgrades append_chars to regular modification
// when the cursor would be hidden under its overlay. Only append_chars renders
// overlay-backed content that can cover the cursor; inline_diff uses plain
// extmarks (deletion highlights and inline virtual text) that leave the cursor
// on real buffer text, so it never needs downgrading.
func ValidateRenderHintsForCursor(groups []*Group, cursorRow, cursorCol int) {
	for _, g := range groups {
		if g.BufferLine != cursorRow || g.RenderHint != "append_chars" {
			continue
		}
		// Skip when cursor is at or past end of old content — append only
		// adds after existing text, so there's nothing for the overlay to hide.
		if len(g.OldLines) == 1 && cursorCol >= len(g.OldLines[0]) {
			continue
		}
		if g.ColStart < cursorCol {
			g.RenderHint = ""
		}
	}
}

// StageContext provides context for finalizing groups within a stage
type StageContext struct {
	BufferStart         int         // Stage's buffer start line (1-indexed)
	CursorRow           int         // Current cursor row (1-indexed)
	CursorCol           int         // Current cursor col (0-indexed)
	LineNumToBufferLine map[int]int // Pre-computed relative line -> buffer line
}

// FinalizeStageGroups creates groups, populates BufferLine for each group,
// validates render hints, and computes cursor position.
// Returns (groups, cursorLine, cursorCol).
func FinalizeStageGroups(changes map[int]LineChange, newLines []string, ctx *StageContext) ([]*Group, int, int) {
	groups := GroupChanges(changes)

	// Set BufferLine for each group using the pre-computed mapping
	for _, g := range groups {
		if bufLine := ctx.LineNumToBufferLine[g.StartLine]; bufLine > 0 {
			g.BufferLine = bufLine
		} else {
			g.BufferLine = ctx.BufferStart + g.StartLine - 1
		}
	}

	ValidateRenderHintsForCursor(groups, ctx.CursorRow, ctx.CursorCol)
	cursorLine, cursorCol := CalculateCursorPosition(changes, newLines)
	return groups, cursorLine, cursorCol
}

// CalculateCursorPosition computes optimal cursor position from changes
// Priority: modifications > additions > append/replace/delete chars > deletions
// Returns (line, col) where line is 1-indexed and col is 0-indexed
// Returns (-1, -1) if no cursor positioning is needed
func CalculateCursorPosition(changes map[int]LineChange, newLines []string) (int, int) {
	if len(changes) == 0 {
		return -1, -1
	}

	// Pick the latest (highest) changed line, excluding deletions
	targetLine := -1
	for lineNum, change := range changes {
		if change.Type != ChangeDeletion && lineNum > targetLine {
			targetLine = lineNum
		}
	}

	if targetLine <= 0 {
		return -1, -1
	}

	if targetLine > len(newLines) {
		targetLine = len(newLines)
	}
	if targetLine <= 0 {
		return -1, -1
	}

	// For character-level changes, position at the end of the changed
	// envelope: ColEnd is in new-line coordinates for both append_chars and
	// inline_diff (for a pure deletion the envelope is empty, so ColEnd is
	// the deletion point). For full-line changes, position at end of line.
	change, exists := changes[targetLine]
	if exists && change.Type.IsCharacterLevel() {
		return targetLine, change.ColEnd
	}

	col := len(newLines[targetLine-1])
	return targetLine, col
}
