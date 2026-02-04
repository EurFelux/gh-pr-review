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

type reviewEditCommentOptions struct {
	Repo      string
	Pull      int
	Selector  string
	CommentID string
	Body      string
}

func newReviewEditCommentCommand() *cobra.Command {
	opts := &reviewEditCommentOptions{}

	cmd := &cobra.Command{
		Use:   "edit-comment [<number> | <url>]",
		Short: "Edit a comment in a pending review",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewEditComment(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.CommentID, "comment-id", "", "Comment identifier (GraphQL comment node ID, PRRC_...)")
	cmd.Flags().StringVar(&opts.Body, "body", "", "New comment body")

	return cmd
}

func runReviewEditComment(cmd *cobra.Command, opts *reviewEditCommentOptions) error {
	commentID := strings.TrimSpace(opts.CommentID)
	if commentID == "" {
		return errors.New("--comment-id is required")
	}
	if !strings.HasPrefix(commentID, "PRRC_") {
		return fmt.Errorf("invalid --comment-id %q: must be a GraphQL node id (PRRC_...)", opts.CommentID)
	}

	trimmedBody := strings.TrimSpace(opts.Body)
	if trimmedBody == "" {
		return errors.New("--body is required")
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

	input := reviewsvc.UpdateCommentInput{
		CommentID: commentID,
		Body:      trimmedBody,
	}
	if err := service.UpdateComment(identity, input); err != nil {
		return err
	}
	return encodeJSON(cmd, map[string]string{"status": "Comment updated successfully"})
}
