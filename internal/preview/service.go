package preview

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

// Service provides preview functionality for pending reviews.
type Service struct {
	API ghcli.API
}

// NewService creates a new preview service.
func NewService(api ghcli.API) *Service {
	return &Service{API: api}
}

// CommentPreview represents a single pending comment with code context.
type CommentPreview struct {
	ID          string   `json:"id"`
	DatabaseID  int      `json:"database_id"`
	Path        string   `json:"path"`
	Line        int      `json:"line"`
	StartLine   *int     `json:"start_line,omitempty"`
	Body        string   `json:"body"`
	CodeContext []string `json:"code_context"`
}

// PreviewResult represents the preview of a pending review.
type PreviewResult struct {
	ReviewID      string             `json:"review_id"`
	DatabaseID    int                `json:"database_id"`
	State         string             `json:"state"`
	CommentsCount int                `json:"comments_count"`
	Comments      []CommentPreview   `json:"comments"`
}

// Preview fetches the current user's pending review with code context.
func (s *Service) Preview(pr resolver.Identity) (*PreviewResult, error) {
	// Get current viewer
	viewer, err := s.currentViewer()
	if err != nil {
		return nil, err
	}

	// Fetch pending review with comments
	review, err := s.fetchPendingReview(pr, viewer)
	if err != nil {
		return nil, err
	}

	if review == nil {
		return nil, fmt.Errorf("no pending review found for %s", viewer)
	}

	// Fetch PR file patches for context resolution
	patches, err := s.fetchFilePatches(pr)
	if err != nil {
		// Non-fatal: we can still return the preview without code context
		patches = make(map[string]string)
	}

	// Resolve code context for each comment
	comments := make([]CommentPreview, 0, len(review.Comments))
	for _, c := range review.Comments {
		preview := CommentPreview{
			ID:         c.ID,
			DatabaseID: c.DatabaseID,
			Path:       c.Path,
			Line:       c.Line,
			Body:       c.Body,
		}
		if c.StartLine > 0 {
			preview.StartLine = &c.StartLine
		}

			// Try to resolve code context from patch
		// Determine side by checking the last line of diffHunk
		// If it starts with '-', it's LEFT side (deleted line)
		// Otherwise, it's RIGHT side (added or unchanged line)
		if patch, ok := patches[c.Path]; ok {
			side := inferSideFromDiffHunk(c.DiffHunk)
			context := extractCodeContext(patch, c.Line, c.StartLine, side, c.OriginalLine, c.OriginalStartLine)
			preview.CodeContext = context
		}

		comments = append(comments, preview)
	}

	result := &PreviewResult{
		ReviewID:      review.ID,
		DatabaseID:    review.DatabaseID,
		State:         review.State,
		CommentsCount: len(comments),
		Comments:      comments,
	}

	return result, nil
}

// pendingReview represents a pending review from GraphQL.
type pendingReview struct {
	ID         string
	DatabaseID int
	State      string
	Author     struct {
		Login string
	}
	Comments []pendingComment
}

// pendingComment represents a comment in a pending review.
type pendingComment struct {
	ID                string
	DatabaseID        int
	Path              string
	Line              int
	StartLine         int
	OriginalLine      int
	OriginalStartLine int
	Body              string
	DiffHunk          string
}

func (s *Service) currentViewer() (string, error) {
	const query = `query { viewer { login } }`

	var response struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}

	if err := s.API.GraphQL(query, nil, &response); err != nil {
		return "", err
	}

	login := strings.TrimSpace(response.Viewer.Login)
	if login == "" {
		return "", errors.New("viewer login unavailable")
	}

	return login, nil
}

func (s *Service) fetchPendingReview(pr resolver.Identity, viewer string) (*pendingReview, error) {
	variables := map[string]interface{}{
		"owner":    pr.Owner,
		"name":     pr.Repo,
		"number":   pr.Number,
		"pageSize": 10,
	}

	var response struct {
		Repository *struct {
			PullRequest *struct {
				Reviews *struct {
					Nodes []struct {
						ID         string `json:"id"`
						DatabaseID int    `json:"databaseId"`
						State      string `json:"state"`
						Author     *struct {
							Login string `json:"login"`
						} `json:"author"`
						Comments struct {
							Nodes []struct {
								ID                string `json:"id"`
								DatabaseID        int    `json:"databaseId"`
								Path              string `json:"path"`
								Line              int    `json:"line"`
								StartLine         int    `json:"startLine"`
								OriginalLine      int    `json:"originalLine"`
								OriginalStartLine int    `json:"originalStartLine"`
								Body              string `json:"body"`
								DiffHunk          string `json:"diffHunk"`
							} `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviews"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(pendingReviewQuery, variables, &response); err != nil {
		return nil, err
	}

	repo := response.Repository
	if repo == nil || repo.PullRequest == nil || repo.PullRequest.Reviews == nil {
		return nil, fmt.Errorf("pull request %s/%s#%d not found", pr.Owner, pr.Repo, pr.Number)
	}

	// Find the pending review for the current viewer
	for _, node := range repo.PullRequest.Reviews.Nodes {
		author := ""
		if node.Author != nil {
			author = node.Author.Login
		}
		if !strings.EqualFold(author, viewer) {
			continue
		}

		review := &pendingReview{
			ID:         node.ID,
			DatabaseID: node.DatabaseID,
			State:      node.State,
		}

		for _, c := range node.Comments.Nodes {
			comment := pendingComment{
				ID:                c.ID,
				DatabaseID:        c.DatabaseID,
				Path:              c.Path,
				Line:              c.Line,
				StartLine:         c.StartLine,
				OriginalLine:      c.OriginalLine,
				OriginalStartLine: c.OriginalStartLine,
				Body:              c.Body,
				DiffHunk:          c.DiffHunk,
			}
			review.Comments = append(review.Comments, comment)
		}

		return review, nil
	}

	return nil, nil
}

// fetchFilePatches retrieves file patches for the PR via REST API.
func (s *Service) fetchFilePatches(pr resolver.Identity) (map[string]string, error) {
	path := fmt.Sprintf("repos/%s/%s/pulls/%d/files", pr.Owner, pr.Repo, pr.Number)

	var files []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	}

	if err := s.API.REST("GET", path, nil, nil, &files); err != nil {
		return nil, err
	}

	patches := make(map[string]string, len(files))
	for _, f := range files {
		patches[f.Filename] = f.Patch
	}

	return patches, nil
}

// hunkHeaderRegex matches the hunk header line: @@ -start,count +start,count @@
var hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// inferSideFromDiffHunk determines if a comment is on LEFT or RIGHT side
// by examining the last line of diffHunk.
// If the last content line starts with '-', it's LEFT side (deleted line).
// Otherwise (starts with '+' or ' '), it's RIGHT side.
func inferSideFromDiffHunk(diffHunk string) string {
	if diffHunk == "" {
		return "RIGHT" // default to RIGHT side
	}
	scanner := bufio.NewScanner(strings.NewReader(diffHunk))
	var lastLine string
	for scanner.Scan() {
		text := scanner.Text()
		// Skip hunk header
		if hunkHeaderRegex.MatchString(text) {
			continue
		}
		// Skip "\ No newline at end of file" (may have leading space)
		trimmed := strings.TrimSpace(text)
		if len(trimmed) > 0 && trimmed[0] == '\\' {
			continue
		}
		// Remember this content line
		lastLine = text
	}
	// Check the last content line
	if len(lastLine) > 0 && lastLine[0] == '-' {
		return "LEFT"
	}
	return "RIGHT"
}

// extractCodeContext extracts the code lines from a patch for the given line range.
// side is "LEFT" or "RIGHT" indicating which side of the diff the comment is on.
// For LEFT side, originalLine is used to find the line in the original file.
// For RIGHT side, line is used to find the line in the new file.
func extractCodeContext(patch string, line, startLine int, side string, originalLine, originalStartLine int) []string {
	if patch == "" {
		return nil
	}

	// Determine which line numbers to use
	var targetLine, start int
	if side == "LEFT" && originalLine > 0 {
		targetLine = originalLine
		start = targetLine
		if originalStartLine > 0 && originalStartLine < targetLine {
			start = originalStartLine
		}
	} else {
		targetLine = line
		start = targetLine
		if startLine > 0 && startLine < targetLine {
			start = startLine
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(patch))
	var context []string
	newLine := 0   // line number in new file (RIGHT side)
	oldLine := 0   // line number in old file (LEFT side)
	inHunk := false

	for scanner.Scan() {
		text := scanner.Text()

		// Check for hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(text); matches != nil {
			inHunk = true
			oldStart, _ := strconv.Atoi(matches[1])
			newStart, _ := strconv.Atoi(matches[2])
			oldLine = oldStart
			newLine = newStart
			continue
		}

		if !inHunk {
			continue
		}

		// Parse diff line
		if len(text) == 0 {
			// Empty line in context - appears in both files
			if side == "LEFT" {
				if oldLine >= start && oldLine <= targetLine {
					context = append(context, fmt.Sprintf("%d: %s", oldLine, text))
				}
			} else {
				if newLine >= start && newLine <= targetLine {
					context = append(context, fmt.Sprintf("%d: %s", newLine, text))
				}
			}
			oldLine++
			newLine++
		} else if text[0] == '+' {
			// Added line - only in new file (RIGHT side)
			if side != "LEFT" && newLine >= start && newLine <= targetLine {
				context = append(context, fmt.Sprintf("%d: +%s", newLine, text[1:]))
			}
			newLine++
		} else if text[0] == '-' {
			// Deleted line - only in old file (LEFT side)
			if side == "LEFT" && oldLine >= start && oldLine <= targetLine {
				context = append(context, fmt.Sprintf("%d: -%s", oldLine, text[1:]))
			}
			oldLine++
		} else if text[0] == ' ' {
			// Context line - appears in both files
			if side == "LEFT" {
				if oldLine >= start && oldLine <= targetLine {
					context = append(context, fmt.Sprintf("%d: %s", oldLine, text[1:]))
				}
			} else {
				if newLine >= start && newLine <= targetLine {
					context = append(context, fmt.Sprintf("%d: %s", newLine, text[1:]))
				}
			}
			oldLine++
			newLine++
		} else if text[0] == '\\' {
			// "\ No newline at end of file" - skip
			continue
		}

		// Check if we've collected all lines we need
		if side == "LEFT" {
			if oldLine > targetLine {
				break
			}
		} else {
			if newLine > targetLine {
				break
			}
		}
	}

	return context
}
