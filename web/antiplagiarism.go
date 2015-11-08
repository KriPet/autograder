package web

import (
	"fmt"
	"net/http"
	"strings"

	git "github.com/hfurubotten/autograder/entities"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/autograde/antiplagiarism/proto"
)

// ManualTestPlagiarismURL is the URL used to call ManualTestPlagiarismHandler.
var ManualTestPlagiarismURL = "/event/manualtestplagiarism"

// ManualTestPlagiarismHandler is a http handler for manually triggering test builds.
func ManualTestPlagiarismHandler(w http.ResponseWriter, r *http.Request) {
	if !git.HasOrganization(r.FormValue("course")) {
		http.Error(w, "Unknown organization", 404)
		return
	}

	org, err := git.NewOrganization(r.FormValue("course"), true)
	if err != nil {
		http.Error(w, "Organization Error", 404)
		return
	}

	var labs []string
	var languages []int32
	var repos []string

	if strings.Contains(r.FormValue("labs"), "group") {
		// Get the information for groups

		// Order of labs and languages matters. They must match.
		length := len(org.GroupLabFolders)
		for i := 1; i <= length; i++ {
			if org.GroupLabFolders[i] != "" {
				labs = append(labs, org.GroupLabFolders[i])
				languages = append(languages, org.GroupLanguages[i])
			}
		}

		// The order of the repos does not matter.
		for groupName, _ := range org.Groups {
			repos = append(repos, groupName)
		}
	} else {
		// Get the information for individuals

		// Order of labs and languages matters. They must match.
		length := len(org.IndividualLabFolders)
		for i := 1; i <= length; i++ {
			if org.IndividualLabFolders[i] != "" {
				labs = append(labs, org.IndividualLabFolders[i])
				languages = append(languages, org.IndividualLanguages[i])
			}
		}

		// The order of the repos does not matter.
		for indvName, _ := range org.Members {
			repos = append(repos, indvName + "-labs")
		}
	}

	// Create request
	request := pb.ApRequest{GithubOrg: r.FormValue("course"),
		GithubToken:  org.AdminToken,
		StudentRepos: repos,
		LabNames:     labs,
		LabLanguages: languages}

	go callAntiplagiarism(request)

	
}

// callAntiplagiarism sends a request to the anti-plagiarism software.
// It takes an ApRequest (anti-plagiarism request) as input.
func callAntiplagiarism(request pb.ApRequest) {
	endpoint := "localhost:11111"
	var opts []grpc.DialOption
	// Currently just on localhost.
	// TODO: Add transport security.
	opts = append(opts, grpc.WithInsecure())

	// Create connection
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		fmt.Printf("Error while connecting to server: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("Connected to server on %v\n", endpoint)

	// Create client
	client := pb.NewApClient(conn)

	// Send request and get response
	response, err := client.CheckPlagiarism(context.Background(), &request)

	// Check response
	if err != nil {
		fmt.Printf("gRPC error: %s\n", err)
	} else if response.Success == false {
		fmt.Printf("Anti-plagiarism error: %s\n", response.Err)
	} else {
		fmt.Printf("Anti-plagiarism application ran successfully.\n")
	}
}
