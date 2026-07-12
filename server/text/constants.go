package text

const (
	// SimilarityThreshold is the minimum similarity score for considering
	// two lines as corresponding (modification vs addition/deletion).
	// Below this threshold, lines are treated as unrelated.
	SimilarityThreshold = 0.3

	// InlineMaxInsertRatio gates inline_diff rendering: a modification renders
	// as a word-level inline diff only when the total inserted bytes across all
	// spans divided by len(newLine) is below this ratio. Beyond it, so much of
	// the line is new that a full-line modification reads better.
	InlineMaxInsertRatio = 0.5

	// InlineInterHunkContext is the maximum number of unchanged tokens between
	// two adjacent token hunks for them to be merged into a single span. Merging
	// avoids confetti rendering when several small edits sit close together.
	InlineInterHunkContext = 4

	// InlineMaxTokens caps the per-line token count for the O(n*m) token diff.
	// Lines beyond this fall back to full-line modification rendering.
	InlineMaxTokens = 512
)
