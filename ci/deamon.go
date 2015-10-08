package ci

import (
	"bufio"
	"bytes"
	// "errors"
	"fmt"
	"log"
	"time"

	"github.com/autograde/kit/score"
	git "github.com/hfurubotten/autograder/entities"
)

// DaemonOptions represent the options needed to start the testing daemon.
type DaemonOptions struct {
	Org   string
	User  string
	Group int

	Repo       string
	BaseFolder string
	LabFolder  string
	LabNumber  int
	AdminToken string
	DestFolder string
	Secret     string
	IsPush     bool
}

// StartTesterDaemon will start a new test build in the background.
//TODO this functions is too long. needs to be split into shorter functions.
func StartTesterDaemon(opt DaemonOptions) {
	// safeguard
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic: ", r)
		}
	}()

	startMsg := fmt.Sprintf("Running tests for: %s/%s", opt.Org, opt.Repo)
	log.Println(startMsg)

	// Test execution
	env, err := NewVirtual()
	if err != nil {
		panic(err)
	}

	err = env.NewContainer("autograder")
	if err != nil {
		panic(err)
	}

	// cleanup
	defer env.RemoveContainer()

	// mkdir /testground/github.com/
	// git clone user-labs
	// git clone test-labs
	// cp test-labs user-labs
	// /bin/sh dependecies.sh
	// /bin/sh test.sh

	cmds := []struct {
		Cmd       string
		Breakable bool
	}{
		{"mkdir -p " + opt.BaseFolder, true},
		{"git clone https://" + opt.AdminToken + ":x-oauth-basic@github.com/" + opt.Org + "/" + opt.Repo + ".git" + " " + opt.BaseFolder + opt.DestFolder + "/", true},
		{"git clone https://" + opt.AdminToken + ":x-oauth-basic@github.com/" + opt.Org + "/" + git.TestRepoName + ".git" + " " + opt.BaseFolder + git.TestRepoName + "/", true},
		{"/bin/bash -c \"cp -rf \"" + opt.BaseFolder + git.TestRepoName + "/*\" \"" + opt.BaseFolder + opt.DestFolder + "/\" \"", true},

		{"chmod 777 " + opt.BaseFolder + opt.DestFolder + "/dependencies.sh", true},
		{"/bin/sh -c \"(cd \"" + opt.BaseFolder + opt.DestFolder + "/\" && ./dependencies.sh)\"", true},
		{"chmod 777 " + opt.BaseFolder + opt.DestFolder + "/" + opt.LabFolder + "/test.sh", true},
		{"/bin/sh -c \"(cd \"" + opt.BaseFolder + opt.DestFolder + "/" + opt.LabFolder + "/\" && ./test.sh)\"", false},
	}

	r, err := NewBuildResult()
	if err != nil {
		log.Println(err)
		return
	}

	r.Log = append(r.Log, startMsg)
	r.Course = opt.Org
	r.Timestamp = time.Now()
	r.PushTime = time.Now()
	r.User = opt.User
	r.Status = "Active lab assignment"
	r.Labnum = opt.LabNumber

	starttime := time.Now()

	// executes build commands
	for _, cmd := range cmds {
		err = execute(&env, cmd.Cmd, r, opt)
		if err != nil {
			r.log(err.Error(), opt)
			log.Println(err)
			if cmd.Breakable {
				r.log("Unexpected end of integration.", opt)
				break
			}
		}
	}

	r.BuildTime = time.Since(starttime)

	//TODO factor this out somewhere else: ResultBuilder??
	// parsing the results
	// SimpleParsing(r)
	if len(r.TestScores) > 0 {
		r.TotalScore = score.Total(r.TestScores)
	} else {
		if r.NumPasses+r.NumFails != 0 {
			r.TotalScore = int((float64(r.NumPasses) / float64(r.NumPasses+r.NumFails)) * 100.0)
		}
	}

	if r.numBuildFailure > 0 {
		r.TotalScore = 0
	}

	defer func() {
		// saves the build results
		if err := r.Save(); err != nil {
			log.Println("Error saving build results:", err)
			return
		}
	}()

	// Build for group assignment. Stores build ID in group.
	if opt.Group > 0 {
		group, err := git.NewGroup(opt.Org, opt.Group, false)
		if err != nil {
			log.Println(err)
			return
		}

		oldbuildID := group.GetLastBuildID(opt.LabNumber)
		if oldbuildID > 0 {
			oldr, err := GetBuildResult(oldbuildID)
			if err != nil {
				log.Println(err)
				return
			}
			r.Status = oldr.Status
			if !opt.IsPush {
				r.PushTime = oldr.PushTime
			}
		}

		group.AddBuildResult(opt.LabNumber, r.ID)

		if err := group.Save(); err != nil {
			group.Unlock()
			log.Println(err)
		}
		// build for single user. Stores build ID to user.
	} else {
		user, err := git.NewMemberFromUsername(opt.User, false)
		if err != nil {
			log.Println(err)
			return
		}

		oldbuildID := user.GetLastBuildID(opt.Org, opt.LabNumber)
		if oldbuildID > 0 {
			oldr, err := GetBuildResult(oldbuildID)
			if err != nil {
				log.Println(err)
				return
			}
			r.Status = oldr.Status
			if !opt.IsPush {
				r.PushTime = oldr.PushTime
			}
		}

		user.AddBuildResult(opt.Org, opt.LabNumber, r.ID)

		if err := user.Save(); err != nil {
			user.Unlock()
			log.Println(err)
		}
	}
}

func execute(v *Virtual, cmd string, l *BuildResult, opt DaemonOptions) error {
	buf := bytes.NewBuffer(make([]byte, 0))
	bufw := bufio.NewWriter(buf)

	//TODO fmt?
	fmt.Println("$", cmd)

	err := v.Execute(cmd, nil, bufw, bufw)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(buf)
	for s.Scan() {
		text := s.Text()
		l.log(text, opt)
	}
	return nil
}

// // GetIntegationResults will find a test result for a user or group.
// func GetIntegationResults(org, user, lab string) (logs Result, err error) {
// 	teststore := GetCIStorage(org, user)
//
// 	if !teststore.Has(lab) {
// 		err = errors.New("Doesn't have any CI logs yet.")
// 		return
// 	}
//
// 	err = teststore.ReadGob(lab, &logs, false)
// 	return
// }
//
// // GetIntegationResultSummary will return a summary of the test results for a user or a group.
// func GetIntegationResultSummary(org, user string) (summary map[string]Result, err error) {
// 	summary = make(map[string]Result)
// 	teststore := GetCIStorage(org, user)
// 	keys := teststore.Keys()
// 	for key := range keys {
// 		var res Result
// 		err = teststore.ReadGob(key, &res, false)
// 		if err != nil {
// 			return
// 		}
// 		res.Log = make([]string, 0)
// 		summary[key] = res
// 	}
// 	return
// }
