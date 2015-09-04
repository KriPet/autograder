package web

import (
	"net/http"
	"strings"

	"github.com/hfurubotten/autograder/global"
)

// HelpURL is the URL used to call HelpHandler.
var HelpURL = "/help/"

// HelpHandler is a http handler used to serve the help pages.
func HelpHandler(w http.ResponseWriter, r *http.Request) {
	addr := strings.TrimPrefix(r.URL.String(), "/")
	addr = strings.TrimSuffix(addr, "/")
	if addr == "help" {
		addr = "help/index"
	}

	// Checks if the user is signed in.
	member, err := checkMemberApproval(w, r, false)
	if err == nil {
		view := StdTemplate{
			Member:            member,
			AppName:           global.AppName,
			VersionSystemName: global.VersionSystemName,
		}

		execTemplate(addr+".html", w, view)
	} else {
		execTemplate(addr+".html", w, nil)
	}

}
