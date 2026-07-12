package text

import (
	"unicode"
	"unicode/utf8"
)

// InlineSpan is one word-level edit within a modified line, expressed as
// half-open byte ranges in both the old and new line.
type InlineSpan struct {
	OldStart int // delete oldLine[OldStart:OldEnd); OldStart == OldEnd means pure insertion point
	OldEnd   int
	NewStart int // insert newLine[NewStart:NewEnd) at OldEnd; NewStart == NewEnd means pure deletion
	NewEnd   int
}

// inlineToken is a token's byte range within its source line.
type inlineToken struct {
	Start int
	End   int
}

// tokenHunk is a contiguous run of differing tokens between the two token
// streams: ai/bi index the first differing token (old/new), ac/bc the counts.
type tokenHunk struct {
	ai, ac int
	bi, bc int
}

func isInlineWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// tokenizeInline splits a line into word tokens (runs of letters, digits, and
// underscores) and single-rune tokens for everything else, mirroring how
// Neovim's iskeyword-based word splitting groups characters.
func tokenizeInline(s string) []inlineToken {
	var tokens []inlineToken
	wordStart := -1
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if isInlineWordRune(r) {
			if wordStart < 0 {
				wordStart = i
			}
		} else {
			if wordStart >= 0 {
				tokens = append(tokens, inlineToken{Start: wordStart, End: i})
				wordStart = -1
			}
			tokens = append(tokens, inlineToken{Start: i, End: i + size})
		}
		i += size
	}
	if wordStart >= 0 {
		tokens = append(tokens, inlineToken{Start: wordStart, End: len(s)})
	}
	return tokens
}

// diffTokenHunks computes differing token runs between the two token streams
// using an LCS dynamic program over token text equality.
func diffTokenHunks(oldLine, newLine string, oldToks, newToks []inlineToken) []tokenHunk {
	n, m := len(oldToks), len(newToks)
	equal := func(i, j int) bool {
		return oldLine[oldToks[i].Start:oldToks[i].End] == newLine[newToks[j].Start:newToks[j].End]
	}

	// dp[i][j] = LCS length of oldToks[i:] vs newToks[j:]
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if equal(i, j) {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}

	var hunks []tokenHunk
	i, j := 0, 0
	hunkAI, hunkBI := -1, -1
	flush := func() {
		if hunkAI >= 0 && (i > hunkAI || j > hunkBI) {
			hunks = append(hunks, tokenHunk{ai: hunkAI, ac: i - hunkAI, bi: hunkBI, bc: j - hunkBI})
		}
		hunkAI, hunkBI = -1, -1
	}
	for i < n || j < m {
		if i < n && j < m && equal(i, j) {
			flush()
			i++
			j++
			continue
		}
		if hunkAI < 0 {
			hunkAI, hunkBI = i, j
		}
		if j >= m || (i < n && dp[i+1][j] >= dp[i][j+1]) {
			i++
		} else {
			j++
		}
	}
	flush()
	return hunks
}

// mergeTokenHunks coalesces adjacent hunks separated by at most
// InlineInterHunkContext unchanged tokens, so nearby edits render as one span.
func mergeTokenHunks(hunks []tokenHunk) []tokenHunk {
	if len(hunks) == 0 {
		return hunks
	}
	merged := []tokenHunk{hunks[0]}
	for _, h := range hunks[1:] {
		last := &merged[len(merged)-1]
		if h.ai-(last.ai+last.ac) <= InlineInterHunkContext {
			last.ac = h.ai + h.ac - last.ai
			last.bc = h.bi + h.bc - last.bi
		} else {
			merged = append(merged, h)
		}
	}
	return merged
}

// spanBounds converts a token run to a byte range; an empty run maps to the
// point at the end of the preceding token (or the line start).
func spanBounds(toks []inlineToken, start, count int) (int, int) {
	if count > 0 {
		return toks[start].Start, toks[start+count-1].End
	}
	point := 0
	if start > 0 {
		point = toks[start-1].End
	}
	return point, point
}

// InlineDiffSpans computes word-level edit spans between two differing lines.
// Spans are sorted ascending and non-overlapping. Returns nil when either line
// exceeds InlineMaxTokens, signalling the caller to fall back to a full-line
// modification.
func InlineDiffSpans(oldLine, newLine string) []InlineSpan {
	oldToks := tokenizeInline(oldLine)
	newToks := tokenizeInline(newLine)
	if len(oldToks) > InlineMaxTokens || len(newToks) > InlineMaxTokens {
		return nil
	}

	hunks := mergeTokenHunks(diffTokenHunks(oldLine, newLine, oldToks, newToks))
	spans := make([]InlineSpan, 0, len(hunks))
	for _, h := range hunks {
		oldStart, oldEnd := spanBounds(oldToks, h.ai, h.ac)
		newStart, newEnd := spanBounds(newToks, h.bi, h.bc)
		spans = append(spans, InlineSpan{
			OldStart: oldStart,
			OldEnd:   oldEnd,
			NewStart: newStart,
			NewEnd:   newEnd,
		})
	}
	return spans
}

// inlineInsertRatio is the fraction of the new line made up of inserted text.
func inlineInsertRatio(spans []InlineSpan, newLine string) float64 {
	if len(newLine) == 0 {
		return 0
	}
	inserted := 0
	for _, s := range spans {
		inserted += s.NewEnd - s.NewStart
	}
	return float64(inserted) / float64(len(newLine))
}
