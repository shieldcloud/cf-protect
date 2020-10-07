package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	fmt "github.com/jhunt/go-ansi"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/jhunt/vcaptive"
	"github.com/shieldproject/shield/client/v2/shield"
)

type AppEnv struct {
	System struct {
		Services map[string]interface{} `json:"VCAP_SERVICES"`
	} `json:"system_env_json"`
}

type ShieldInfo struct {
	Client *shield.Client
	Agent  string
}

const mysql = "mysql"

type Plugin struct{}

func getShieldInfo() *ShieldInfo {
	fmt.Printf("Connecting to SHIELD...\n")

	agent := os.Getenv("SHIELD_AGENT")
	if agent == "" {
		fmt.Fprintf(os.Stderr, "@R{!!!} SHIELD_AGENT not found\n")
		os.Exit(2)
	}

	cli, err, found := shield.EnvConfig()
	if !found {
		path := fmt.Sprintf("%s/.shield", os.Getenv("HOME"))
		config, err := shield.ReadConfig(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} unable to read SHIELD configuration from %s: %s\n", path, err)
			os.Exit(2)
		}
		cli, err = config.Client(os.Getenv("SHIELD_CORE")) // FIXME sort of
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} configuration failed: %s\n", err)
		os.Exit(2)
	}
	if cli == nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} configuration failed: unknown failure\n")
		os.Exit(2)
	}

	return &ShieldInfo{
		Client: cli,
		Agent:  agent,
	}
}

func protectMySQL(target string, inst vcaptive.Instance, shieldInfo ShieldInfo) {
	fmt.Printf("\n")
	fmt.Printf("protecting @W{service} @G{%s} (mysql):\n", inst.Name)

	hostname, ok := inst.GetString("hostname")
	if ok {
		fmt.Printf("  hostname: @W{%s}\n", hostname)
	}

	port, ok := inst.GetString("port")
	if ok {
		fmt.Printf("  port:     @W{%s}\n", port)
	}

	db, ok := inst.GetString("name")
	if ok {
		fmt.Printf("  database: @W{%s}\n", db)
	}

	username, ok := inst.GetString("username")
	if ok {
		fmt.Printf("  username: @W{%s}\n", strings.Repeat("*", utf8.RuneCountInString(username)))
	}

	password, ok := inst.GetString("password")
	if ok {
		fmt.Printf("  password: @W{%s}\n", strings.Repeat("*", utf8.RuneCountInString(password)))
	}

	targets, err := shieldInfo.Client.ListTargets(&shield.TargetFilter{
		Name:  target,
		Fuzzy: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} Failed to search targets: %s\n", err)
		os.Exit(2)
	}
	if len(targets) > 1 {
		fmt.Fprintf(os.Stderr, "@R{!!!} Multiple targets found named '%s'\n", target)
		fmt.Fprintf(os.Stderr, "@Y{found:}\n")
		for _, t := range targets {
			fmt.Fprintf(os.Stderr, "  - %s\n", t.Name)
		}
		os.Exit(2)
	}

	tverb := "created"
	t := &shield.Target{
		Name:   target,
		Plugin: "mysql",
		Agent:  shieldInfo.Agent,
		Config: map[string]interface{}{
			"host":     fmt.Sprintf("%s:%s", hostname, port),
			"username": username,
			"password": password,
			"database": db,
			"options":  "--ssl-mode=disabled --column-statistics=0", // FIXME
		},
	}
	if len(targets) == 0 {
		t, err = shieldInfo.Client.CreateTarget(t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} Failed to create target: %s\n", err)
			os.Exit(2)
		}
	} else {
		tverb = "updated"
		t.UUID = targets[0].UUID
		_, err = shieldInfo.Client.UpdateTarget(t)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} Failed to update target: %s\n", err)
			os.Exit(2)
		}
	}

	jobs, err := shieldInfo.Client.ListJobs(&shield.JobFilter{
		Name:   "Daily",
		Fuzzy:  false,
		Target: t.UUID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} Failed to search jobs: %s\n", err)
		os.Exit(2)
	}
	if len(jobs) > 1 {
		fmt.Fprintf(os.Stderr, "@R{!!!} Multiple jobs found for '%s'\n", target)
		fmt.Fprintf(os.Stderr, "@Y{found:}\n")
		for _, j := range jobs {
			fmt.Fprintf(os.Stderr, "  - %s / %s\n", j.Target.Name, j.Name)
		}
		os.Exit(2)
	}

	jverb := "created"
	j := &shield.Job{
		Name:       "Daily",
		Schedule:   "daily 4am",
		Retain:     "4d",
		Paused:     true,
		Bucket:     "storage",
		TargetUUID: t.UUID,
	}
	if len(jobs) == 0 {
		j, err = shieldInfo.Client.CreateJob(j)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} Failed to create job: %s\n", err)
			os.Exit(2)
		}
	} else {
		jverb = "updated"
		j.UUID = jobs[0].UUID
		j, err = shieldInfo.Client.UpdateJob(j)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} Failed to update job: %s\n", err)
			os.Exit(2)
		}
	}

	fmt.Printf("\n")
	fmt.Printf("%s system @G{%s} [%s]...\n", tverb, t.Name, t.UUID)
	fmt.Printf("%s job @G{%s} [%s]...\n", jverb, j.Name, j.UUID)
	fmt.Printf("@B{%s/#!/systems/system:uuid:%s}\n", shieldInfo.Client.URL, t.UUID)
}

func (p Plugin) Run(c plugin.CliConnection, args []string) {
	cmd := args[0]
	if cmd == "CLI-MESSAGE-UNINSTALL" {
		os.Exit(0)
	}

	if cmd != "protect" {
		fmt.Fprintf(os.Stderr, "@R{!!!} Unrecognized command @Y{%s}\n", cmd)
		os.Exit(1)
	}

	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "@R{!!!} Missing required APP name\n")
		os.Exit(1)
	}

	app, err := c.GetApp(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}

	api, err := c.ApiEndpoint()
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}

	tok, err := c.AccessToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}

	fmt.Printf("protecting @W{application} @M{%s}\n", app.Name)

	req, err := http.NewRequest("GET", api+"/v2/apps/"+app.Guid+"/env", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}
	req.Header.Set("Authorization", tok)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}

	var env AppEnv
	err = json.Unmarshal(b, &env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}
	services, err := vcaptive.ParseServices(env.System.Services)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
		os.Exit(2)
	}

	org, err := c.GetCurrentOrg()
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} Unable to get current Cloud Foundry org: %s\n", err)
		os.Exit(2)
	}

	space, err := c.GetCurrentSpace()
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!!!} Unable to get current Cloud Foundry space: %s\n", err)
		os.Exit(2)
	}

	shieldInfo := getShieldInfo()

	if inst, found := services.Tagged(mysql); found {
		protectMySQL(fmt.Sprintf("%s/%s/%s/%s", org.Name, space.Name, app.Name, inst.Name), inst, *shieldInfo)
	}
}

func (Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "protect",
		Version: getVersion(Version),
		Commands: []plugin.Command{
			{
				Name:     "protect",
				HelpText: "Protect the Data Services bound to an application, using SHIELD Cloud <https://shieldcloud.io>",
			},
		},
	}
}

func main() {
	plugin.Start(&Plugin{})
}
