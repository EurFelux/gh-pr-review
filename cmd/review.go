package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newReviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Manage pending reviews via GraphQL helpers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return errors.New("specify a subcommand: start, add-comment, edit-comment, delete-comment, submit, preview, or view")
		},
	}

	cmd.AddCommand(newReviewStartCommand())
	cmd.AddCommand(newReviewAddCommentCommand())
	cmd.AddCommand(newReviewEditCommentCommand())
	cmd.AddCommand(newReviewDeleteCommentCommand())
	cmd.AddCommand(newReviewSubmitCommand())
	cmd.AddCommand(newReviewPreviewCommand())
	cmd.AddCommand(newReviewViewCommand())

	return cmd
}

func normalizeSide(side string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(side))
	switch s {
	case "LEFT", "RIGHT":
		return s, nil
	case "":
		return "", errors.New("side is required")
	default:
		return "", fmt.Errorf("invalid side %q: must be LEFT or RIGHT", side)
	}
}

func normalizeEvent(event string) (string, error) {
	e := strings.ToUpper(strings.TrimSpace(event))
	switch e {
	case "APPROVE", "COMMENT", "REQUEST_CHANGES":
		return e, nil
	default:
		return "", fmt.Errorf("invalid event %q: must be APPROVE, COMMENT, or REQUEST_CHANGES", event)
	}
}

func ensureGraphQLReviewID(value string) (string, error) {
	id := strings.TrimSpace(value)
	if id == "" {
		return "", errors.New("review id is required")
	}
	if strings.HasPrefix(id, "PRR_") {
		return id, nil
	}
	isNumeric := true
	for _, r := range id {
		if r < '0' || r > '9' {
			isNumeric = false
			break
		}
	}
	if isNumeric {
		return "", fmt.Errorf("--review-id %q is a REST review id; provide the GraphQL review node id (PRR_...)", id)
	}
	return "", fmt.Errorf("--review-id %q is not a GraphQL review node id (expected prefix PRR_)", id)
}
