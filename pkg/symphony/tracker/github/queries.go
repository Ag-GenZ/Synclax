package github

const (
	resolveProjectQuery = `
query ResolveProject($projectOwner: String!, $projectNumber: Int!, $repoOwner: String!, $repoName: String!, $fieldName: String!) {
  repositoryOwner(login: $projectOwner) {
    login
    ... on Organization {
      projectV2(number: $projectNumber) {
        id
        title
        field(name: $fieldName) {
          __typename
          ... on ProjectV2SingleSelectField {
            id
            name
            options {
              id
              name
            }
          }
        }
      }
    }
    ... on User {
      projectV2(number: $projectNumber) {
        id
        title
        field(name: $fieldName) {
          __typename
          ... on ProjectV2SingleSelectField {
            id
            name
            options {
              id
              name
            }
          }
        }
      }
    }
  }
  repository(owner: $repoOwner, name: $repoName) {
    id
    name
    owner {
      login
    }
  }
}`

	projectItemsQuery = `
query ProjectItems($projectId: ID!, $fieldName: String!, $first: Int!, $after: String) {
  node(id: $projectId) {
    ... on ProjectV2 {
      items(first: $first, after: $after) {
        nodes {
          id
          fieldValueByName(name: $fieldName) {
            __typename
            ... on ProjectV2ItemFieldSingleSelectValue {
              name
              optionId
              field {
                __typename
                ... on ProjectV2SingleSelectField {
                  id
                  name
                }
              }
            }
          }
          content {
            __typename
            ... on DraftIssue {
              id
            }
            ... on PullRequest {
              id
            }
            ... on Issue {
              id
              number
              title
              body
              url
              state
              createdAt
              updatedAt
              repository {
                name
                owner {
                  login
                }
              }
              labels(first: 20) {
                nodes {
                  name
                }
              }
              blockedBy(first: 50) {
                nodes {
                  id
                  number
                  url
                  state
                  repository {
                    name
                    owner {
                      login
                    }
                  }
                }
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

	projectItemStatesByItemIDsQuery = `
query ProjectItemStatesByItemIDs($ids: [ID!]!, $fieldName: String!) {
  nodes(ids: $ids) {
    __typename
    ... on ProjectV2Item {
      id
      fieldValueByName(name: $fieldName) {
        __typename
        ... on ProjectV2ItemFieldSingleSelectValue {
          name
          optionId
          field {
            __typename
            ... on ProjectV2SingleSelectField {
              id
              name
            }
          }
        }
      }
      content {
        __typename
        ... on Issue {
          id
          number
          repository {
            name
            owner {
              login
            }
          }
        }
      }
    }
  }
}`

	projectItemStateScanQuery = `
query ProjectItemStateScan($projectId: ID!, $fieldName: String!, $first: Int!, $after: String) {
  node(id: $projectId) {
    ... on ProjectV2 {
      items(first: $first, after: $after) {
        nodes {
          id
          fieldValueByName(name: $fieldName) {
            __typename
            ... on ProjectV2ItemFieldSingleSelectValue {
              name
              optionId
              field {
                __typename
                ... on ProjectV2SingleSelectField {
                  id
                  name
                }
              }
            }
          }
          content {
            __typename
            ... on Issue {
              id
              number
              repository {
                name
                owner {
                  login
                }
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
)
