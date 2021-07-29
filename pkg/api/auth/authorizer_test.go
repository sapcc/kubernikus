package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/spec"
)

func TestAuthorizer(t *testing.T) {

	admin := models.Principal{
		ID:      "test",
		Name:    "Test Mc Dougle",
		Domain:  "TestDomain",
		Account: "testaccount",
		Roles:   []string{"kubernetes_admin"},
	}
	user := models.Principal{
		ID:      "test",
		Name:    "Test Mc Dougle",
		Domain:  "TestDomain",
		Account: "testaccount",
		Roles:   []string{"kubernetes_member"},
	}
	document, err := spec.Spec()
	assert.NoError(t, err)
	rules, err := LoadPolicy("../../../etc/policy.json")
	assert.NoError(t, err)
	authorizer, err := NewOsloPolicyAuthorizer(document, rules, nil)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/clusters", nil)
	assert.NoError(t, authorizer.Authorize(req, &admin), "admin can list clusters")
	assert.NoError(t, authorizer.Authorize(req, &user), "user can list clusters")

	req = httptest.NewRequest("POST", "/api/v1/clusters", nil)
	assert.NoError(t, authorizer.Authorize(req, &admin), "admin can create clusters")
	assert.Error(t, authorizer.Authorize(req, &user), "user can not create clusters")
}
