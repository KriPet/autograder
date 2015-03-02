package git

import (
	"crypto/md5"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
	"github.com/hfurubotten/autograder/global"
	"github.com/hfurubotten/diskv"
	"github.com/hfurubotten/github-gamification/entities"
)

func init() {
	gob.Register(Organization{})
	gob.Register(CodeReview{})
}

type CodeReview struct {
	Title string
	Ext   string
	Desc  string
	Code  string
	User  string

	// Data from Github
	URL string
}

type Organization struct {
	entities.Organization

	GroupAssignments      int
	IndividualAssignments int

	// Lab assignment info. TODO: collect this into one struct!
	IndividualLabFolders map[int]string
	GroupLabFolders      map[int]string
	IndividualDeadlines  map[int]time.Time
	GroupDeadlines       map[int]time.Time

	StudentTeamID int
	OwnerTeamID   int
	Private       bool

	GroupCount         int
	PendingGroup       map[int]interface{}
	PendingRandomGroup map[string]interface{}
	Groups             map[string]interface{}
	PendingUser        map[string]interface{}
	Members            map[string]interface{}
	Teachers           map[string]interface{}

	CodeReview     bool
	CodeReviewlist []CodeReview

	AdminToken  string
	githubadmin *github.Client

	CI CIOptions
}

// NewOrganization tries to fetch a organization from storage on disk or memory.
// If non exists with given name, it creates a new organization.
func NewOrganization(name string) (org *Organization, err error) {
	org = new(Organization)
	if GetOrganizationStore().Has(name) {
		err = GetOrganizationStore().ReadGob(name, org, false)
		if err != nil {
			return nil, err
		}

		// migrating to use of CI.Secret
		if org.CI.Secret == "" {
			org.CI.Secret = fmt.Sprintf("%x", md5.Sum([]byte(name+time.Now().String())))
			org.Save()
		}

		return org, nil
	}

	o, err := entities.NewOrganization(name)
	if err != nil {
		return nil, err
	}

	return &Organization{
		Organization:         *o,
		IndividualLabFolders: make(map[int]string),
		GroupLabFolders:      make(map[int]string),
		PendingGroup:         make(map[int]interface{}),
		PendingRandomGroup:   make(map[string]interface{}),
		Groups:               make(map[string]interface{}),
		PendingUser:          make(map[string]interface{}),
		Members:              make(map[string]interface{}),
		Teachers:             make(map[string]interface{}),
		IndividualDeadlines:  make(map[int]time.Time),
		GroupDeadlines:       make(map[int]time.Time),
		CodeReviewlist:       make([]CodeReview, 0),
		CI: CIOptions{
			Basepath: "/testground/src/github.com/" + name + "/",
			Secret:   fmt.Sprintf("%x", md5.Sum([]byte(name+time.Now().String()))),
		},
	}, nil
}

func NewOrganizationWithGithubData(gorg *github.Organization) (org *Organization, err error) {
	if gorg == nil {
		return nil, errors.New("Cannot use nil github.Organization object")
	}

	org, err = NewOrganization(*gorg.Login)
	if err != nil {
		return nil, err
	}

	org.ImportGithubData(gorg)
	return
}

// connectAdminToGithub will create a github client. This client will be used to talk with githubs api.
func (o *Organization) connectAdminToGithub() error {
	if o.githubadmin != nil {
		return nil
	}

	if o.AdminToken == "" {
		return errors.New("Missing AccessToken to the memeber. Can't contact github.")
	}

	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: o.AdminToken},
	}
	o.githubadmin = github.NewClient(t.Client())
	return nil
}

// LoadStoredData fetches the organization data stored on disk or in cached memory.
func (o *Organization) LoadStoredData() (err error) {
	if GetOrganizationStore().Has(o.Name) {

		err = GetOrganizationStore().ReadGob(o.Name, o, false)
		if err != nil {
			return
		}
	}

	return
}

// StickToSystem will store the organization to cached memory and disk.
func (o *Organization) Save() (err error) {
	if o.IndividualLabFolders == nil {
		o.IndividualLabFolders = make(map[int]string)
	}

	var newfoldernames map[int]string
	if len(o.IndividualLabFolders) != o.IndividualAssignments {
		newfoldernames = make(map[int]string)
		for i := 1; i <= o.IndividualAssignments; i++ {
			if v, ok := o.IndividualLabFolders[i]; ok {
				newfoldernames[i] = v
			} else {
				newfoldernames[i] = "lab" + strconv.Itoa(i)
			}
		}
		o.IndividualLabFolders = newfoldernames
	}

	if o.GroupLabFolders == nil {
		o.GroupLabFolders = make(map[int]string)
	}

	if len(o.GroupLabFolders) != o.GroupAssignments {
		newfoldernames = make(map[int]string)
		for i := 1; i <= o.GroupAssignments; i++ {
			if v, ok := o.GroupLabFolders[i]; ok {
				newfoldernames[i] = v
			} else {
				newfoldernames[i] = "grouplab" + strconv.Itoa(i)
			}
		}
		o.GroupLabFolders = newfoldernames
	}

	if o.CodeReviewlist == nil {
		o.CodeReviewlist = make([]CodeReview, 0)
	}

	return GetOrganizationStore().WriteGob(o.Name, o)
}

// AddCodeReview will add a new code review. This method will
// upload the codereview to github and append it to the list over
// code reviews in this organization.
//
// Filename format committed to github: 'CR-ID'-'Title'-'Username'.'file_ext'
// Commit message: 'CR-ID' 'Username': 'Title'
//
// This method needs locking
func (o *Organization) AddCodeReview(cr *CodeReview) (err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	_, labname, _ := o.FindCurrentLab()

	var path string
	if labname != "" {
		path = fmt.Sprintf("%s/%d-%s-%s.%s", labname, len(o.CodeReviewlist)+1, strings.Replace(cr.Title, " ", "", -1), cr.User, cr.Ext)
	} else {
		path = fmt.Sprintf("%d-%s-%s.%s", len(o.CodeReviewlist)+1, strings.Replace(cr.Title, " ", "", -1), cr.User, cr.Ext)
	}
	commitmsg := fmt.Sprintf("%d %s: %s", len(o.CodeReviewlist)+1, cr.User, cr.Title)

	// Creates the review file
	SHA, err := o.CreateFile(CODEREVIEW_REPO_NAME, path, cr.Code+"\n", commitmsg)
	if err != nil {
		return
	}

	commentmsg := fmt.Sprintf("Code Review %d: %s\n\n%s\n\nHey, could someone look through this and give me some feedback conserning this? \nSincerely @%s\n\n---------\n@%s, follow this tread for feedback.\n",
		len(o.CodeReviewlist)+1, cr.Title, cr.Desc, cr.User, cr.User)

	// Makes a comment on the commit.
	comment := new(github.RepositoryComment)
	comment.Body = github.String(commentmsg)

	_, _, err = o.githubadmin.Repositories.CreateComment(o.Name, CODEREVIEW_REPO_NAME, SHA, comment)
	if err != nil {
		return
	}

	cr.URL = fmt.Sprintf("https://github.com/%s/%s/commit/%s", o.Name, CODEREVIEW_REPO_NAME, SHA)

	o.CodeReviewlist = append(o.CodeReviewlist, *cr)
	return nil
}

// FindCurrentLab will find out which lab has the nearest deadline.
// If no lab has been found the labnum return value will be zero.
func (o *Organization) FindCurrentLab() (labnum int, labname string, labtype int) {
	var lowesttimediff int64 = math.MaxInt64
	for i, t := range o.IndividualDeadlines {
		if time.Now().After(t) {
			continue
		}

		diff := t.Unix() - time.Now().Unix()
		if diff < lowesttimediff {
			labnum = i
			labname = o.IndividualLabFolders[i]
			labtype = INDIVIDUAL
			lowesttimediff = diff
		}
	}

	for i, t := range o.GroupDeadlines {
		if time.Now().After(t) {
			continue
		}

		diff := t.Unix() - time.Now().Unix()
		if diff < lowesttimediff {
			labnum = i
			labname = o.GroupLabFolders[i]
			labtype = GROUP
			lowesttimediff = diff
		}
	}
	return
}

// AddMembership will add a user as a pending student to this
// organization. A pending student is a student which still has
// to be approved by the teaching staff. This method will also
// add the user to the student team on github.
//
// This method needs locking
func (o *Organization) AddMembership(member *Member) (err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	_, _, err = o.githubadmin.Organizations.AddTeamMembership(o.StudentTeamID, member.Username)
	if err != nil {
		return
	}

	//member.AddOrganization(*o)
	//err = member.StickToSystem()
	if o.PendingUser == nil {
		o.PendingUser = make(map[string]interface{})
	}

	if _, ok := o.PendingUser[member.Username]; !ok {
		o.PendingUser[member.Username] = nil
	}

	return
}

// AddTeacher will add a teacher to the teaching staff. This
// method also adds the user to the owners team on github.
//
// TODO: Owners team is no longer a spesial admin team over the
// organization on github. This method needs to be rewritten
// to suppert the new admin API.
//
// This method needs locking
func (o *Organization) AddTeacher(member *Member) (err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	o.Teachers[member.Username] = nil

	var teams map[string]Team
	if o.OwnerTeamID == 0 {
		teams, err = o.ListTeams()
		owners, ok := teams["Owners"]
		if !ok {
			return errors.New("Couldn't find the owners team.")
		}
		o.OwnerTeamID = owners.ID
	}

	_, _, err = o.githubadmin.Organizations.AddTeamMembership(o.OwnerTeamID, member.Username)
	return
}

// IsTeacher returns whether if a user is a teacher or not.
func (o *Organization) IsTeacher(member *Member) bool {
	_, orgok := o.Teachers[member.Username]
	_, mok := member.Teaching[o.Name]

	if orgok && !mok {
		member.Teaching[o.Name] = nil
		member.Save() // This line is not tread safe!
	} else if !orgok && mok {
		o.Teachers[member.Username] = nil
	}

	return orgok || mok
}

// IsMember return whether if the user is a member or not.
func (o *Organization) IsMember(member *Member) bool {
	_, orgok := o.Members[member.Username]
	_, mok := member.Courses[o.Name]

	if orgok && !mok {
		member.Courses[o.Name] = NewCourseOptions(o.Name)
	} else if !orgok && mok {
		o.Members[member.Username] = nil
	}

	return orgok || mok
}

// AddGroup will add a group to the list of groups in the
// organization and also in the pending group list. The
// pending group list will have to be approved by the teaching
// staff.
//
// This method needs locking
func (o *Organization) AddGroup(g *Group) {
	if o.Groups == nil {
		o.Groups = make(map[string]interface{})
	}

	if _, ok := o.PendingGroup[g.ID]; ok {
		delete(o.PendingGroup, g.ID)
	}
	o.Groups["group"+strconv.Itoa(g.ID)] = nil
}

// GetMembership will return the status of a membership to the
// student team on github. The states possible is active or
// pending. Returns error if user is never invited.
func (o *Organization) GetMembership(member *Member) (status string, err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	memship, _, err := o.githubadmin.Organizations.GetTeamMembership(o.StudentTeamID, member.Username)
	if err != nil {
		return
	}

	if memship.State == nil {
		err = errors.New("Couldn't find any role on the username " + member.Username)
		return
	}

	status = *memship.State

	return

}

// Fork will fork a different repository into the organization on
// github. The fork call is async. Only communication errors will
// be reported back, no errors in the forking process.
func (o *Organization) Fork(owner, repo string) (err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	forkopt := github.RepositoryCreateForkOptions{Organization: o.Name}
	_, _, err = o.githubadmin.Repositories.CreateFork(owner, repo, &forkopt)
	return
}

// CreateRepo will create a new repository in the organization on github.
//
// TODO: When the hook option is activated, it can only create a push hook.
// Extend this to include a optional event hook.
func (o *Organization) CreateRepo(opt RepositoryOptions) (err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	if opt.Name == "" {
		return errors.New("Missing required name field. ")
	}

	repo := &github.Repository{}
	repo.Name = github.String(opt.Name)
	repo.Private = github.Bool(opt.Private)
	repo.AutoInit = github.Bool(opt.AutoInit)
	if opt.TeamID != 0 {
		repo.TeamID = github.Int(opt.TeamID)
	}

	_, _, err = o.githubadmin.Repositories.Create(o.Name, repo)
	if err != nil {
		return
	}

	if opt.Hook {
		config := make(map[string]interface{})
		config["url"] = global.Hostname + "/event/hook"
		config["content_type"] = "json"

		hook := github.Hook{
			Name:   github.String("web"),
			Config: config,
		}

		_, _, err = o.githubadmin.Repositories.CreateHook(o.Name, opt.Name, &hook)
	}
	return
}

// CreateTeam will create a new team in the organization on github.
func (o *Organization) CreateTeam(opt TeamOptions) (teamID int, err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	team := &github.Team{}
	team.Name = github.String(opt.Name)
	if opt.Permission != "" {
		team.Permission = github.String(opt.Permission)
	}
	team, _, err = o.githubadmin.Organizations.CreateTeam(o.Name, team)
	if err != nil {
		return
	}

	if opt.RepoNames != nil {
		for _, repo := range opt.RepoNames {
			_, err = o.githubadmin.Organizations.AddTeamRepo(*team.ID, o.Name, repo)
			if err != nil {
				log.Println(err)
			}
		}
	}

	return *team.ID, nil
}

// LinkRepoToTeam will link a repo to a team on github.
func (o *Organization) LinkRepoToTeam(teamID int, repo string) (err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	_, err = o.githubadmin.Organizations.AddTeamRepo(teamID, o.Name, repo)
	return
}

// AddMemberToTeam will add a user to a team on github.
func (o *Organization) AddMemberToTeam(teamID int, user string) (err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	_, _, err = o.githubadmin.Organizations.AddTeamMembership(teamID, user)
	return
}

// ListTeams will list all the teams within the organization on github.
func (o *Organization) ListTeams() (teams map[string]Team, err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	teams = make(map[string]Team)

	gitteams, _, err := o.githubadmin.Organizations.ListTeams(o.Name, nil)
	if err != nil {
		return
	}

	var team Team
	for _, t := range gitteams {
		team = Team{}
		if t.ID != nil {
			team.ID = *t.ID
		}
		if t.Name != nil {
			team.Name = *t.Name
		}
		if t.Permission != nil {
			team.Permission = *t.Permission
		}
		if t.MembersCount != nil {
			team.MemberCount = *t.MembersCount
		}
		if t.ReposCount != nil {
			team.Repocount = *t.ReposCount
		}

		teams[team.Name] = team
	}

	return
}

// ListRepos lists all the repositories in the organization on github.
func (o *Organization) ListRepos() (repos map[string]Repo, err error) {
	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	repolist, _, err := o.githubadmin.Repositories.ListByOrg(o.Name, nil)

	repos = make(map[string]Repo)

	var repo Repo
	for _, r := range repolist {
		repo = Repo{}
		if r.Name != nil {
			repo.Name = *r.Name
		}
		if r.HTMLURL != nil {
			repo.HTMLURL = *r.HTMLURL
		}
		if r.CloneURL != nil {
			repo.CloneURL = *r.CloneURL
		}
		if r.Private != nil {
			repo.Private = *r.Private
		}
		if r.TeamID != nil {
			repo.TeamID = *r.TeamID
		}

		repos[repo.Name] = repo
	}

	return
}

// CreateFile will commit a new file to a repository in the organization on github.
func (o *Organization) CreateFile(repo, path, content, commitmsg string) (commitcode string, err error) {
	if repo == "" || path == "" || content == "" || commitmsg == "" {
		err = errors.New("Missing one of the arguments to create a file.")
		return
	}

	err = o.connectAdminToGithub()
	if err != nil {
		return
	}

	contentopt := github.RepositoryContentFileOptions{
		Message: github.String(commitmsg),
		Content: []byte(content),
	}
	commit, _, err := o.githubadmin.Repositories.CreateFile(o.Name, repo, path, &contentopt)
	if err != nil {
		return
	}

	return *commit.SHA, nil
}

// ListRegisteredOrganizations will list all the organizations registered in autograder.
func ListRegisteredOrganizations() (out []*Organization) {
	out = make([]*Organization, 0)
	keys := GetOrganizationStore().Keys()

	for key := range keys {
		org, err := NewOrganization(key)
		if err != nil {
			log.Println(err)
			continue
		}

		out = append(out, org)
	}

	return
}

// HasOrganization checks if the organization is already registered in autograder.
func HasOrganization(name string) bool {
	return GetOrganizationStore().Has(name)
}

var orgstore *diskv.Diskv

// GetOrganizationStore returns a diskv object used to store the organization object to memory and disk.
func GetOrganizationStore() *diskv.Diskv {
	if orgstore == nil {
		orgstore = diskv.New(diskv.Options{
			BasePath:     global.Basepath + "diskv/orgs/",
			CacheSizeMax: 1024 * 1024 * 256,
		})
	}

	return orgstore
}
