// Copyright 2016 EF CTX. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru/cmd"
)

type projectCreate struct {
	fs       *gnuflag.FlagSet
	name     string
	platform string
	team     string
	plan     string
	envs     commaSeparatedFlag
}

func (*projectCreate) Info() *cmd.Info {
	return &cmd.Info{
		Name: "project-create",
		Desc: "creates a remote project in the tranor server",
	}
}

func (c *projectCreate) Run(ctx *cmd.Context, client *cmd.Client) error {
	config, err := loadConfigFile()
	if err != nil {
		return errors.New("unable to load environments file, please make sure that tranor is properly configured")
	}
	err = c.envs.validate(config.envNames())
	if err != nil {
		return fmt.Errorf("failed to load environments: %s", err)
	}
	envs := c.filterEnvironments(config.Environments, c.envs.Values())
	apps, err := c.createApps(envs, client)
	if err != nil {
		return err
	}
	err = c.setCNames(apps, client)
	if err != nil {
		c.deleteApps(apps, client)
		return fmt.Errorf("failed to configure project %q: %s", c.name, err)
	}
	fmt.Fprintf(ctx.Stdout, "successfully created the project %q!\n", c.name)
	if gitRepo := apps[0]["repository_url"]; gitRepo != "" {
		fmt.Fprintf(ctx.Stdout, "Git repository: %s\n", gitRepo)
	}
	return nil
}

func (c *projectCreate) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("project-create", gnuflag.ExitOnError)
		c.fs.StringVar(&c.name, "name", "", "name of the project")
		c.fs.StringVar(&c.name, "n", "", "name of the project")
		c.fs.StringVar(&c.platform, "platform", "", "platform of the project")
		c.fs.StringVar(&c.platform, "l", "", "platform of the project")
		c.fs.StringVar(&c.team, "team", "", "team that owns the project")
		c.fs.StringVar(&c.team, "t", "", "team that owns the project")
		c.fs.StringVar(&c.plan, "plan", "", "plan to use for the project")
		c.fs.StringVar(&c.plan, "p", "", "plan to use for the project")
		c.fs.Var(&c.envs, "envs", "comma-separated list of environments to use (defaults to dev,qa,stage,production)")
		c.fs.Var(&c.envs, "e", "comma-separated list of environments to use (defaults to env,qa,stage,production)")
		c.envs.Set("dev,qa,stage,production")
	}
	return c.fs
}

func (c *projectCreate) createApps(envs []Environment, client *cmd.Client) ([]map[string]string, error) {
	createdApps := make([]map[string]string, 0, len(envs))
	for _, env := range envs {
		appName := fmt.Sprintf("%s-%s", c.name, env.Name)
		app, err := createApp(client, createAppOptions{
			name:     appName,
			plan:     c.plan,
			platform: c.platform,
			pool:     env.poolName(),
			team:     c.team,
		})
		if err != nil {
			c.deleteApps(createdApps, client)
			return nil, fmt.Errorf("failed to create the project in env %q: %s", env.Name, err)
		}
		app["name"] = appName
		app["dnsSuffix"] = env.DNSSuffix
		createdApps = append(createdApps, app)
	}
	return createdApps, nil
}

func (c *projectCreate) deleteApps(apps []map[string]string, client *cmd.Client) error {
	for _, app := range apps {
		url, err := cmd.GetURL("/apps/" + app["name"])
		if err != nil {
			return err
		}
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}
	return nil
}

func (c *projectCreate) setCNames(apps []map[string]string, client *cmd.Client) error {
	for _, app := range apps {
		reqURL, err := cmd.GetURL(fmt.Sprintf("/apps/%s/cname", app["name"]))
		if err != nil {
			return err
		}
		cname := fmt.Sprintf("%s.%s", c.name, app["dnsSuffix"])
		v := make(url.Values)
		v.Set("cname", cname)
		req, err := http.NewRequest("POST", reqURL, strings.NewReader(v.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}
	return nil
}

func (c *projectCreate) filterEnvironments(envs []Environment, names []string) []Environment {
	var filtered []Environment
	for _, e := range envs {
		for _, name := range names {
			if e.Name == name {
				filtered = append(filtered, e)
				break
			}
		}
	}
	return filtered
}

type commaSeparatedFlag struct {
	values []string
}

func (f *commaSeparatedFlag) Values() []string {
	return f.values
}

func (f *commaSeparatedFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *commaSeparatedFlag) Set(v string) error {
	f.values = strings.Split(v, ",")
	return nil
}

func (f *commaSeparatedFlag) validate(validValues []string) error {
	var invalidValues []string
	for _, cv := range f.values {
		var found bool
		for _, validValue := range validValues {
			if validValue == cv {
				found = true
				break
			}
		}
		if !found {
			invalidValues = append(invalidValues, cv)
		}
	}
	if len(invalidValues) > 0 {
		return fmt.Errorf("invalid values: %s (valid options are: %s)", strings.Join(invalidValues, ", "), strings.Join(validValues, ", "))
	}
	return nil
}