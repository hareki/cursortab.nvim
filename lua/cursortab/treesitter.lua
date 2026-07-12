local M = {}

local scope_types = {
	function_declaration = true,
	function_definition = true,
	method_declaration = true,
	method_definition = true,
	class_declaration = true,
	class_definition = true,
	type_declaration = true,
	type_spec = true,
	impl_item = true,
	func_literal = true,
	arrow_function = true,
	function_item = true,
}

local import_types = {
	import_declaration = true,
	import_statement = true,
	use_declaration = true,
	preproc_include = true,
	import_spec = true,
	import_spec_list = true,
}

---Get treesitter scope context around the cursor position.
---@param bufnr integer Buffer number
---@param row integer 1-indexed cursor row
---@param col integer 0-indexed cursor column
---@param max_siblings integer Maximum sibling nodes to return
---@return table|nil
function M.get_context(bufnr, row, col, max_siblings)
	row = row - 1 -- convert to 0-indexed

	local ok, parser = pcall(vim.treesitter.get_parser, bufnr)
	if not ok or not parser then
		return nil
	end

	local cursor_node = vim.treesitter.get_node({ bufnr = bufnr, pos = { row, col } })
	if not cursor_node then
		return {}
	end

	-- Walk up to find enclosing scope
	local enclosing = nil
	local node = cursor_node ---@type TSNode?
	while node do
		if scope_types[node:type()] then
			enclosing = node
			break
		end
		node = node:parent()
	end

	local enclosing_sig = ""
	if enclosing then
		local start_row = enclosing:start()
		local line = vim.api.nvim_buf_get_lines(bufnr, start_row, start_row + 1, false)[1] or ""
		enclosing_sig = line
	end

	-- Get sibling scope nodes from the enclosing scope's parent
	local siblings = {}
	local parent = enclosing and enclosing:parent()
	if parent then
		for child in parent:iter_children() do
			if scope_types[child:type()] and child ~= enclosing then
				local s_row = child:start()
				local line = vim.api.nvim_buf_get_lines(bufnr, s_row, s_row + 1, false)[1] or ""
				local name = ""
				local name_node = child:field("name")[1]
				if name_node then
					name = vim.treesitter.get_node_text(name_node, bufnr)
				end
				table.insert(siblings, { name = name, signature = line, line = s_row + 1 })
			end
		end
		if #siblings > max_siblings then
			table.sort(siblings, function(a, b)
				return math.abs(a.line - 1 - row) < math.abs(b.line - 1 - row)
			end)
			local trimmed = {}
			for i = 1, max_siblings do
				trimmed[i] = siblings[i]
			end
			siblings = trimmed
		end
	end

	-- Collect imports from root
	local imports = {}
	local root = parser:trees()[1]:root()
	for child in root:iter_children() do
		if import_types[child:type()] then
			local s_row, _, e_row = child:start(), nil, child:end_()
			local lines = vim.api.nvim_buf_get_lines(bufnr, s_row, e_row + 1, false)
			table.insert(imports, table.concat(lines, "\n"))
		end
	end

	-- Collect ancestor node line ranges (innermost to outermost)
	-- Used by providers to snap editable/context regions to syntax boundaries
	local syntax_ranges = {}
	node = cursor_node
	while node do
		local s_row = node:start()
		local e_row = node:end_()
		-- Only include nodes that span multiple lines (meaningful boundaries)
		if e_row > s_row then
			table.insert(syntax_ranges, { start_line = s_row + 1, end_line = e_row + 1 })
		end
		node = node:parent()
	end

	return {
		enclosing_signature = enclosing_sig,
		siblings = siblings,
		imports = imports,
		syntax_ranges = syntax_ranges,
	}
end

---@class HLChunk
---@field text string
---@field hl string|nil Treesitter highlight group, nil for unhighlighted text

---Treesitter-highlight a single line (as a detached string), split into chunks
---at highlight boundaries. Returns nil when no parser or highlight query is
---available for the filetype, so callers can fall back to plain highlighting.
---@param line string
---@param ft string
---@return HLChunk[]|nil
function M.line_chunks(line, ft)
	if line == "" or not ft or ft == "" then
		return nil
	end

	local lang = vim.treesitter.language.get_lang(ft)
	if not lang then
		return nil
	end

	local ok, parser = pcall(vim.treesitter.get_string_parser, line, lang)
	if not ok or not parser then
		return nil
	end
	if not pcall(parser.parse, parser, true) then
		return nil
	end

	-- Per-byte highlight index (1-indexed); higher-priority captures win
	---@type table<integer, {hl: string, priority: integer}>
	local index = {}
	local has_query = false
	parser:for_each_tree(function(tstree, tree)
		if not tstree then
			return
		end
		local query_ok, query = pcall(vim.treesitter.query.get, tree:lang(), "highlights")
		if not query_ok or not query then
			return
		end
		has_query = true
		for capture, node, metadata in query:iter_captures(tstree:root(), line) do
			local name = query.captures[capture]
			if name ~= "spell" and name ~= "nospell" and name ~= "conceal" then
				local _, start_col, _, end_col = node:range()
				local priority = tonumber(metadata.priority or (metadata[capture] and metadata[capture].priority))
					or 100
				local hl = "@" .. name .. "." .. tree:lang()
				for i = start_col + 1, math.min(end_col, #line) do
					local existing = index[i]
					if not existing or priority >= existing.priority then
						index[i] = { hl = hl, priority = priority }
					end
				end
			end
		end
	end)
	if not has_query then
		return nil
	end

	-- Coalesce per-byte highlights into contiguous chunks
	---@type HLChunk[]
	local chunks = {}
	local from = 1
	local current = index[1] and index[1].hl
	for col = 2, #line + 1 do
		local hl = index[col] and index[col].hl
		if col > #line or hl ~= current then
			table.insert(chunks, { text = line:sub(from, col - 1), hl = current })
			from = col
			current = hl
		end
	end
	return chunks
end

---Slice a byte range out of highlighted chunks as extmark virt_text, combining
---each chunk's treesitter group with a background group. Falls back to the raw
---slice with just the background group when chunks is nil.
---@param chunks HLChunk[]|nil From M.line_chunks over the full line
---@param line string Full line the chunks were computed from
---@param start_col integer 0-based byte column, inclusive
---@param end_col integer 0-based byte column, exclusive
---@param bg string Background highlight group (e.g. "CursorTabAddition")
---@return [string, string|string[]][]
function M.slice_chunks(chunks, line, start_col, end_col, bg)
	if not chunks then
		return { { string.sub(line, start_col + 1, end_col), bg } }
	end

	---@type [string, string|string[]][]
	local result = {}
	local pos = 0 -- 0-based byte offset of the current chunk's start
	for _, chunk in ipairs(chunks) do
		local chunk_end = pos + #chunk.text
		local from = math.max(pos, start_col)
		local to = math.min(chunk_end, end_col)
		if to > from then
			-- The bg group comes last so its background wins over the
			-- (foreground-only) treesitter group
			table.insert(result, { string.sub(line, from + 1, to), chunk.hl and { chunk.hl, bg } or bg })
		end
		pos = chunk_end
		if pos >= end_col then
			break
		end
	end

	if #result == 0 then
		return { { string.sub(line, start_col + 1, end_col), bg } }
	end
	return result
end

---Get treesitter node types from cursor to root.
---@param bufnr integer Buffer number
---@param row integer 1-indexed cursor row
---@param col integer 0-indexed cursor column
---@return string[]
function M.cursor_scopes(bufnr, row, col)
	row = row - 1 -- convert to 0-indexed
	-- Query the character just typed, not the position after it
	if col > 0 then
		col = col - 1
	end

	local node = vim.treesitter.get_node({ bufnr = bufnr, pos = { row, col } })
	if not node then
		return {}
	end

	local scopes = {}
	while node do
		scopes[#scopes + 1] = node:type()
		node = node:parent()
	end
	return scopes
end

return M
