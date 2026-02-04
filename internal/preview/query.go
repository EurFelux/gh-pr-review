package preview

// reviewThreadsQuery retrieves review threads for a pull request.
const reviewThreadsQuery = `query ReviewThreads(
  $owner: String!,
  $name: String!,
  $number: Int!,
  $pageSize: Int!,
  $cursor: String
) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: $pageSize, after: $cursor) {
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          startLine
          originalLine
          originalStartLine
          diffSide
          comments(first: 100) {
            nodes {
              id
              databaseId
              body
              diffHunk
              author {
                login
              }
              pullRequestReview {
                id
                databaseId
                state
                author {
                  login
                }
              }
              commit {
                oid
              }
              originalCommit {
                oid
              }
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
