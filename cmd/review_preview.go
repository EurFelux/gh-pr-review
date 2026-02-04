package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/preview"
	"github.com/agynio/gh-pr-review/internal/resolver"
)

type reviewPreviewOptions struct {
	Repo     string
	Pull     int
	Selector string
}

func newReviewPreviewCommand() *cobra.Command {
	opts := &reviewPreviewOptions{}

	cmd := &cobra.Command{
		Use:   "preview [<number> | <url>]",
		Short: "Preview pending review comments with code context",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewPreview(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")

	return cmd
}

func runReviewPreview(cmd *cobra.Command, opts *reviewPreviewOptions) error {
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	identity, err := resolver.Resolve(selector, opts.Repo, os.Getenv("GH_HOST"))
	if err != nil {
		return err
	}

	service := preview.NewService(apiClientFactory(identity.Host))
	result, err := service.Preview(identity)
	if err != nil {
		return err
	}

	return encodeJSON(cmd, result)
}
