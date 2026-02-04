package preview

import (
	"errors"
	"fmt"
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
	Side        string   `json:"side"`
	Body        string   `json:"body"`
	CodeContext []string `json:"code_context,omitempty"`
}

// PreviewResult represents the preview of a pending review.
type PreviewResult struct {
	ReviewID      string           `json:"review_id"`
	DatabaseID    int              `json:"database_id"`
	State         string           `json:"state"`
	CommentsCount int              `json:"comments_count"`
	Comments      []CommentPreview `json:"comments"`
}

// Preview fetches the current user's pending review with code context.
func (s *Service) Preview(pr resolver.Identity) (*PreviewResult, error) {
	// Get current viewer
	viewer, err := s.currentViewer()
	if err != nil {
		return nil, err
	}

	// Fetch review threads and find pending review for viewer
	review, threads, err := s.fetchPendingReviewThreads(pr, viewer)
	if err != nil {
		return nil, err
	}

	if review == nil {
		return nil, fmt.Errorf("no pending review found for %s", viewer)
	}

	if len(threads) == 0 {
		return &PreviewResult{
			ReviewID:      review.ID,
			DatabaseID:    review.DatabaseID,
			State:         review.State,
			CommentsCount: 0,
			Comments:      []CommentPreview{},
		}, nil
	}

	// Fetch PR file patches for context resolution
	patches, err := s.fetchFilePatches(pr)
	if err != nil {
		// Non-fatal: we can still return the preview without code context
		patches = make(map[string]string)
	}

	// Build comment previews from threads
	comments := make([]CommentPreview, 0, len(threads))
	for _, thread := range threads {
		// Get the first comment from the thread
		if len(thread.Comments) == 0 {
			continue
		}
		c := thread.Comments[0]

		preview := CommentPreview{
			ID:         c.ID,
			DatabaseID: c.DatabaseID,
			Path:       thread.Path,
			Side:       thread.DiffSide,
			Body:       c.Body,
		}

		// Set line numbers based on side
		if thread.DiffSide == "LEFT" {
			preview.Line = thread.OriginalLine
			if thread.OriginalStartLine > 0 && thread.OriginalStartLine < thread.OriginalLine {
				preview.StartLine = &thread.OriginalStartLine
			}
		} else {
			preview.Line = thread.Line
			if thread.StartLine > 0 && thread.StartLine < thread.Line {
				preview.StartLine = &thread.StartLine
			}
		}

		// Extract code context from patch if available
		if patch, ok := patches[thread.Path]; ok && !thread.IsOutdated {
			context := s.extractCodeContext(patch, thread)
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

// reviewInfo holds basic review information.
type reviewInfo struct {
	ID         string
	DatabaseID int
	State      string
}

// threadInfo represents a review thread.
type threadInfo struct {
	ID                string
	IsResolved        bool
	IsOutdated        bool
	Path              string
	Line              int
	StartLine         int
	OriginalLine      int
	OriginalStartLine int
	DiffSide          string
	Comments          []commentInfo
}

// commentInfo represents a comment in a thread.
type commentInfo struct {
	ID         string
	DatabaseID int
	Body       string
	DiffHunk   string
	Author     string
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

func (s *Service) fetchPendingReviewThreads(pr resolver.Identity, viewer string) (*reviewInfo, []threadInfo, error) {
	variables := map[string]interface{}{
		"owner":    pr.Owner,
		"name":     pr.Repo,
		"number":   pr.Number,
		"pageSize": 100,
	}

	var response struct {
		Repository *struct {
			PullRequest *struct {
				ReviewThreads *struct {
					Nodes []struct {
						ID                string `json:"id"`
						IsResolved        bool   `json:"isResolved"`
						IsOutdated        bool   `json:"isOutdated"`
						Path              string `json:"path"`
						Line              int    `json:"line"`
						StartLine         int    `json:"startLine"`
						OriginalLine      int    `json:"originalLine"`
						OriginalStartLine int    `json:"originalStartLine"`
						DiffSide          string `json:"diffSide"`
						Comments          struct {
							Nodes []struct {
								ID         string `json:"id"`
								DatabaseID int    `json:"databaseId"`
								Body       string `json:"body"`
								DiffHunk   string `json:"diffHunk"`
								Author     *struct {
									Login string `json:"login"`
								} `json:"author"`
								PullRequestReview *struct {
									ID         string `json:"id"`
									DatabaseID int    `json:"databaseId"`
									State      string `json:"state"`
									Author     *struct {
										Login string `json:"login"`
									} `json:"author"`
								} `json:"pullRequestReview"`
							} `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := s.API.GraphQL(reviewThreadsQuery, variables, &response); err != nil {
		return nil, nil, err
	}

	repo := response.Repository
	if repo == nil || repo.PullRequest == nil || repo.PullRequest.ReviewThreads == nil {
		return nil, nil, fmt.Errorf("pull request %s/%s#%d not found", pr.Owner, pr.Repo, pr.Number)
	}

	// Find threads belonging to the current viewer's pending review
	var pendingReview *reviewInfo
	var threads []threadInfo

	for _, node := range repo.PullRequest.ReviewThreads.Nodes {
		// Check each comment in the thread
		for _, c := range node.Comments.Nodes {
			if c.PullRequestReview == nil {
				continue
			}

			review := c.PullRequestReview
			author := ""
			if review.Author != nil {
				author = review.Author.Login
			}

			// Only process threads from the current viewer's pending review
			if !strings.EqualFold(author, viewer) || review.State != "PENDING" {
				continue
			}

			// Record the review info (first occurrence)
			if pendingReview == nil {
				pendingReview = &reviewInfo{
					ID:         review.ID,
					DatabaseID: review.DatabaseID,
					State:      review.State,
				}
			}

			// Build thread info
			thread := threadInfo{
				ID:                node.ID,
				IsResolved:        node.IsResolved,
				IsOutdated:        node.IsOutdated,
				Path:              node.Path,
				Line:              node.Line,
				StartLine:         node.StartLine,
				OriginalLine:      node.OriginalLine,
				OriginalStartLine: node.OriginalStartLine,
				DiffSide:          node.DiffSide,
			}

			// Add comments to thread
			for _, tc := range node.Comments.Nodes {
				author := ""
				if tc.Author != nil {
					author = tc.Author.Login
				}
				thread.Comments = append(thread.Comments, commentInfo{
					ID:         tc.ID,
					DatabaseID: tc.DatabaseID,
					Body:       tc.Body,
					DiffHunk:   tc.DiffHunk,
					Author:     author,
				})
			}

			threads = append(threads, thread)
			break // Only add this thread once
		}
	}

	return pendingReview, threads, nil
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

// extractCodeContext extracts the code lines from a patch for the given thread.
// This is a simplified implementation that uses diffHunk from the comment.
func (s *Service) extractCodeContext(patch string, thread threadInfo) []string {
	if len(thread.Comments) == 0 {
		return nil
	}

	// Use the diffHunk from the first comment
	diffHunk := thread.Comments[0].DiffHunk
	if diffHunk == "" {
		return nil
	}

	// Determine target line range based on side
	targetLine := thread.Line
	startLine := targetLine
	if thread.DiffSide == "LEFT" {
		targetLine = thread.OriginalLine
		startLine = targetLine
		if thread.OriginalStartLine > 0 && thread.OriginalStartLine < targetLine {
			startLine = thread.OriginalStartLine
		}
	} else {
		if thread.StartLine > 0 && thread.StartLine < targetLine {
			startLine = thread.StartLine
		}
	}

	// Parse diffHunk to extract relevant lines
	return parseDiffHunk(diffHunk, startLine, targetLine, thread.DiffSide)
}

// parseDiffHunk parses a diff hunk and extracts lines for the given range.
func parseDiffHunk(diffHunk string, startLine, targetLine int, side string) []string {
	if diffHunk == "" {
		return nil
	}

	lines := strings.Split(diffHunk, "\n")
	var result []string

	var oldLine, newLine int
	inHunk := false

	for _, line := range lines {
		// Parse hunk header: @@ -start,count +start,count @@
		// Also handle: @@ -start +start @@ (no count)
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			// Extract starting line numbers using a more robust approach
			oldStart, newStart := parseHunkHeader(line)
			oldLine = oldStart
			newLine = newStart
			continue
		}

		if !inHunk {
			continue
		}

		// Skip "No newline at end of file" marker
		if strings.HasPrefix(line, "\\") {
			continue
		}

		// Skip empty lines that might appear at the start
		if len(line) == 0 {
			continue
		}

		// Determine which line number to use based on side and line type
		var currentLine int
		var content string
		var shouldInclude bool

		if line[0] == '+' {
			// Added line - only on RIGHT side
			currentLine = newLine
			content = "+" + line[1:]
			shouldInclude = (side == "RIGHT")
			newLine++
		} else if line[0] == '-' {
			// Deleted line - only on LEFT side
			currentLine = oldLine
			content = "-" + line[1:]
			shouldInclude = (side == "LEFT")
			oldLine++
		} else if line[0] == ' ' {
			// Context line - on both sides
			if side == "LEFT" {
				currentLine = oldLine
			} else {
				currentLine = newLine
			}
			content = line[1:]
			shouldInclude = true
			oldLine++
			newLine++
		}

		// Check if this line is in our target range
		if shouldInclude && currentLine >= startLine && currentLine <= targetLine {
			result = append(result, fmt.Sprintf("%d: %s", currentLine, content))
		}

		// Stop if we've passed the target line
		if side == "LEFT" && oldLine > targetLine {
			break
		}
		if side == "RIGHT" && newLine > targetLine {
			break
		}
	}

	return result
}

// parseHunkHeader extracts the starting line numbers from a hunk header.
// Handles formats like "@@ -1,5 +1,6 @@" or "@@ -1 +1 @@"
func parseHunkHeader(header string) (oldStart, newStart int) {
	// Remove the @@ markers
	content := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(header, "@@"), "@@"))

	// Parse the old file info
	oldIdx := strings.Index(content, "-")
	if oldIdx == -1 {
		return 0, 0
	}
	content = content[oldIdx+1:]

	// Find the comma or space for old file
	oldEnd := strings.IndexAny(content, ", ")
	if oldEnd == -1 {
		return 0, 0
	}
	oldStart, _ = strconv.Atoi(content[:oldEnd])

	// Parse the new file info
	newIdx := strings.Index(content, "+")
	if newIdx == -1 {
		return 0, 0
	}
	content = content[newIdx+1:]

	// Find the comma or space for new file
	newEnd := strings.IndexAny(content, ", ")
	if newEnd == -1 {
		newStart, _ = strconv.Atoi(content)
	} else {
		newStart, _ = strconv.Atoi(content[:newEnd])
	}

	return oldStart, newStart
}
