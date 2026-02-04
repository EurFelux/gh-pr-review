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

type reviewDeleteCommentOptions struct {
	Repo      string
	Pull      int
	Selector  string
	CommentID string
}

func newReviewDeleteCommentCommand() *cobra.Command {
	opts := &reviewDeleteCommentOptions{}

	cmd := &cobra.Command{
		Use:   "delete-comment [<number> | <url>]",
		Short: "Delete a comment from a pending review",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewDeleteComment(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.CommentID, "comment-id", "", "Comment identifier (GraphQL comment node ID, PRRC_...)")

	return cmd
}

func runReviewDeleteComment(cmd *cobra.Command, opts *reviewDeleteCommentOptions) error {
	commentID := strings.TrimSpace(opts.CommentID)
	if commentID == "" {
		return errors.New("--comment-id is required")
	}
	if !strings.HasPrefix(commentID, "PRRC_") {
		return fmt.Errorf("invalid --comment-id %q: must be a GraphQL node id (PRRC_...)", opts.CommentID)
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

	input := reviewsvc.DeleteCommentInput{
		CommentID: commentID,
	}
	if err := service.DeleteComment(identity, input); err != nil {
		return err
	}
	return encodeJSON(cmd, map[string]string{"status": "Comment deleted successfully"})
}
