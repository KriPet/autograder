package entities

// RepositoryOptions represent the option when needed to create a repository within a organization.
type RepositoryOptions struct {
	Name     string
	Private  bool
	TeamID   int
	AutoInit bool
	Issues   bool
	Hook     string
}
