// Copyright 2016 EF CTX. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/tsuru/gnuflag"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
)

func TestCommaSeparatedFlag(t *testing.T) {
	var value gnuflag.Value
	value = &commaSeparatedFlag{}
	input := "dev,qa,staging,prod"
	err := value.Set(input)
	if err != nil {
		t.Fatal(err)
	}
	output := value.String()
	if output != input {
		t.Errorf("wrong output.\nwant %q\ngot  %q", input, output)
	}
	values := value.(*commaSeparatedFlag).Values()
	expectedValues := []string{"dev", "qa", "staging", "prod"}
	if !reflect.DeepEqual(values, expectedValues) {
		t.Errorf("wrong value list\nwant %#v\ngot  %#v", expectedValues, values)
	}
}

func TestCommaSeparatedFlagValidate(t *testing.T) {
	var tests = []struct {
		testCase      string
		flagInput     string
		validValues   []string
		expectedError string
	}{
		{
			"valid values",
			"dev,qa,staging,prod",
			[]string{"dev", "qa", "staging", "demo", "prod"},
			"",
		},
		{
			"one invalid value",
			"dev,qa,staging,prod",
			[]string{"dev", "qa", "demo", "prod"},
			"invalid values: staging (valid options are: dev, qa, demo, prod)",
		},
		{
			"all invalid values",
			"dev,qa,staging,prod",
			nil,
			"invalid values: dev, qa, staging, prod (valid options are: )",
		},
	}
	for _, test := range tests {
		var value commaSeparatedFlag
		err := value.Set(test.flagInput)
		if err != nil {
			t.Errorf("%s: %s", test.testCase, err)
			continue
		}
		err = value.validate(test.validValues)
		if err == nil {
			err = errors.New("")
		}
		if err.Error() != test.expectedError {
			t.Errorf("wrong error message\nwant %q\ngot  %q", test.expectedError, err.Error())
		}
	}
}

func TestProjectCreateInfo(t *testing.T) {
	info := (&projectCreate{}).Info()
	if info == nil {
		t.Fatal("unexpected <nil> info")
	}
	if info.Name != "project-create" {
		t.Errorf("wrong name. Want %q. Got %q", "project-create", info.Name)
	}
}

func TestProjectCreateNoRepo(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "POST",
		path:    "/apps",
		code:    http.StatusCreated,
		payload: []byte(`{}`),
	})
	appNames := []string{"superproj-dev", "superproj-prod"}
	for _, appName := range appNames {
		server.prepareResponse(preparedResponse{
			method:  "POST",
			path:    "/apps/" + appName + "/cname",
			code:    http.StatusOK,
			payload: []byte(`{}`),
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectCreate
	err = c.Flags().Parse(true, []string{
		"-n", "superproj",
		"-l", "python",
		"-t", "myteam",
		"-p", "medium",
		"-e", "dev,prod",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	expectedOutput := `successfully created the project "superproj"!
`
	if stdout.String() != expectedOutput {
		t.Errorf("wrong output returned\nWant: %s\nGot:  %s", expectedOutput, stdout.String())
	}
}

func TestProjectCreateMissingParams(t *testing.T) {
	var tests = []struct {
		testCase string
		flags    []string
	}{
		{
			"missing name",
			[]string{"-l", "python"},
		},
		{
			"missing platform",
			[]string{"-n", "superproj"},
		},
		{
			"missing all flags",
			[]string{},
		},
	}
	cleanup, err := setupFakeTarget("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	for _, test := range tests {
		var c projectCreate
		err = c.Flags().Parse(true, test.flags)
		if err != nil {
			t.Errorf("%s: %s", test.testCase, err)
			continue
		}
		var stdout, stderr bytes.Buffer
		ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
		client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
		err = c.Run(&ctx, client)
		if err == nil {
			t.Errorf("%s: unexpected <nil> error", test.testCase)
			continue
		}
		expectedMsg := "please provide the name and the platform"
		if err.Error() != expectedMsg {
			t.Errorf("%s: wrong error message\nwant %q\ngot  %q", test.testCase, expectedMsg, err.Error())
		}
	}
}

func TestProjectCreateInvalidEnv(t *testing.T) {
	cleanup, err := setupFakeTarget("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectCreate
	err = c.Flags().Parse(true, []string{
		"-n", "superproj",
		"-l", "python",
		"-t", "myteam",
		"-p", "medium",
		"-e", "dev,production",
	})
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

func TestProjectCreateFailToLoadConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectCreate
	err = c.Flags().Parse(true, []string{
		"-n", "superproj",
		"-l", "python",
		"-t", "myteam",
		"-p", "medium",
	})
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

func TestProjectCreateFailToSetCNames(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "POST",
		path:    "/apps",
		code:    http.StatusCreated,
		payload: []byte(`{"repository_url":"git@git.example.com:superproj-dev.git"}`),
	})
	server.prepareResponse(preparedResponse{
		method:  "POST",
		path:    "/apps/superproj-dev/cname",
		code:    http.StatusOK,
		payload: []byte(`{}`),
	})
	server.prepareResponse(preparedResponse{
		method: "DELETE",
		path:   "/apps/superproj-prod",
		code:   http.StatusOK,
	})
	server.prepareResponse(preparedResponse{
		method:  "POST",
		path:    "/apps/superproj-prod/cname",
		code:    http.StatusInternalServerError,
		payload: []byte(`{}`),
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectCreate
	err = c.Flags().Parse(true, []string{
		"-n", "superproj",
		"-l", "python",
		"-t", "myteam",
		"-p", "medium",
		"-e", "dev,prod",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	err = c.Run(&ctx, cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{}))
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
	expectedMethods := []string{"POST", "POST", "POST", "POST", "DELETE", "DELETE"}
	expectedPaths := []string{
		"/1.0/apps", "/1.0/apps",
		"/1.0/apps/superproj-dev/cname",
		"/1.0/apps/superproj-prod/cname",
		"/1.0/apps/superproj-dev",
		"/1.0/apps/superproj-prod",
	}
	if len(server.reqs) != len(expectedPaths) {
		t.Fatalf("wrong number of requests sent to the server. Want %d. Got %d", len(expectedPaths), len(server.reqs))
	}
	for i, req := range server.reqs {
		if req.Method != expectedMethods[i] {
			t.Errorf("wrong method. Want %q. Got %q", expectedMethods[i], req.Method)
		}
		if req.URL.Path != expectedPaths[i] {
			t.Errorf("wrong path. Want %q. Got %q", expectedPaths[i], req.URL.Path)
		}
	}
}

func TestProjectUpdateMissingName(t *testing.T) {
	var c projectUpdate
	err := c.Flags().Parse(true, []string{
		"--add-envs", "qa,stage",
		"--remove-envs", "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("got unexpected <nil> error")
	}
	expectedMsg := "please provide the name of the project"
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectUpdateNotFound(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method: "GET",
		path:   "/apps?name=" + url.QueryEscape("^proj3"),
		code:   http.StatusNoContent,
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "proj3",
		"-d", "updated project description",
		"-t", "superteam",
		"-p", "huge",
		"--add-envs", "qa,stage",
		"--remove-envs", "dev",
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
	expectedMsg := "project not found"
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectUpdateFailToCreateNewApps(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=" + url.QueryEscape("^proj3"),
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	appRespMap := map[string][]byte{
		"proj3-dev":  []byte(appInfo5),
		"proj3-prod": []byte(appInfo6),
	}
	for appName, payload := range appRespMap {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/apps/" + appName,
			code:    http.StatusOK,
			payload: payload,
		})
	}
	server.prepareResponse(preparedResponse{
		method:  "POST",
		path:    "/apps",
		code:    http.StatusInternalServerError,
		payload: []byte(`{}`),
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "proj3",
		"-d", "updated project description",
		"-t", "superteam",
		"-p", "huge",
		"--add-envs", "qa,stage",
		"--remove-envs", "dev",
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
}

func TestProjectUpdateFailToSetCNames(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=" + url.QueryEscape("^proj3"),
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	appRespMap := map[string][]byte{
		"proj3-dev":  []byte(appInfo5),
		"proj3-prod": []byte(appInfo6),
	}
	for appName, payload := range appRespMap {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/apps/" + appName,
			code:    http.StatusOK,
			payload: payload,
		})
	}
	server.prepareResponse(preparedResponse{
		method:  "POST",
		path:    "/apps",
		code:    http.StatusCreated,
		payload: []byte(`{}`),
	})
	server.prepareResponse(preparedResponse{
		method: "POST",
		path:   "/apps/proj3-qa/cname",
		code:   http.StatusOK,
	})
	server.prepareResponse(preparedResponse{
		method: "POST",
		path:   "/apps/proj3-stage/cname",
		code:   http.StatusInternalServerError,
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "proj3",
		"-d", "updated project description",
		"-t", "superteam",
		"-p", "huge",
		"--add-envs", "qa,stage",
		"--remove-envs", "dev",
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
}

func TestProjectUpdateFailToUpdate(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=" + url.QueryEscape("^proj3"),
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	appRespMap := map[string][]byte{
		"proj3-dev":  []byte(appInfo5),
		"proj3-prod": []byte(appInfo6),
	}
	for appName, payload := range appRespMap {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/apps/" + appName,
			code:    http.StatusOK,
			payload: payload,
		})
	}
	server.prepareResponse(preparedResponse{
		method: "DELETE",
		path:   "/apps/proj3-dev",
		code:   http.StatusOK,
	})
	server.prepareResponse(preparedResponse{
		method: "PUT",
		path:   "/apps/proj3-prod",
		code:   http.StatusInternalServerError,
	})
	server.prepareResponse(preparedResponse{
		method:  "POST",
		path:    "/apps",
		code:    http.StatusCreated,
		payload: []byte(`{}`),
	})
	server.prepareResponse(preparedResponse{
		method: "POST",
		path:   "/apps/proj3-qa/cname",
		code:   http.StatusOK,
	})
	server.prepareResponse(preparedResponse{
		method: "POST",
		path:   "/apps/proj3-stage/cname",
		code:   http.StatusOK,
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "proj3",
		"-d", "updated project description",
		"-t", "superteam",
		"-p", "huge",
		"--add-envs", "qa,stage",
		"--remove-envs", "dev",
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
}

func TestProjectUpdateInvalidNewEnv(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=" + url.QueryEscape("^proj3"),
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	appRespMap := map[string][]byte{
		"proj3-dev":  []byte(appInfo5),
		"proj3-prod": []byte(appInfo6),
	}
	for appName, payload := range appRespMap {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/apps/" + appName,
			code:    http.StatusOK,
			payload: payload,
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "proj3",
		"--add-envs", "qa2,stg",
		"--remove-envs", "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> errror")
	}
}

func TestProjectUpdateDuplicateEnv(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=" + url.QueryEscape("^proj3"),
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	appRespMap := map[string][]byte{
		"proj3-dev":  []byte(appInfo5),
		"proj3-prod": []byte(appInfo6),
	}
	for appName, payload := range appRespMap {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/apps/" + appName,
			code:    http.StatusOK,
			payload: payload,
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "proj3",
		"--add-envs", "qa,prod",
		"--remove-envs", "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> errror")
	}
	expectedMsg := `env "prod" is already defined in this project`
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectUpdateInvalidRemoveEnvs(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=" + url.QueryEscape("^proj3"),
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	appRespMap := map[string][]byte{
		"proj3-dev":  []byte(appInfo5),
		"proj3-prod": []byte(appInfo6),
	}
	for appName, payload := range appRespMap {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/apps/" + appName,
			code:    http.StatusOK,
			payload: payload,
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "proj3",
		"--add-envs", "qa",
		"--remove-envs", "stage",
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> errror")
	}
	expectedMsg := `env "stage" is not defined in this project`
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectUpdateFailToLoadConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectUpdate
	err = c.Flags().Parse(true, []string{
		"-n", "superproj",
		"-t", "myteam",
		"-p", "medium",
	})
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

func TestRemoveProject(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	appNames := []string{"myproj-dev", "myproj-qa", "myproj-stage", "myproj-prod"}
	expectedPaths := make([]string, len(appNames))
	for i, appName := range appNames {
		path := "/apps/" + appName
		expectedPaths[i] = path
		server.prepareResponse(preparedResponse{
			method: "DELETE",
			path:   path,
			code:   http.StatusOK,
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectRemove
	err = c.Flags().Parse(true, []string{"-yn", "myproj"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(server.reqs) != len(expectedPaths) {
		t.Fatalf("wrong number of requests. Want %d. Got %d", len(expectedPaths), len(server.reqs))
	}
	for i, req := range server.reqs {
		if req.Method != "DELETE" {
			t.Errorf("wrong method. Want DELETE. Got %s", req.Method)
		}
		if p := strings.Replace(req.URL.Path, "/1.0", "", 1); p != expectedPaths[i] {
			t.Errorf("wrong path\nwant %q\ngot  %q", expectedPaths[i], p)
		}
	}
	expectedOutput := `Deleting from env "dev"... ok
Deleting from env "qa"... ok
Deleting from env "stage"... ok
Deleting from env "prod"... ok
`
	if stdout.String() != expectedOutput {
		t.Errorf("Wrong output\nWant:\n%s\nGot:\n%s", expectedOutput, stdout.String())
	}
}

func TestRemoveProjectSomeEnvironments(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	appNames := []string{"myproj-dev", "myproj-qa", "myproj-stage", "myproj-prod"}
	expectedPaths := make([]string, len(appNames))
	for i, appName := range appNames {
		code := http.StatusOK
		if i%2 == 0 {
			code = http.StatusNotFound
		}
		path := "/apps/" + appName
		expectedPaths[i] = path
		server.prepareResponse(preparedResponse{
			method: "DELETE",
			path:   path,
			code:   code,
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectRemove
	err = c.Flags().Parse(true, []string{"-yn", "myproj"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(server.reqs) != len(expectedPaths) {
		t.Fatalf("wrong number of requests. Want %d. Got %d", len(expectedPaths), len(server.reqs))
	}
	for i, req := range server.reqs {
		if req.Method != "DELETE" {
			t.Errorf("wrong method. Want DELETE. Got %s", req.Method)
		}
		if p := strings.Replace(req.URL.Path, "/1.0", "", 1); p != expectedPaths[i] {
			t.Errorf("wrong path\nwant %q\ngot  %q", expectedPaths[i], p)
		}
	}
}

func TestRemoveProjectNoConfirmation(t *testing.T) {
	cleanup, err := setupFakeTarget("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectRemove
	err = c.Flags().Parse(true, []string{"-n", "myproj"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader("n\n")}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveProjectValidation(t *testing.T) {
	var c projectRemove
	err := c.Flags().Parse(true, []string{})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr, Stdin: strings.NewReader("n\n")}
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

func TestRemoveProjectNotFound(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	appNames := []string{"myproj-dev", "myproj-qa", "myproj-stage", "myproj-prod"}
	for _, appName := range appNames {
		path := "/apps/" + appName
		server.prepareResponse(preparedResponse{
			method: "DELETE",
			path:   path,
			code:   http.StatusNotFound,
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectRemove
	err = c.Flags().Parse(true, []string{"-yn", "myproj"})
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
	expectedMessage := "project not found"
	if err.Error() != expectedMessage {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMessage, err.Error())
	}
}

func TestRemoveProjectConfigurationIssue(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectRemove
	err = c.Flags().Parse(true, []string{"-n", "superproj"})
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

func TestProjectInfo(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=" + url.QueryEscape("^proj1"),
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/deploys?limit=1&app=proj1-dev",
		code:    http.StatusOK,
		payload: []byte(deployments),
	})
	for _, appName := range []string{"proj1-qa", "proj1-stage"} {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/deploys?limit=1&app=" + appName,
			code:    http.StatusNoContent,
			payload: nil,
		})
	}
	appRespMap := map[string][]byte{
		"proj1-dev":   []byte(appInfo1),
		"proj1-qa":    []byte(appInfo2),
		"proj1-stage": []byte(appInfo3),
		"proj1-prod":  []byte(appInfo4),
	}
	for appName, payload := range appRespMap {
		server.prepareResponse(preparedResponse{
			method:  "GET",
			path:    "/apps/" + appName,
			code:    http.StatusOK,
			payload: payload,
		})
	}
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectInfo
	err = c.Flags().Parse(true, []string{"-n", "proj1"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	table := cmd.Table{Headers: cmd.Row([]string{"Environment", "Address", "Image", "Git hash/tag", "Deploy date", "Units"})}
	expectedOutput := `Project name: proj1
Description: my nice project
Repository: git@example.com:proj1-dev.git
Platform: python
Teams: admin, sysop
Owner: webmaster@example.com
Team owner: admin` + "\n\n"
	rows := []cmd.Row{
		{"dev", "proj1.dev.example.com", "v938", "(git) 40244ff2866eba7e2da6eee8a6fc51464c9f604f", "Mon, 05 Sep 2016 01:24:25 UTC", "1"},
		{"qa", "proj1.qa.example.com", "", "", "", "2"},
		{"stage", "proj1.stage.example.com", "", "", "", "2"},
		{"prod", "proj1.example.com", "", "", "", "5"},
	}
	for _, row := range rows {
		table.AddRow(row)
	}
	expectedOutput += table.String()
	if stdout.String() != expectedOutput {
		t.Errorf("wrong output\nWant:\n%s\nGot:\n%s", expectedOutput, stdout.String())
	}
}

func TestProjectInfoNotFound(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method: "GET",
		path:   "/apps?name=" + url.QueryEscape("^proj1"),
		code:   http.StatusNoContent,
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectInfo
	err = c.Flags().Parse(true, []string{"-n", "proj1"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal(err)
	}
	expectedMsg := "project not found"
	if err.Error() != expectedMsg {
		t.Errorf("wrong error message\nwant %q\ngot  %q", expectedMsg, err.Error())
	}
}

func TestProjectInfoErrorToListApps(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?name=proj1",
		code:    http.StatusInternalServerError,
		payload: []byte("something went wrong"),
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectInfo
	err = c.Flags().Parse(true, []string{"-n", "proj1"})
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
}

func TestProjectInfoConfigIssue(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectInfo
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

func TestProjectInfoMissingName(t *testing.T) {
	var c projectInfo
	err := c.Flags().Parse(true, []string{})
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

func TestProjectEnvInfo(t *testing.T) {
	fakeServer := newFakeServer(t)
	defer fakeServer.stop()
	fakeServer.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps/proj1-prod",
		code:    http.StatusOK,
		payload: []byte(appInfo4),
	})
	fakeServer.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps/proj1-prod/quota",
		code:    http.StatusOK,
		payload: []byte(`{"Limit":10,"InUse":1}`),
	})
	fakeServer.prepareResponse(preparedResponse{
		method: "GET",
		path:   "/services/instances?app=proj1-prod",
		code:   http.StatusNoContent,
	})
	cleanup, err := setupFakeTarget(fakeServer.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	cli := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	var buf bytes.Buffer
	var appInfoCmd client.AppInfo
	err = appInfoCmd.Flags().Parse(true, []string{"-a", "proj1-prod"})
	if err != nil {
		t.Fatal(err)
	}
	err = appInfoCmd.Run(&cmd.Context{Stdout: &buf}, cli)
	if err != nil {
		t.Fatal(err)
	}
	var c projectEnvInfo
	err = c.Flags().Parse(true, []string{"-n", "proj1", "-e", "prod"})
	if err != nil {
		t.Fatal(err)
	}
	err = c.Run(&ctx, cli)
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != buf.String() {
		t.Errorf("Wrong output\nWant:\n%s\nGot:\n%s", &buf, &stdout)
	}
}

func TestProjectEnvInfoMissingName(t *testing.T) {
	var c projectEnvInfo
	err := c.Flags().Parse(true, []string{"-e", "prod"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	cli := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, cli)
	if err == nil {
		t.Fatal("got unexpected <nil> error")
	}
}

func TestProjectEnvInfoMissingEnv(t *testing.T) {
	var c projectEnvInfo
	err := c.Flags().Parse(true, []string{"-n", "proj1"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	cli := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, cli)
	if err == nil {
		t.Fatal("got unexpected <nil> error")
	}
}

func TestProjectList(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps?",
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps",
		code:    http.StatusOK,
		payload: []byte(listOfApps),
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectList
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	expectedOutput := `+---------+--------------+-------------------------+
| Project | Environments | Address                 |
+---------+--------------+-------------------------+
| proj1   | dev          | proj1.dev.example.com   |
|         | qa           | proj1.qa.example.com    |
|         | stage        | proj1.stage.example.com |
|         | prod         | proj1.example.com       |
+---------+--------------+-------------------------+
| proj2   | dev          | proj2.dev.example.com   |
|         | qa           | proj2.qa.example.com    |
|         | stage        | proj2.stage.example.com |
|         | prod         | proj2.example.com       |
+---------+--------------+-------------------------+
| proj3   | dev          | proj3.dev.example.com   |
|         | prod         | proj3.example.com       |
+---------+--------------+-------------------------+
`
	if stdout.String() != expectedOutput {
		t.Errorf("wrong output\nWant:\n%s\nGot:\n%s", expectedOutput, stdout.String())
	}
}

func TestProjectListErrorToListApps(t *testing.T) {
	server := newFakeServer(t)
	defer server.stop()
	server.prepareResponse(preparedResponse{
		method:  "GET",
		path:    "/apps",
		code:    http.StatusInternalServerError,
		payload: []byte("something went wrong"),
	})
	cleanup, err := setupFakeTarget(server.url())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var c projectList
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	client := cmd.NewClient(http.DefaultClient, &ctx, &cmd.Manager{})
	err = c.Run(&ctx, client)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
}

func TestProjectListNoConfiguration(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	os.Setenv("HOME", dir)
	var c projectList
	var stdout, stderr bytes.Buffer
	ctx := cmd.Context{Stdout: &stdout, Stderr: &stderr}
	err = c.Run(&ctx, nil)
	if err == nil {
		t.Fatal("unexpected <nil> error")
	}
}

func setupFakeTarget(target string) (func(), error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	os.Setenv("HOME", dir)
	config := Config{
		Target: target,
		Environments: []Environment{
			{
				Name:      "dev",
				DNSSuffix: "dev.example.com",
			},
			{
				Name:      "qa",
				DNSSuffix: "qa.example.com",
			},
			{
				Name:      "stage",
				DNSSuffix: "stage.example.com",
			},
			{
				Name:      "prod",
				DNSSuffix: "example.com",
			},
		},
	}
	err = config.writeTarget()
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return func() { os.RemoveAll(dir) }, writeConfigFile(&config)
}
