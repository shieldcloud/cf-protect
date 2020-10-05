package main

import (
	"encoding/json"
	fmt "github.com/jhunt/go-ansi"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/jhunt/vcaptive"
	"github.com/shieldproject/shield/client/v2/shield"
)

type AppEnv struct {
	System struct {
		Services map[string]interface{} `json:"VCAP_SERVICES"`
	} `json:"system_env_json"`
}

type Plugin struct{}

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

	fmt.Printf("@*{protecting} %s [@W{%s}]\n", app.Name, app.Guid)

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

	if inst, found := services.Tagged("mysql"); found {
		fmt.Printf("protecting @G{%s} (mysql):\n", inst.Name)

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
			fmt.Printf("  username: @W{%s}\n", username)
		}

		password, ok := inst.GetString("password")
		if ok {
			fmt.Printf("  password: @W{%s}\n", password)
		}

		// go talk to shield
		cli := shield.Client{
			URL:                os.Getenv("SHIELD_URL"),
			Debug:              os.Getenv("SHIELD_DEBUG") == "yes",
			Trace:              os.Getenv("SHIELD_TRACE") == "yes",
			InsecureSkipVerify: true,
		}
		err := cli.Authenticate(&shield.LocalAuth{
			Username: os.Getenv("SHIELD_USERNAME"),
			Password: os.Getenv("SHIELD_PASSWORD"),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
			os.Exit(2)
		}

		t, err := cli.CreateTarget(&shield.Target{
			Name:   fmt.Sprintf("%s - %s MySQL", app.Name, inst.Name),
			Plugin: "mysql",
			Agent:  os.Getenv("SHIELD_AGENT"),
			Config: map[string]interface{}{
				"host":     fmt.Sprintf("%s:%s", hostname, port),
				"username": username,
				"password": password,
				"database": db,
				"options":  "--ssl-mode=disabled --column-statistics=0", // FIXME
			},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
			os.Exit(2)
		}

		j, err := cli.CreateJob(&shield.Job{
			Name:       "Daily",
			Schedule:   "daily 4am",
			Retain:     "4d",
			Paused:     true,
			Bucket:     "storage",
			TargetUUID: t.UUID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!!!} %s\n", err)
			os.Exit(2)
		}

		fmt.Printf("created job [%s] for target [%s]...\n", j.UUID, t.UUID)
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
