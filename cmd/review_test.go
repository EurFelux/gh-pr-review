package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/agynio/gh-pr-review/internal/ghcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type obj = map[string]interface{}

func TestReviewStartCommand_GraphQLOnly(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	call := 0
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		call++
		switch call {
		case 1:
			payload := map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"id":         "PRR_node",
						"headRefOid": "abc123",
					},
				},
			}
			return assignJSON(result, payload)
		case 2:
			payload := map[string]interface{}{
				"addPullRequestReview": map[string]interface{}{
					"pullRequestReview": map[string]interface{}{
						"id":    "PRR_review",
						"state": "PENDING",
					},
				},
			}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected graphql invocation")
		}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--start", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "PRR_review", payload["id"])
	assert.Equal(t, "PENDING", payload["state"])
	_, hasSubmitted := payload["submitted_at"]
	assert.False(t, hasSubmitted)
	_, hasHTML := payload["html_url"]
	assert.False(t, hasHTML)
	_, hasDatabase := payload["database_id"]
	assert.False(t, hasDatabase)
	assert.Equal(t, 2, call)
}

func TestReviewAddCommentCommand_GraphQLOnly(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRR_review", input["pullRequestReviewId"])
		require.Equal(t, "scenario.md", input["path"])
		require.Equal(t, 12, input["line"])
		require.Equal(t, "RIGHT", input["side"])
		require.Equal(t, "note", input["body"])

		payload := map[string]interface{}{
			"addPullRequestReviewThread": map[string]interface{}{
				"thread": map[string]interface{}{
					"id":         "THREAD1",
					"path":       "scenario.md",
					"isOutdated": false,
					"line":       12,
				},
			},
		}
		return assignJSON(result, payload)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--add-comment", "--review-id", "PRR_review", "--path", "scenario.md", "--line", "12", "--body", "note", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "THREAD1", payload["id"])
	assert.Equal(t, "scenario.md", payload["path"])
	assert.Equal(t, false, payload["is_outdated"])
	assert.Equal(t, float64(12), payload["line"])
}

func TestReviewAddCommentCommandRequiresGraphQLReviewID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected graphql invocation")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"review", "--add-comment", "--review-id", "123", "--path", "scenario.md", "--line", "12", "--body", "note", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL node id")
}

func TestReviewSubmitCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		require.Contains(t, query, "submitPullRequestReview")
		payload, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRR_kwM123", payload["pullRequestReviewId"])
		require.Equal(t, "COMMENT", payload["event"])
		require.Equal(t, "Please update", payload["body"])

		return assignJSON(result, obj{
			"data": obj{
				"submitPullRequestReview": obj{
					"pullRequestReview": obj{"id": "PRR_kwM123"},
				},
			},
		})
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--submit", "--review-id", "PRR_kwM123", "--event", "COMMENT", "--body", "Please update", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Review submitted successfully", payload["status"])
}

func TestReviewSubmitCommandRequiresGraphQLReviewID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected GraphQL call")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--submit", "--review-id", "511", "--event", "APPROVE", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REST review id")
}

func TestReviewSubmitCommandRejectsNonPRRPrefix(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected GraphQL call")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--submit", "--review-id", "RANDOM_ID", "--event", "COMMENT", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL review node id")
}

func TestReviewDeleteCommentCommand_GraphQLOnly(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRRC_kwDOAAABbcdEFG12", input["id"])

		return assignJSON(result, obj{
			"data": obj{
				"deletePullRequestReviewComment": obj{
					"pullRequestReview": obj{"id": "PRR_review123"},
				},
			},
		})
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{
		"review", "--delete-comment",
		"--comment-id", "PRRC_kwDOAAABbcdEFG12",
		"--repo", "octo/demo", "7",
	})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Comment deleted successfully", payload["status"])
}

func TestReviewDeleteCommentCommandRequiresGraphQLCommentID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected graphql invocation")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"review", "--delete-comment",
		"--comment-id", "12345",
		"--repo", "octo/demo", "7",
	})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL node id")
}

func TestReviewDeleteCommentCommandRequiresCommentID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected graphql invocation")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"review", "--delete-comment",
		"--repo", "octo/demo", "7",
	})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--comment-id is required")
}

func TestReviewSubmitCommandAllowsNullReview(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		response := obj{
			"data": obj{
				"submitPullRequestReview": obj{
					"pullRequestReview": nil,
				},
			},
		}
		return assignJSON(result, response)
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"review", "--submit", "--review-id", "PRR_kwM123", "--event", "COMMENT", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.NoError(t, err)
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Review submitted successfully", payload["status"])
}

func TestReviewEditCommentCommand_GraphQLOnly(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		input, ok := variables["input"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "PRRC_kwDOAAABbcdEFG12", input["pullRequestReviewCommentId"])
		require.Equal(t, "Updated comment text", input["body"])

		return assignJSON(result, obj{
			"data": obj{
				"updatePullRequestReviewComment": obj{
					"pullRequestReviewComment": obj{"id": "PRRC_kwDOAAABbcdEFG12"},
				},
			},
		})
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{
		"review", "--edit-comment",
		"--comment-id", "PRRC_kwDOAAABbcdEFG12",
		"--body", "Updated comment text",
		"--repo", "octo/demo", "7",
	})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Comment updated successfully", payload["status"])
}

func TestReviewEditCommentCommandRequiresGraphQLCommentID(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected graphql invocation")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"review", "--edit-comment",
		"--comment-id", "12345",
		"--body", "Updated text",
		"--repo", "octo/demo", "7",
	})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GraphQL node id")
}

func TestReviewEditCommentCommandRequiresBody(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return errors.New("unexpected graphql invocation")
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{
		"review", "--edit-comment",
		"--comment-id", "PRRC_kwDOAAABbcdEFG12",
		"--repo", "octo/demo", "7",
	})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--body is required")
}

func TestReviewSubmitCommandHandlesGraphQLErrors(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		return &ghcli.GraphQLError{Errors: []ghcli.GraphQLErrorEntry{{Message: "mutation failed", Path: []interface{}{"mutation", "submitPullRequestReview"}}}}
	}
	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"review", "--submit", "--review-id", "PRR_kwM123", "--event", "COMMENT", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "review submission failed")
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "Review submission failed", payload["status"])
	errorsField, ok := payload["errors"].([]interface{})
	require.True(t, ok)
	require.Len(t, errorsField, 1)
	first, ok := errorsField[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "mutation failed", first["message"])
}

func TestReviewPreviewCommand(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	callCount := 0
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		callCount++
		switch callCount {
		case 1:
			// viewer query - ghcli.GraphQL extracts data field automatically
			payload := obj{
				"viewer": obj{"login": "testuser"},
			}
			return assignJSON(result, payload)
		case 2:
			// pending review query - ghcli.GraphQL extracts data field automatically
			payload := obj{
				"repository": obj{
					"pullRequest": obj{
						"reviews": obj{
							"nodes": []obj{
								{
									"id":         "PRR_preview123",
									"databaseId": 12345,
									"state":      "PENDING",
									"author":     obj{"login": "testuser"},
									"comments": obj{
										"nodes": []obj{
											{
												"id":         "PRRC_comment1",
												"databaseId": 67890,
												"path":       "src/main.go",
												"line":       42,
												"body":       "This needs refactoring",
											},
										},
									},
								},
							},
						},
					},
				},
			}
			return assignJSON(result, payload)
		default:
			return errors.New("unexpected graphql call")
		}
	}

	fake.restFunc = func(method, path string, params map[string]string, body interface{}, result interface{}) error {
		// Return file patches for code context
		files := []obj{
			{
				"filename": "src/main.go",
				"patch": "@@ -40,5 +40,5 @@ func example() {\n oldFunc()\n-new line\n+refactored line\n }",
			},
		}
		return assignJSON(result, files)
	}

	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"review", "--preview", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.NoError(t, err)
	assert.Empty(t, stderr.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &payload))
	assert.Equal(t, "PRR_preview123", payload["review_id"])
	assert.Equal(t, float64(12345), payload["database_id"])
	assert.Equal(t, "PENDING", payload["state"])
	assert.Equal(t, float64(1), payload["comments_count"])

	comments, ok := payload["comments"].([]interface{})
	require.True(t, ok)
	require.Len(t, comments, 1)

	firstComment := comments[0].(map[string]interface{})
	assert.Equal(t, "PRRC_comment1", firstComment["id"])
	assert.Equal(t, "src/main.go", firstComment["path"])
	assert.Equal(t, float64(42), firstComment["line"])
	assert.Equal(t, "This needs refactoring", firstComment["body"])
}

func TestReviewPreviewCommandNoPendingReview(t *testing.T) {
	originalFactory := apiClientFactory
	defer func() { apiClientFactory = originalFactory }()

	fake := &commandFakeAPI{}
	fake.graphqlFunc = func(query string, variables map[string]interface{}, result interface{}) error {
		// viewer query - ghcli.GraphQL extracts data field automatically
		if strings.Contains(query, "viewer") {
			payload := obj{
				"viewer": obj{"login": "testuser"},
			}
			return assignJSON(result, payload)
		}

		// pending review query - empty result (ghcli.GraphQL extracts data field)
		payload := obj{
			"repository": obj{
				"pullRequest": obj{
					"reviews": obj{
						"nodes": []obj{},
					},
				},
			},
		}
		return assignJSON(result, payload)
	}

	apiClientFactory = func(host string) ghcli.API { return fake }

	root := newRootCommand()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"review", "--preview", "--repo", "octo/demo", "7"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending review")
}
