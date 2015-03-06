package web

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/hfurubotten/autograder/git"
)

// PublishReviewURL is the URL used to call PublishReviewHandler.
var PublishReviewURL string = "/review/publish"

// PublishReviewView is the JSON view used in the reply in PublishReviewHandler.
type PublishReviewView struct {
	Error     bool
	Errormsg  string
	CommitURL string
}

// PublishReviewHandler is a http handler which will publish a new
// code review to github. The function output json as the answer.
//
// Expected input keys: course, title, fileext, desc and code.
func PublishReviewHandler(w http.ResponseWriter, r *http.Request) {
	view := PublishReviewView{
		Error: true,
	}

	enc := json.NewEncoder(w)

	// Checks if the user is signed in.
	member, err := checkMemberApproval(w, r, true)
	if err != nil {
		view.Errormsg = "Not logged in."
		enc.Encode(view)
		return
	}

	if r.FormValue("course") == "" || r.FormValue("title") == "" ||
		r.FormValue("fileext") == "" || r.FormValue("desc") == "" ||
		r.FormValue("code") == "" {
		view.Errormsg = "Missing some required input data."
		enc.Encode(view)
		return
	}

	if !git.HasOrganization(r.FormValue("course")) {
		view.Errormsg = "Unknown course."
		enc.Encode(view)
		return
	}

	org, err := git.NewOrganization(r.FormValue("course"))
	if err != nil {
		view.Errormsg = "Error while getting orgaization data from storage."
		enc.Encode(view)
		return
	}

	if !org.IsMember(member) {
		view.Errormsg = "Not a member of this course."
		enc.Encode(view)
		return
	}

	alfanumreg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		view.Errormsg = "Internal sanitazion error."
		enc.Encode(view)
		return
	}

	reg, err := regexp.Compile("[^A-Za-z0-9 -.]+")
	if err != nil {
		view.Errormsg = "Internal sanitazion error."
		enc.Encode(view)
		return
	}

	// removes illigal chars
	ext := r.FormValue("fileext")
	ext = alfanumreg.ReplaceAllString(ext, "")
	ext = strings.TrimSpace(ext)
	title := r.FormValue("title")
	title = reg.ReplaceAllString(title, "")
	title = strings.TrimSpace(title)

	org.Lock()
	defer org.Unlock()

	cr := git.CodeReview{
		Title: title,
		Ext:   ext,
		Desc:  r.FormValue("desc"),
		Code:  r.FormValue("code"),
		User:  member.Username,
	}

	err = org.AddCodeReview(&cr)
	if err != nil {
		view.Errormsg = err.Error()
		enc.Encode(view)
		return
	}
	org.Save()

	view.Error = false
	view.CommitURL = cr.URL
	enc.Encode(view)
}

// ListReviewsURL is the URL used to call ListReviewsHandler.
var ListReviewsURL string = "/review/list"

// ListReviewsView is the JSON view used to crate a reply in ListReviewsHandler.
type ListReviewsView struct {
	Error    bool
	Errormsg string
	Reviews  []git.CodeReview
}

// ListReviewsHandler will write back a list of all the code reviews
// in a course, as json data.
//
// Expected input keys: course
func ListReviewsHandler(w http.ResponseWriter, r *http.Request) {
	view := ListReviewsView{
		Error: true,
	}

	enc := json.NewEncoder(w)

	// Checks if the user is signed in.
	member, err := checkMemberApproval(w, r, true)
	if err != nil {
		log.Println(err)
		return
	}

	if r.FormValue("course") == "" {
		view.Errormsg = "Missing required course field."
		enc.Encode(view)
		return
	}

	if !git.HasOrganization(r.FormValue("course")) {
		view.Errormsg = "Unknown course."
		enc.Encode(view)
		return
	}

	org, err := git.NewOrganization(r.FormValue("course"))
	if err != nil {
		view.Errormsg = "Unknown course."
		enc.Encode(view)
		return
	}

	if !org.IsMember(member) {
		view.Errormsg = "Not a member of this course."
		enc.Encode(view)
		return
	}

	view.Error = false
	view.Reviews = org.CodeReviewlist
	enc.Encode(view)
}
