// Copyright 2016 EF CTX. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/tsuru/tsuru/cmd"
)

func TestProjectEnvVarSetInfo(t *testing.T) {
	info := (&projectEnvVarSet{}).Info()
	if info == nil {
		t.Fatal("unexpected <nil> info")
	}
	if info.Name != "envvar-set" {
		t.Errorf("wrong name. want %q. got %q", "envvar-set", info.Name)
	}
	if info.MinArgs != 1 {
		t.Errorf("wrong min args. want 1. got %d", info.MinArgs)
	}
}

func TestProjectEnvVarSetMissingName(t *testing.T) {
	var c projectEnvVarSet
	err := c.Flags().Parse(true, []string{"--no-restart"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
	expectedMsg := "please provide the name of the project"
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectEnvVarSetInvalidFormat(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	cleanup, err := setupFakeConfig(server.url(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectEnvVarSet
	err = c.Flags().Parse(true, []string{"-n", "proj1"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr, Args: []string{"ENV"}}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
	expectedMsg := "configuration vars must be specified in the form NAME=value"
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectEnvVarSetNoConfiguration(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectEnvVarSet
	err = c.Flags().Parse(true, []string{"-n", "proj1"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	err = c.Run(&ctx, nil)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
}

func TestProjectEnvVarSetErrorInOneOfTheApps(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  http.MethodPost,
		path:    "/apps/proj1-stage/env",
		code:    http.StatusInternalServerError,
		payload: []byte("something went wrong"),
	})
	appNames := []string{"proj1-dev", "proj1-prod"}
	for _, appName := range appNames {
		server.prepareResponse(preparedResponse{
			method:  http.MethodPost,
			path:    "/apps/" + appName + "/env",
			code:    http.StatusOK,
			payload: []byte("{}"),
		})
	}
	cleanup, err := setupFakeConfig(server.url(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectEnvVarSet
	err = c.Flags().Parse(true, []string{
		"-n", "proj1",
		"-e", "dev,stage,prod",
		"--no-restart",
		"-p",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"USER_NAME=root", "USER_PASSWORD=r00t", `PREFERRED_TEAM="some nice team"`},
	}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("got unexpected <nil> error")
	}
	expectedOutput := `setting variables in environment "dev"... ok
setting variables in environment "stage"... failed
setting variables in environment "prod"... ok
`
	if stdout.String() != expectedOutput {
		t.Errorf("wrong output\nwant:\n%s\ngot:\n%s", expectedOutput, stdout.String())
	}
}

func TestProjectEnvVarGetInfo(t *testing.T) {
	info := (&projectEnvVarGet{}).Info()
	if info == nil {
		t.Fatal("unexpected <nil> info")
	}
	if info.Name != "envvar-get" {
		t.Errorf("wrong name. want %q. got %q", "envvar-get", info.Name)
	}
}

func TestProjectEnvVarGetNoConfiguration(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectEnvVarGet
	err = c.Flags().Parse(true, []string{"-n", "proj1"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	err = c.Run(&ctx, nil)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
}

func TestProjectEnvVarGetMissingName(t *testing.T) {
	var c projectEnvVarGet
	err := c.Flags().Parse(true, []string{"-e", "dev,stage,prod"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
	expectedMsg := "please provide the name of the project"
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectEnvVarGetErrorInOneOfTheApps(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  http.MethodGet,
		path:    "/apps/proj1-stage/env",
		code:    http.StatusInternalServerError,
		payload: []byte("something went wrong"),
	})
	appNames := []string{"proj1-dev", "proj1-prod"}
	for _, appName := range appNames {
		rawPayload := []map[string]interface{}{
			{
				"name":   "APP_NAME",
				"value":  appName,
				"public": true,
			},
			{
				"name":   "USER_NAME",
				"value":  "root",
				"public": true,
			},
			{
				"name":   "USER_PASSWORD",
				"value":  "r00t",
				"public": false,
			},
		}
		payload, _ := json.Marshal(rawPayload)
		server.prepareResponse(preparedResponse{
			method:  http.MethodGet,
			path:    "/apps/" + appName + "/env",
			code:    http.StatusOK,
			payload: payload,
		})
	}
	cleanup, err := setupFakeConfig(server.url(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectEnvVarGet
	err = c.Flags().Parse(true, []string{
		"-n", "proj1",
		"-e", "dev,stage,prod",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
	expectedOutput := `variables in "dev":

 APP_NAME=proj1-dev
 USER_NAME=root
 USER_PASSWORD=*** (private variable)


`
	if stdout.String() != expectedOutput {
		t.Errorf("wrong output\nwant:\n%q\ngot:\n%q", expectedOutput, stdout.String())
	}
}

func TestProjectEnvVarUnsetInfo(t *testing.T) {
	info := (&projectEnvVarUnset{}).Info()
	if info == nil {
		t.Fatal("unexpected <nil> info")
	}
	if info.Name != "envvar-unset" {
		t.Errorf("wrong name. want %q. got %q", "envvar-unset", info.Name)
	}
	if info.MinArgs != 1 {
		t.Errorf("wrong min args. want 1. got %d", info.MinArgs)
	}
}

func TestProjectEnvVarUnsetNoConfiguration(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectEnvVarUnset
	err = c.Flags().Parse(true, []string{"-n", "proj1"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	err = c.Run(&ctx, nil)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
}

func TestProjectEnvVarUnsetMissingName(t *testing.T) {
	var c projectEnvVarUnset
	err := c.Flags().Parse(true, []string{"-e", "dev,stage,prod"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
	expectedMsg := "please provide the name of the project"
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectEnvVarUnsetError(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:   http.MethodDelete,
		path:     "/apps/proj1-stage/env",
		code:     http.StatusInternalServerError,
		payload:  []byte("something went wrong"),
		ignoreQS: true,
	})
	appNames := []string{"proj1-dev", "proj1-prod"}
	for _, appName := range appNames {
		server.prepareResponse(preparedResponse{
			method:   http.MethodDelete,
			path:     "/apps/" + appName + "/env",
			code:     http.StatusOK,
			payload:  []byte("{}"),
			ignoreQS: true,
		})
	}
	cleanup, err := setupFakeConfig(server.url(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectEnvVarUnset
	err = c.Flags().Parse(true, []string{
		"-n", "proj1",
		"-e", "dev,stage,prod",
		"--no-restart",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Args:   []string{"USER_NAME", "USER_PASSWORD", "PREFERRED_TEAM"},
	}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
	expectedOutput := `unsetting variables from environment "dev"... ok
unsetting variables from environment "stage"... failed
unsetting variables from environment "prod"... ok
`
	if stdout.String() != expectedOutput {
		t.Errorf("wrong output\nwant:\n%s\ngot:\n%s", expectedOutput, stdout.String())
	}
}
