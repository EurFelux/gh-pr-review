package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/resolver"
	reviewsvc "github.com/agynio/gh-pr-review/internal/review"
)

type reviewSubmitOptions struct {
	Repo     string
	Pull     int
	Selector string
	ReviewID string
	Event    string
	Body     string
}

func newReviewSubmitCommand() *cobra.Command {
	opts := &reviewSubmitOptions{Event: "COMMENT"}

	cmd := &cobra.Command{
		Use:   "submit [<number> | <url>]",
		Short: "Submit a pending review",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewSubmit(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.ReviewID, "review-id", "", "Review identifier (GraphQL review node ID)")
	cmd.Flags().StringVar(&opts.Event, "event", opts.Event, "Review submission event (APPROVE, COMMENT, REQUEST_CHANGES)")
	cmd.Flags().StringVar(&opts.Body, "body", "", "Review body")

	return cmd
}

func runReviewSubmit(cmd *cobra.Command, opts *reviewSubmitOptions) error {
	event, err := normalizeEvent(opts.Event)
	if err != nil {
		return err
	}
	reviewID, err := ensureGraphQLReviewID(opts.ReviewID)
	if err != nil {
		return err
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

	input := reviewsvc.SubmitInput{
		ReviewID: reviewID,
		Event:    event,
		Body:     opts.Body,
	}
	status, err := service.Submit(identity, input)
	if err != nil {
		return err
	}
	if status.Success {
		return encodeJSON(cmd, map[string]string{"status": "Review submitted successfully"})
	}
	failure := map[string]interface{}{
		"status": "Review submission failed",
	}
	if len(status.Errors) > 0 {
		failure["errors"] = status.Errors
	}
	if err := encodeJSON(cmd, failure); err != nil {
		return err
	}
	return errors.New("review submission failed")
}
