package entities

// RepositoryOptions are used when creating a repository within a organization.
type RepositoryOptions struct {
	Name     string
	Private  bool
	TeamID   int
	AutoInit bool
	Issues   bool
	Hook     string
}

// NewRepo creates a RepositoryOptions struct with details to create a repo.
func NewRepo(name string, private bool) RepositoryOptions {
	return RepositoryOptions{
		Name:     name,
		Private:  private,
		AutoInit: true,
		Issues:   true,
		//Hook:     "push", // TODO: uncomment when CI rebuilds all on new test.
	}
}

type ownerType int

const (
	orgOwner ownerType = iota
	usrOwner
)

// Repo represent a git repository. TODO This is currently not used.
type Repo struct {
	Name        string
	Fullname    string
	Description string
	Language    string

	// Owners
	OwnerType ownerType
	Owner     string
	Admins    map[string]interface{}

	// URLs
	HTMLURL  string
	CloneURL string
	Homepage string
}
