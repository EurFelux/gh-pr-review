package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/resolver"
	reviewsvc "github.com/agynio/gh-pr-review/internal/review"
)

type reviewEditOptions struct {
	Repo     string
	Pull     int
	Selector string
	ReviewID string
	Body     string
}

func newReviewEditCommand() *cobra.Command {
	opts := &reviewEditOptions{}

	cmd := &cobra.Command{
		Use:   "edit [<number> | <url>]",
		Short: "Edit the body of a submitted review",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewEdit(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.ReviewID, "review-id", "", "Review identifier (GraphQL review node ID, PRR_...)")
	cmd.Flags().StringVar(&opts.Body, "body", "", "New review body")

	return cmd
}

func runReviewEdit(cmd *cobra.Command, opts *reviewEditOptions) error {
	reviewID, err := ensureGraphQLReviewID(opts.ReviewID)
	if err != nil {
		return err
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

	input := reviewsvc.UpdateReviewInput{
		ReviewID: reviewID,
		Body:     trimmedBody,
	}
	if err := service.UpdateReview(identity, input); err != nil {
		return err
	}
	return encodeJSON(cmd, map[string]string{"status": "Review updated successfully"})
}
