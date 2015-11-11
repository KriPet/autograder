package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"text/template"

	"github.com/hfurubotten/autograder/config"
	"github.com/hfurubotten/autograder/database"
	"github.com/hfurubotten/autograder/entities"
	"github.com/hfurubotten/autograder/web"
)

const instructions = `
The first time you start {{.SysName}} you will need to supply a few details
about your host environment, the administrator and the git repository hosting
environment. Currently, we only support GitHub for hosting git repositories.

{{.SysName}} can either read a configuration file with the necessary
information (see the example below), or you can provide these details as
command line arguments (also shown below).

Here is an example {{.ConfigFileName}} file:

{
  "HomepageURL": "http://example.com/",
  "ClientID": "123456789",
  "ClientSecret": "123456789abcdef",
  "BasePath": "/usr/share/{{.SysNameLC}}/"
}

Before you can start you will need to register the {{.SysName}} application
at GitHub; you will need to do this from the administrator account.

1. Go to https://github.com/settings/applications/new
2. Enter the information requested.
   - Application name: e.g. "{{.SysName}} at University of Stavanger"
   - Homepage URL: e.g. "http://{{.SysNameLC}}.ux.uis.no"
   - Authorization callback URL: e.g. "http://{{.SysNameLC}}.ux.uis.no/oauth"

Note that, the Homepage URL must be a fully qualified URL, including http://.
This must be the hostname (or an alias) of server running the '{{.SysNameLC}}'
program. This server must have a public IP address, since GitHub will make calls
to this server to support {{.SysName}}'s functionality. Further, {{.SysName}}
requires that the Authorization callback URL is the same as the Homepage URL
with the added "/oauth" path.

Once you have completed the above steps, the Client ID and Client Secret will be
available from the GitHub web interface. Simply copy each of these OAuth tokens
and paste them into the configuration file, or on the command line when starting
{{.SysName}} for the first time. You will not need to repeat this process
when starting {{.SysName}} in the future.

If you need to obtain the OAuth tokens at a later time, e.g. if you have deleted
the configuration file, go to: https://github.com/settings/developers and
select your Application to be able to view the OAuth tokens again.

`

var (
	admin        = flag.String("admin", "", "Admin must be a valid GitHub username")
	url          = flag.String("url", "", "Homepage URL for "+config.SysName)
	clientID     = flag.String("id", "", "Client ID for OAuth with Github")
	clientSecret = flag.String("secret", "", "Client Secret for OAuth with Github")
	path         = flag.String("path", config.StdPath, "Path for data storage for "+config.SysName)
	help         = flag.Bool("help", false, "Helpful instructions")
)

func main() {
	flag.Parse()

	// set log print appearance
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// print instructions and command usage
	if *help {
		data := struct {
			SysName, SysNameLC, ConfigFileName string
		}{
			config.SysName, config.SysNameLC, config.FileName,
		}
		t := template.Must(template.New("instructions").Parse(instructions))
		err := t.Execute(os.Stdout, data)
		if err != nil {
			log.Fatalln(err)
		}
		flag.Usage()
		return
	}

	// load configuration data from the provided base path
	if *path != "" {
		conf, err := config.Load(*path)
		if err != nil {
			log.Println(err)
			// can't load config file; create config based on command line arguments
			conf, err = config.NewConfig(*url, *clientID, *clientSecret, *path)
			if err != nil {
				// can't continue without proper configuration
				log.Fatal(err)
			}
			if err := conf.Save(); err != nil {
				log.Fatal(err)
			}
		}
		// set global configuration struct; will be accessible through config.Get()
		conf.SetCurrent()
	}

	// start database
	if err := database.Start(config.Get().BasePath); err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// checks for an admin username
	if *admin != "" {
		log.Println("New admin added to the system: ", *admin)
		m, err := entities.GetMember(*admin)
		if err != nil {
			log.Fatal(err)
		}

		m.IsAdmin = true
		err = m.Save()
		if err != nil {
			m.Unlock()
			log.Println("Couldn't store admin user in system:", err)
		}
	}

	// TODO: checks if the system should be set up as a deamon that starts on system startup.

	// TODO: checks for docker installation
	// TODO: install on supported systems
	// TODO: give notice for those systems not supported

	log.Println("Starting webserver")
	server := web.NewServer(80)
	server.Start()

	// prevent main from returning immediately; wait for interrupt
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Kill, os.Interrupt)
	<-signalChan
	log.Println("Application shutdown by user.")
}
