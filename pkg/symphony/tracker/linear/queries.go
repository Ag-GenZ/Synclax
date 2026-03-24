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

	issuesByStateIDsQuery = `
query IssuesByStateIDs($stateIDs: [ID!]!, $first: Int!, $after: String) {
  issues(
    filter: { state: { id: { in: $stateIDs } } }
    first: $first
    after: $after
  ) {
    nodes {
      id
      identifier
      state {
        id
        name
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`

	projectTeamStatesQuery = `
query ProjectTeamStates($projectSlug: String!) {
  projects(filter: { slugId: { eq: $projectSlug } }, first: 1) {
    nodes {
      teams {
        nodes {
          id
          states {
            nodes {
              id
              name
              type
              position
            }
          }
        }
      }
    }
  }
}`

	workflowStateCreateMutation = `
mutation WorkflowStateCreate($input: WorkflowStateCreateInput!) {
  workflowStateCreate(input: $input) {
    success
    workflowState {
      id
      name
      type
      position
    }
  }
}`

	workflowStateArchiveMutation = `
mutation WorkflowStateArchive($id: String!) {
  workflowStateArchive(id: $id) {
    success
    entity {
      id
      name
    }
  }
}`

	issueUpdateStateMutation = `
mutation IssueUpdateState($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
    issue {
      id
      identifier
      state {
        id
        name
      }
    }
  }
}`
)
