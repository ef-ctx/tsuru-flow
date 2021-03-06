// Copyright 2016 EF CTX. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"testing"

	"github.com/tsuru/tsuru-client/tsuru/admin"
	"github.com/tsuru/tsuru-client/tsuru/client"
	"github.com/tsuru/tsuru/cmd"
)

func TestBaseCommandsAreRegistered(t *testing.T) {
	baseManager := cmd.BuildBaseManager("tranor", "", "", nil)
	manager := buildManager("tranor")
	for name, expectedCommand := range baseManager.Commands {
		var skip bool
		for _, c := range baseCommandsToRemove {
			if name == c {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		gotCommand, ok := manager.Commands[name]
		if !ok {
			t.Errorf("Command %q not found", name)
		}
		if reflect.TypeOf(gotCommand) != reflect.TypeOf(expectedCommand) {
			t.Errorf("Command %q: want %#v. Got %#v", name, expectedCommand, gotCommand)
		}
	}
}

func TestDefaultTargetCommandsArentRegistered(t *testing.T) {
	manager := buildManager("tranor")
	cmds := []string{"target-add", "target-list", "target-remove"}
	for _, cmd := range cmds {
		if _, ok := manager.Commands[cmd]; ok {
			t.Errorf("command %q should not be registered", cmd)
		}
	}
}

func TestEnvListIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["env-list"]
	if !ok {
		t.Error("command env-list not found")
	}
	if _, ok := gotCommand.(envList); !ok {
		t.Errorf("command %#v is not of type envList{}", gotCommand)
	}
}

func TestPlatformListIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["platform-list"]
	if !ok {
		t.Error("command platform-list not found")
	}
	if _, ok := gotCommand.(admin.PlatformList); !ok {
		t.Errorf("command %#v is not of type PlatformList{}", gotCommand)
	}
}

func TestTeamListIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["team-list"]
	if !ok {
		t.Error("command team-list not found")
	}
	if _, ok := gotCommand.(*client.TeamList); !ok {
		t.Errorf("command %#v is not of type TeamList{}", gotCommand)
	}
}

func TestTeamCreateIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["team-create"]
	if !ok {
		t.Error("command team-create not found")
	}
	if _, ok := gotCommand.(*client.TeamCreate); !ok {
		t.Errorf("command %#v is not of type TeamCreate{}", gotCommand)
	}
}

func TestTeamRemoveIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["team-remove"]
	if !ok {
		t.Error("command team-remove not found")
	}
	if _, ok := gotCommand.(*client.TeamRemove); !ok {
		t.Errorf("command %#v is not of type TeamRemove{}", gotCommand)
	}
}

func TestPlanListIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["plan-list"]
	if !ok {
		t.Error("command plan-list not found")
	}
	if _, ok := gotCommand.(*client.PlanList); !ok {
		t.Errorf("command %#v is not of type PlanList{}", gotCommand)
	}
}

func TestProjectCreateIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-create"]
	if !ok {
		t.Error("command project-create not found")
	}
	if _, ok := gotCommand.(*projectCreate); !ok {
		t.Errorf("command %#v is not of type projectCreate{}", gotCommand)
	}
}

func TestProjectUpdateIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-update"]
	if !ok {
		t.Error("command project-update not found")
	}
	if _, ok := gotCommand.(*projectUpdate); !ok {
		t.Errorf("command %#v is not of type projectUpdate{}", gotCommand)
	}
}

func TestProjectRemoveIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-remove"]
	if !ok {
		t.Error("command project-remove not found")
	}
	if _, ok := gotCommand.(*projectRemove); !ok {
		t.Errorf("command %#v is not of type projectRemove{}", gotCommand)
	}
}

func TestProjectInfoIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-info"]
	if !ok {
		t.Error("command project-info not found")
	}
	if _, ok := gotCommand.(*projectInfo); !ok {
		t.Errorf("command %#v is not of type projectInfo{}", gotCommand)
	}
}

func TestProjectEnvInfoIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-env-info"]
	if !ok {
		t.Error("command project-env-info not found")
	}
	if _, ok := gotCommand.(*projectEnvInfo); !ok {
		t.Errorf("command %#v is not of type projectEnvInfo{}", gotCommand)
	}
}

func TestProjectListIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-list"]
	if !ok {
		t.Error("command project-list not found")
	}
	if _, ok := gotCommand.(*projectList); !ok {
		t.Errorf("command %#v is not of type projectList{}", gotCommand)
	}
}

func TestProjectEnvVarGetIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["envvar-get"]
	if !ok {
		t.Error("command envvar-get not found")
	}
	if _, ok := gotCommand.(*projectEnvVarGet); !ok {
		t.Errorf("command %#v is not of type projectConfigGet{}", gotCommand)
	}
}

func TestProjectEnvVarSetIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["envvar-set"]
	if !ok {
		t.Error("command envvar-set not found")
	}
	if _, ok := gotCommand.(*projectEnvVarSet); !ok {
		t.Errorf("command %#v is not of type projectConfigSet{}", gotCommand)
	}
}

func TestProjectEnvVarUnsetIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["envvar-unset"]
	if !ok {
		t.Error("command envvar-unset not found")
	}
	if _, ok := gotCommand.(*projectEnvVarUnset); !ok {
		t.Errorf("command %#v is not of type projectConfigUnset{}", gotCommand)
	}
}

func TestProjectDeployIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-deploy"]
	if !ok {
		t.Error("command deploy not found")
	}
	if _, ok := gotCommand.(*projectDeploy); !ok {
		t.Errorf("command %#v is not of type projectDeploy{}", gotCommand)
	}
}

func TestProjectDeployListIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-deploy-list"]
	if !ok {
		t.Error("command deploy-list not found")
	}
	if _, ok := gotCommand.(*projectDeployList); !ok {
		t.Errorf("command %#v is not of type projectDeployList{}", gotCommand)
	}
}

func TestProjectLogIsRegistered(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["project-log"]
	if !ok {
		t.Error("command project-log not found")
	}
	if _, ok := gotCommand.(*projectLog); !ok {
		t.Errorf("command %#v is not of type projectLog{}", gotCommand)
	}
}

func TestBuiltinTargetSetIsOverwritten(t *testing.T) {
	manager := buildManager("tranor")
	gotCommand, ok := manager.Commands["target-set"]
	if !ok {
		t.Error("command target-set not found")
	}
	if _, ok := gotCommand.(targetSet); !ok {
		t.Errorf("command %#v is not of type targetSet{}", gotCommand)
	}
}
