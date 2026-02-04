package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/resolver"
	reviewsvc "github.com/agynio/gh-pr-review/internal/review"
)

type reviewAddCommentOptions struct {
	Repo      string
	Pull      int
	Selector  string
	ReviewID  string
	Path      string
	Line      int
	Side      string
	StartLine int
	StartSide string
	Body      string
}

func newReviewAddCommentCommand() *cobra.Command {
	opts := &reviewAddCommentOptions{Side: "RIGHT"}

	cmd := &cobra.Command{
		Use:   "add-comment [<number> | <url>]",
		Short: "Add an inline comment to a pending review",
		Long: `Add an inline comment to a pending review.

LINE NUMBER:

The --line flag takes the absolute line number in the file. For RIGHT side
(default), use the line number in the modified file. For LEFT side, use the
line number in the original file.

The line must fall within a diff hunk range. Check the diff header:
  @@ -oldStart,oldCount +newStart,newCount @@
  Valid range for RIGHT: newStart to (newStart + newCount - 1)

Examples:
  - New file @@ -0,0 +1,173 @@:     use --line 80 for line 80
  - Modified @@ -224,6 +224,112 @@: use --line 280 for line 280 of the new file

Get diff info: gh api repos/OWNER/REPO/pulls/PR/files --jq '.[].patch'`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewAddComment(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.ReviewID, "review-id", "", "Review identifier (GraphQL review node ID)")
	cmd.Flags().StringVar(&opts.Path, "path", "", "File path for inline comment")
	cmd.Flags().IntVar(&opts.Line, "line", 0, "Absolute line number in the file (must fall within a diff hunk range)")
	cmd.Flags().StringVar(&opts.Side, "side", opts.Side, "Diff side for inline comment (LEFT or RIGHT)")
	cmd.Flags().IntVar(&opts.StartLine, "start-line", 0, "Start line for multi-line comments")
	cmd.Flags().StringVar(&opts.StartSide, "start-side", "", "Start side for multi-line comments")
	cmd.Flags().StringVar(&opts.Body, "body", "", "Comment body")

	return cmd
}

func runReviewAddComment(cmd *cobra.Command, opts *reviewAddCommentOptions) error {
	reviewID := strings.TrimSpace(opts.ReviewID)
	if reviewID == "" {
		return errors.New("--review-id is required")
	}
	if !strings.HasPrefix(reviewID, "PRR_") {
		return fmt.Errorf("invalid --review-id %q: must be a GraphQL node id (PRR_...)", opts.ReviewID)
	}

	side, err := normalizeSide(opts.Side)
	if err != nil {
		return err
	}
	var startLine *int
	if opts.StartLine > 0 {
		startLine = &opts.StartLine
	}
	var startSide *string
	if opts.StartSide != "" {
		normalized, err := normalizeSide(opts.StartSide)
		if err != nil {
			return fmt.Errorf("invalid start-side: %w", err)
		}
		startSide = &normalized
	}

	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	identity, err := resolver.Resolve(selector, opts.Repo, os.Getenv("GH_HOST"))
	if err != nil {
		return err
	}

	service := reviewsvc.NewService(apiClientFactory(identity.Host))

	input := reviewsvc.ThreadInput{
		ReviewID:  reviewID,
		Path:      strings.TrimSpace(opts.Path),
		Line:      opts.Line,
		Side:      side,
		StartLine: startLine,
		StartSide: startSide,
		Body:      opts.Body,
	}

	thread, err := service.AddThread(identity, input)
	if err != nil {
		return err
	}
	return encodeJSON(cmd, thread)
}
