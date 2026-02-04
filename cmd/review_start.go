package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/agynio/gh-pr-review/internal/resolver"
	reviewsvc "github.com/agynio/gh-pr-review/internal/review"
)

type reviewStartOptions struct {
	Repo     string
	Pull     int
	Selector string
	Commit   string
}

func newReviewStartCommand() *cobra.Command {
	opts := &reviewStartOptions{}

	cmd := &cobra.Command{
		Use:   "start [<number> | <url>]",
		Short: "Open a pending review",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Selector = args[0]
			}
			return runReviewStart(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in 'owner/repo' format")
	cmd.Flags().IntVar(&opts.Pull, "pr", 0, "Pull request number")
	cmd.Flags().StringVar(&opts.Commit, "commit", "", "Commit SHA for review start (defaults to current head)")

	return cmd
}

func runReviewStart(cmd *cobra.Command, opts *reviewStartOptions) error {
	selector, err := resolver.NormalizeSelector(opts.Selector, opts.Pull)
	if err != nil {
		return err
	}

	identity, err := resolver.Resolve(selector, opts.Repo, os.Getenv("GH_HOST"))
	if err != nil {
		return err
	}

	service := reviewsvc.NewService(apiClientFactory(identity.Host))
	state, err := service.Start(identity, strings.TrimSpace(opts.Commit))
	if err != nil {
		return err
	}
	return encodeJSON(cmd, state)
}
