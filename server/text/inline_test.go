package text

import (
	"strings"
	"testing"

	"cursortab/assert"
)

func tokenTexts(s string) []string {
	var texts []string
	for _, tok := range tokenizeInline(s) {
		texts = append(texts, s[tok.Start:tok.End])
	}
	return texts
}

func TestTokenizeInlineWordsAndPunctuation(t *testing.T) {
	assert.Equal(t, []string{"foo_bar1", " ", "baz"}, tokenTexts("foo_bar1 baz"), "words with underscore and digits")
	assert.Equal(t, []string{"a", ".", "b", "(", "c", ")"}, tokenTexts("a.b(c)"), "punctuation as single tokens")
	assert.Equal(t, []string{" ", " ", "x"}, tokenTexts("  x"), "each space is its own token")
}

func TestTokenizeInlineUTF8(t *testing.T) {
	assert.Equal(t, []string{"héllo", " ", "wörld"}, tokenTexts("héllo wörld"), "multibyte letters join words")
	assert.Equal(t, []string{"a", " ", "🚀", " ", "b"}, tokenTexts("a 🚀 b"), "emoji is a single token")

	tokens := tokenizeInline("a 🚀 b")
	assert.Equal(t, inlineToken{Start: 2, End: 6}, tokens[2], "emoji token spans 4 bytes")
}

func TestInlineDiffSpansWordReplacement(t *testing.T) {
	spans := InlineDiffSpans("Hello world", "Hello there")

	assert.Equal(t, 1, len(spans), "span count")
	assert.Equal(t, InlineSpan{OldStart: 6, OldEnd: 11, NewStart: 6, NewEnd: 11}, spans[0], "span")
}

func TestInlineDiffSpansPureDeletion(t *testing.T) {
	spans := InlineDiffSpans("keep remove keep", "keep keep")

	assert.Equal(t, 1, len(spans), "span count")
	assert.Equal(t, InlineSpan{OldStart: 5, OldEnd: 12, NewStart: 5, NewEnd: 5}, spans[0], "deleted range with empty new range")
}

func TestInlineDiffSpansPureInsertion(t *testing.T) {
	spans := InlineDiffSpans("foo(a)", "foo(a, b)")

	assert.Equal(t, 1, len(spans), "span count")
	assert.Equal(t, InlineSpan{OldStart: 5, OldEnd: 5, NewStart: 5, NewEnd: 8}, spans[0], "insertion point with new range")
}

func TestInlineDiffSpansDistantEditsStaySeparate(t *testing.T) {
	spans := InlineDiffSpans("aa bb cc dd ee ff gg", "xx bb cc dd ee ff yy")

	assert.Equal(t, 2, len(spans), "edits separated by more than InlineInterHunkContext tokens stay separate")
	assert.Equal(t, InlineSpan{OldStart: 0, OldEnd: 2, NewStart: 0, NewEnd: 2}, spans[0], "first span")
	assert.Equal(t, InlineSpan{OldStart: 18, OldEnd: 20, NewStart: 18, NewEnd: 20}, spans[1], "second span")
}

func TestInlineDiffSpansNearbyEditsMerge(t *testing.T) {
	spans := InlineDiffSpans("aa bb cc", "xx bb yy")

	assert.Equal(t, 1, len(spans), "edits within InlineInterHunkContext tokens merge")
	assert.Equal(t, InlineSpan{OldStart: 0, OldEnd: 8, NewStart: 0, NewEnd: 8}, spans[0], "merged span covers the gap")
}

func TestInlineDiffSpansUTF8(t *testing.T) {
	spans := InlineDiffSpans("Hello 🎉 world", "Hello 🚀 world")

	assert.Equal(t, 1, len(spans), "span count")
	assert.Equal(t, InlineSpan{
		OldStart: len("Hello "),
		OldEnd:   len("Hello 🎉"),
		NewStart: len("Hello "),
		NewEnd:   len("Hello 🚀"),
	}, spans[0], "byte offsets respect UTF-8 boundaries")
}

func TestInlineDiffSpansTokenCapReturnsNil(t *testing.T) {
	long := strings.Repeat("word ", InlineMaxTokens)

	assert.Nil(t, InlineDiffSpans(long, long+"tail"), "over-long lines fall back")
}

func TestCategorizeLineChangeGates(t *testing.T) {
	tests := []struct {
		name     string
		oldLine  string
		newLine  string
		expected ChangeType
	}{
		{"light word edit", "Hello world", "Hello there", ChangeInlineDiff},
		{"interior deletion", "Hello world John", "Hello John", ChangeInlineDiff},
		{"mostly rewritten", "start middle end", "beginning middle finish extra", ChangeModification},
		{"pure append", "Hello", "Hello world", ChangeAppendChars},
		{"empty old line", "", "content", ChangeAppendChars},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changeType, _, _, spans := categorizeLineChange(tt.oldLine, tt.newLine)
			assert.Equal(t, tt.expected, changeType, "change type")
			assert.Equal(t, tt.expected == ChangeInlineDiff, len(spans) > 0, "spans only for inline_diff")
		})
	}
}

func TestCategorizeLineChangeInlineEnvelope(t *testing.T) {
	changeType, colStart, colEnd, spans := categorizeLineChange("Hello world", "Hello there")

	assert.Equal(t, ChangeInlineDiff, changeType, "change type")
	assert.Equal(t, 6, colStart, "envelope start")
	assert.Equal(t, 11, colEnd, "envelope end in new-line coordinates")
	assert.Equal(t, 1, len(spans), "span count")
}
