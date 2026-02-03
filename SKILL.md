---
name: gh-pr-review
description: View and manage inline GitHub PR review comments with full thread context from the terminal
---

# gh-pr-review

A GitHub CLI extension that provides complete inline PR review comment access from the terminal with LLM-friendly JSON output.

## When to Use

Use this skill when you need to:

- View inline review comments and threads on a pull request
- Reply to review comments programmatically
- Resolve or unresolve review threads
- Create and submit PR reviews with inline comments
- Edit comments in pending reviews
- Delete comments from pending reviews
- Access PR review context for automated workflows
- Filter reviews by state, reviewer, or resolution status

This tool is particularly useful for:
- Automated PR review workflows
- LLM-based code review agents
- Terminal-based PR review processes
- Getting structured review data without multiple API calls

## Installation

First, ensure the extension is installed:

```sh
gh extension install agynio/gh-pr-review
```

## Core Commands

### 1. View All Reviews and Threads

Get complete review context with inline comments and thread replies:

```sh
gh pr-review review view -R owner/repo --pr <number>
```

**Useful filters:**
- `--unresolved` - Only show unresolved threads
- `--reviewer <login>` - Filter by specific reviewer
- `--states <APPROVED|CHANGES_REQUESTED|COMMENTED|DISMISSED>` - Filter by review state
- `--tail <n>` - Keep only last n replies per thread
- `--not_outdated` - Exclude outdated threads

**Output:** Structured JSON with reviews, comments, thread_ids, and resolution status.

### 2. Reply to Review Threads

Reply to an existing inline comment thread:

```sh
gh pr-review comments reply <pr-number> -R owner/repo \
  --thread-id <PRRT_...> \
  --body "Your reply message"
```

### 3. List Review Threads

Get a filtered list of review threads:

```sh
gh pr-review threads list -R owner/repo <pr-number> --unresolved --mine
```

### 4. Resolve/Unresolve Threads

Mark threads as resolved:

```sh
gh pr-review threads resolve -R owner/repo <pr-number> --thread-id <PRRT_...>
```

### 5. Create and Submit Reviews

Start a pending review:

```sh
gh pr-review review --start -R owner/repo <pr-number>
```

Add inline comment (recommended - explicit single line):

```sh
gh pr-review review --add-comment \
  --review-id <PRR_...> \
  --path <file-path> \
  --start-line <line-number> \
  --line <line-number> \
  --body "Your comment" \
  -R owner/repo <pr-number>
```

Add multi-line comment (range):

```sh
gh pr-review review --add-comment \
  --review-id <PRR_...> \
  --path <file-path> \
  --start-line <start-line> \
  --line <end-line> \
  --start-side <LEFT|RIGHT> \
  --side <LEFT|RIGHT> \
  --body "Your comment" \
  -R owner/repo <pr-number>
```

> **Note**: While `--line` alone works for single-line comments, using both `--start-line` and `--line` with the same value is recommended for clarity.

**Comment positioning parameters:**

- `--line` (required) - The line number where the comment ends. **Important:** This must be a line number within the PR diff hunk, NOT the absolute line number in the original file. See line number calculation below.
- `--side` (optional) - Which version of the code to comment on: `LEFT` (original) or `RIGHT` (modified). Default: `RIGHT`
- `--start-line` (optional) - The starting line number for multi-line comments. When specified, `--line` becomes the end line
  - **Best practice**: Even for single-line comments, consider using `--start-line` with the same value as `--line`. This makes your intent explicit and avoids confusion about whether you're targeting a single line or a range.
- `--start-side` (optional) - Which side the start line is on. Use when `--start-line` is specified

**Line Number Calculation:**

The `--line` value must be a **diff position** (1-based index within the diff hunk), NOT the absolute line number in the original file.

### Step-by-Step Guide

**Step 1: Get the patch for the file**
```sh
gh api repos/OWNER/REPO/pulls/PR/files --jq '.[] | select(.filename == "path/to/file.ts") | .patch'
```

**Step 2: Analyze the diff hunk header**
```
@@ -oldStart,oldCount +newStart,newCount @@
```
- For **new files** (`-0,0 +1,N`): Line numbers start at 1
- For **modified files**: Count lines from the start of the hunk (first `+` line = position 1)

**Step 3: Count to your target line**
- Count only lines starting with `+` (added/modified lines)
- The count is 1-based within each hunk

### Examples

**Example 1: New file**
```diff
@@ -0,0 +1,10 @@
+line 1   <- position 1
+line 2   <- position 2
+line 3   <- position 3
```
To comment on "line 3": Use `--line 3`

**Example 2: Modified file**
```diff
@@ -100,5 +100,15 @@
 existing code line 100
 existing code line 101
-existing code line 102
+new code A    <- position 1
+new code B    <- position 2 (comment here: --line 2)
+new code C    <- position 3
 existing code line 103
```
To comment on "new code B": Use `--line 2`

**Example 3: Multiple hunks**
Each hunk restarts counting at 1. If your target is in the second hunk, use the position within that hunk.

### Quick Reference Table

| Diff Header | File Type | To comment on... | Use |
|-------------|-----------|------------------|-----|
| `@@ -0,0 +1,173 @@` | New file | Line 80 of new file | `--line 80` |
| `@@ -224,6 +224,112 @@` | Modified | 6th added line | `--line 6` |

### Common Mistakes

❌ **Wrong**: Using absolute file line numbers
❌ **Wrong**: Counting from the original file's line numbers
❌ **Wrong**: Counting `-` lines (removed lines)

✅ **Correct**: Count only `+` lines from the start of the hunk
✅ **Correct**: Each hunk restarts at position 1

**Debug tip**: If you get "line number is invalid" error, your line number is outside the hunk range. Re-check the patch and count again.

Edit a comment in pending review (requires comment node ID PRRC_...):

```sh
gh pr-review review --edit-comment \
  --comment-id <PRRC_...> \
  --body "Updated comment text" \
  -R owner/repo <pr-number>
```

Delete a comment from pending review (requires comment node ID PRRC_...):

```sh
gh pr-review review --delete-comment \
  --comment-id <PRRC_...> \
  -R owner/repo <pr-number>
```

Submit the review:

```sh
gh pr-review review --submit \
  --review-id <PRR_...> \
  --event <APPROVE|REQUEST_CHANGES|COMMENT> \
  --body "Overall review summary" \
  -R owner/repo <pr-number>
```

## Output Format

All commands return structured JSON optimized for programmatic use:

- Consistent field names
- Stable ordering
- Omitted fields instead of null values
- Essential data only (no URLs or metadata noise)
- Pre-joined thread replies

Example output structure:

```json
{
  "reviews": [
    {
      "id": "PRR_...",
      "state": "CHANGES_REQUESTED",
      "author_login": "reviewer",
      "comments": [
        {
          "thread_id": "PRRT_...",
          "path": "src/file.go",
          "author_login": "reviewer",
          "body": "Consider refactoring this",
          "created_at": "2024-01-15T10:30:00Z",
          "is_resolved": false,
          "is_outdated": false,
          "thread_comments": [
            {
              "author_login": "author",
              "body": "Good point, will fix",
              "created_at": "2024-01-15T11:00:00Z"
            }
          ]
        }
      ]
    }
  ]
}
```

## Best Practices

1. **Always use `-R owner/repo`** to specify the repository explicitly
2. **Use `--unresolved` and `--not_outdated`** to focus on actionable comments
3. **Save thread_id values** from `review view` output for replying
4. **Filter by reviewer** when dealing with specific review feedback
5. **Use `--tail 1`** to reduce output size by keeping only latest replies
6. **Parse JSON output** instead of trying to scrape text
7. **Always verify line numbers from patch** before adding inline comments
   - Use `gh api repos/OWNER/REPO/pulls/PR/files` to get patches
   - Count only `+` lines in each diff hunk
   - Remember: position restarts at 1 for each hunk

## Common Workflows

### Get Unresolved Comments for Current PR

```sh
gh pr-review review view --unresolved --not_outdated -R owner/repo --pr $(gh pr view --json number -q .number)
```

### Reply to All Unresolved Comments

1. Get unresolved threads: `gh pr-review threads list --unresolved -R owner/repo <pr>`
2. For each thread_id, reply: `gh pr-review comments reply <pr> -R owner/repo --thread-id <id> --body "..."`
3. Optionally resolve: `gh pr-review threads resolve <pr> -R owner/repo --thread-id <id>`

### Create Review with Inline Comments

**Important**: Line numbers must be diff positions, not absolute file line numbers. See Line Number Calculation section above.

1. Start: `gh pr-review review --start -R owner/repo <pr>`
2. **Get patch to verify line numbers**:
   ```sh
   gh api repos/OWNER/REPO/pulls/PR/files --jq '.[] | select(.filename == "target/file.ts") | {filename, patch}'
   ```
3. **Calculate the correct line number** from the diff hunk
4. Add comments: `gh pr-review review --add-comment -R owner/repo <pr> --review-id <PRR_...> --path <file> --line <num> --body "..."`
5. Edit comments (if needed): `gh pr-review review --edit-comment -R owner/repo <pr> --comment-id <PRRC_...> --body "Updated text"`
6. Delete comments (if needed): `gh pr-review review --delete-comment -R owner/repo <pr> --comment-id <PRRC_...>`
7. Submit: `gh pr-review review --submit -R owner/repo <pr> --review-id <PRR_...> --event REQUEST_CHANGES --body "Summary"`

## Important Notes

- All IDs use GraphQL format (PRR_... for reviews, PRRT_... for threads)
- Commands use pure GraphQL (no REST API fallbacks)
- Empty arrays `[]` are returned when no data matches filters
- The `--include-comment-node-id` flag adds PRRC_... IDs when needed
- Thread replies are sorted by created_at ascending
- Use `--start-line` and `--start-side` for multi-line comments; `--line` becomes the end line

## Documentation Links

- Usage guide: docs/USAGE.md
- JSON schemas: docs/SCHEMAS.md
- Agent workflows: docs/AGENTS.md
- Blog post: https://agyn.io/blog/gh-pr-review-cli-agent-workflows
