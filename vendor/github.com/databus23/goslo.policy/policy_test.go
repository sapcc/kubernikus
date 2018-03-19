package policy

import (
	"encoding/json"
	"os"
	"testing"
)

func TestRules(t *testing.T) {

	context := Context{
		Roles: []string{"guest", "member"},
		Auth: map[string]string{
			"user_id":    "u-1",
			"project_id": "p-2",
		},
		Request: map[string]string{
			"target.user_id": "u-1",
			"user_id":        "u-2",
			"some_number":    "1",
			"some_bool":      "True",
		},
		Logger: t.Logf,
	}

	testCases := []struct {
		rule   string
		result bool
	}{
		{"", true},
		{"@", true},
		{"!", false},
		{"role:member", true},
		{"not role:member", false},
		{"role:admin", false},
		{"role:admin or role:guest", true},
		{"role:admin and role:guest", false},
		{"user_id:u-1", true},
		{"user_id:u-2", false},
		{"'u-2':%(user_id)s", true},
		{"True:%(some_bool)s", true},
		{"1:%(some_number)s", true},
		{"domain_id:%(does_not_exit)s", false},
		{"not (@ or @)", false},
		{"not @ or @", true},
		{"@ and (! or (not !))", true},
	}

	for _, c := range testCases {
		p, err := NewEnforcer(map[string]string{"test": c.rule})
		if err != nil {
			t.Error(err)
			continue
		}
		if result := p.Enforce("test", context); result != c.result {
			t.Errorf("Rule %q returned %v, expected %v", c.rule, result, c.result)
		}
	}

}

func TestPolicy(t *testing.T) {
	var keystonePolicy map[string]string

	file, err := os.Open("testdata/keystone.v3cloudsample.json")
	if err != nil {
		t.Fatal("Failed to open policy file: ", err)
	}
	if err := json.NewDecoder(file).Decode(&keystonePolicy); err != nil {
		t.Fatal("Failed to decode policy file: ", err)
	}

	serviceContext := Context{
		Roles: []string{"service"},
	}

	adminContext := Context{
		Roles: []string{"admin"},
		Auth: map[string]string{
			"domain_id": "admin_domain_id",
		},
		Logger: t.Logf,
	}
	userContext := Context{
		Roles: []string{"member"},
		Auth: map[string]string{
			"user_id": "u-1",
		},
		Request: map[string]string{
			"user_id": "u-1",
		},
		Logger: t.Logf,
	}

	enforcer, err := NewEnforcer(keystonePolicy)
	if err != nil {
		t.Fatal("Failed to parse policy ", err)
	}
	if !enforcer.Enforce("service_or_admin", serviceContext) {
		t.Error("service_or_admin check should have returned true")
	}
	if enforcer.Enforce("non_existant_rule", serviceContext) {
		t.Error("Non existant rule should not pass")
	}
	if !enforcer.Enforce("cloud_admin", adminContext) {
		t.Error("cloud_admin check should pass")
	}
	if !enforcer.Enforce("service_admin_or_owner", adminContext) {
		t.Error("service_admin_or_owner should pass for admin")
	}
	if !enforcer.Enforce("service_admin_or_owner", userContext) {
		t.Error("service_admin_or_owner should pass for owner")
	}
	userContext.Request["user_id"] = "u-2"
	if enforcer.Enforce("service_admin_or_owner", userContext) {
		t.Error("service_admin_or_owner should pass for non owning user")
	}

}
