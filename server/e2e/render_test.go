package e2e

import (
	"strings"
	"testing"

	"cursortab/assert"
)

func TestRenderLineMapsByteColumnsToUTF8Characters(t *testing.T) {
	var b strings.Builder

	old := "Hello 🚀 world"
	new := "Hello 🎉 world"
	RenderLine(&b, 1, old, LineHighlight{
		RenderHint: "inline_diff",
		NewText:    new,
		Spans: []SpanInfo{{
			ColStart:    len("Hello "),
			ColEnd:      len("Hello 🚀"),
			NewColStart: len("Hello "),
			NewColEnd:   len("Hello 🎉"),
		}},
	}, -1)

	assert.True(t, strings.Contains(b.String(), `Hello <span class="del-hl">🚀</span><span class="add-hl">🎉</span> world`), "highlighted emoji")
}

func TestRenderLineInlineDiffMultipleSpans(t *testing.T) {
	var b strings.Builder

	old := "foo(alpha, beta)"
	new := "bar(alpha, gamma)"
	RenderLine(&b, 1, old, LineHighlight{
		RenderHint: "inline_diff",
		NewText:    new,
		Spans: []SpanInfo{
			{ColStart: 0, ColEnd: 3, NewColStart: 0, NewColEnd: 3},
			{ColStart: 11, ColEnd: 15, NewColStart: 11, NewColEnd: 16},
		},
	}, -1)

	out := b.String()
	assert.True(t, strings.Contains(out, `<span class="del-hl">foo</span><span class="add-hl">bar</span>(alpha, `), "first span")
	assert.True(t, strings.Contains(out, `<span class="del-hl">beta</span><span class="add-hl">gamma</span>)`), "second span")
}

func TestRenderLineInlineDiffPureDeletion(t *testing.T) {
	var b strings.Builder

	RenderLine(&b, 1, "keep remove keep", LineHighlight{
		RenderHint: "inline_diff",
		NewText:    "keep keep",
		Spans:      []SpanInfo{{ColStart: 5, ColEnd: 12, NewColStart: 5, NewColEnd: 5}},
	}, -1)

	out := b.String()
	assert.True(t, strings.Contains(out, `keep <span class="del-hl">remove </span>keep`), "deletion span")
	assert.True(t, !strings.Contains(out, "add-hl"), "no insertion rendered")
}
