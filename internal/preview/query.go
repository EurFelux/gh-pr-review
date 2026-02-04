package preview

// pendingReviewQuery retrieves pending reviews for the current viewer with comments.
const pendingReviewQuery = `query PendingReviewWithComments(
  $owner: String!,
  $name: String!,
  $number: Int!,
  $pageSize: Int!,
  $cursor: String
) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviews(states: [PENDING], first: $pageSize, after: $cursor) {
        nodes {
          id
          databaseId
          state
          author {
            login
          }
          comments(first: 100) {
            nodes {
              id
              databaseId
              path
              line
              startLine
              originalLine
              originalStartLine
              side
              startSide
              body
              diffHunk
              commit {
                oid
              }
              originalCommit {
                oid
              }
            }
            pageInfo {
              hasNextPage
              endCursor
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}`
