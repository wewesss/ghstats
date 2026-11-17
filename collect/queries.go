package collect

import "time"

const reposQuery = `query($cursor: String, $aff: [RepositoryAffiliation]) {
  viewer {
    login
    followers { totalCount }
    following { totalCount }
    repositories(first: 50, after: $cursor, ownerAffiliations: $aff, orderBy: {field: NAME, direction: ASC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id
        nameWithOwner
        owner { login }
        isPrivate
        isFork
        isArchived
        stargazerCount
        forkCount
        languages(first: 100, orderBy: {field: SIZE, direction: DESC}) {
          totalCount
          edges { size node { name } }
        }
        releases(first: 20, orderBy: {field: CREATED_AT, direction: DESC}) {
          pageInfo { hasNextPage endCursor }
          nodes {
            tagName
            publishedAt
            releaseAssets(first: 100) { nodes { name downloadCount } }
          }
        }
      }
    }
  }
  rateLimit { remaining resetAt }
}`

const releasesQuery = `query($id: ID!, $cursor: String) {
  node(id: $id) {
    ... on Repository {
      releases(first: 50, after: $cursor, orderBy: {field: CREATED_AT, direction: DESC}) {
        pageInfo { hasNextPage endCursor }
        nodes {
          tagName
          publishedAt
          releaseAssets(first: 100) { nodes { name downloadCount } }
        }
      }
    }
  }
}`

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type reposResponse struct {
	Viewer struct {
		Login     string `json:"login"`
		Followers struct {
			TotalCount int `json:"totalCount"`
		} `json:"followers"`
		Following struct {
			TotalCount int `json:"totalCount"`
		} `json:"following"`
		Repositories struct {
			PageInfo pageInfo  `json:"pageInfo"`
			Nodes    []apiRepo `json:"nodes"`
		} `json:"repositories"`
	} `json:"viewer"`
	RateLimit *struct {
		Remaining int       `json:"remaining"`
		ResetAt   time.Time `json:"resetAt"`
	} `json:"rateLimit"`
}

type apiRepo struct {
	ID            string `json:"id"`
	NameWithOwner string `json:"nameWithOwner"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
	IsPrivate      bool `json:"isPrivate"`
	IsFork         bool `json:"isFork"`
	IsArchived     bool `json:"isArchived"`
	StargazerCount int  `json:"stargazerCount"`
	ForkCount      int  `json:"forkCount"`
	Languages      struct {
		Edges []struct {
			Size int64 `json:"size"`
			Node struct {
				Name string `json:"name"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"languages"`
	Releases apiReleases `json:"releases"`
}

type apiReleases struct {
	PageInfo pageInfo     `json:"pageInfo"`
	Nodes    []apiRelease `json:"nodes"`
}

type apiRelease struct {
	TagName       string     `json:"tagName"`
	PublishedAt   *time.Time `json:"publishedAt"`
	ReleaseAssets struct {
		Nodes []struct {
			Name          string `json:"name"`
			DownloadCount int64  `json:"downloadCount"`
		} `json:"nodes"`
	} `json:"releaseAssets"`
}

type releasesResponse struct {
	Node *struct {
		Releases apiReleases `json:"releases"`
	} `json:"node"`
}
