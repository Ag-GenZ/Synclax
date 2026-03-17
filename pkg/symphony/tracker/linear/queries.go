package linear

const (
	candidateIssuesQuery = `
query CandidateIssues($projectSlug: String!, $states: [String!], $first: Int!, $after: String) {
  issues(
    filter: {
      project: { slugId: { eq: $projectSlug } }
      state: { name: { in: $states } }
    }
    first: $first
    after: $after
  ) {
    nodes {
      id
      identifier
      title
      description
      priority
      url
      branchName
      state { name }
      labels { nodes { name } }
      inverseRelations { nodes { type issue { id identifier state { name } } } }
      createdAt
      updatedAt
    }
    pageInfo { hasNextPage endCursor }
  }
}`

	issuesByStatesQuery = `
query IssuesByStates($projectSlug: String!, $states: [String!], $first: Int!, $after: String) {
  issues(
    filter: {
      project: { slugId: { eq: $projectSlug } }
      state: { name: { in: $states } }
    }
    first: $first
    after: $after
  ) {
    nodes {
      id
      identifier
      title
      state { name }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

	issueStatesByIDsQuery = `
query IssueStatesByIDs($ids: [ID!]!) {
  issues(filter: { id: { in: $ids } }, first: 250) {
    nodes { id identifier state { name } }
  }
}`
)
