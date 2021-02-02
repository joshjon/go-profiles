package auth

import (
	"github.com/joshjon/go-profiles/internal/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAuthorizerAccept(t *testing.T) {
	var testCases = []struct{ action string }{
		{action: "create"},
		{action: "read"},
		{action: "update"},
		{action: "delete"},
	}

	auth := New(config.ACLModelFile, config.ACLPolicyFile)
	subject := "root"
	object := "*"

	for _, tc := range testCases {
		assert.NoError(t, auth.Authorize(subject, object, tc.action), "scenario: root * "+tc.action)
	}
}

func TestAuthorizerDeny(t *testing.T) {
	var testCases = []struct{ scenario, subject, object, action string }{
		{scenario: "bad subject", subject: "foo", object: "*", action: "create"},
		{scenario: "bad object", subject: "root", object: "foo", action: "create"},
		{scenario: "bad action", subject: "root", object: "*", action: "foo"},
	}
	auth := New(config.ACLModelFile, config.ACLPolicyFile)
	for _, tc := range testCases {
		assert.Error(t, auth.Authorize(tc.subject, tc.object, tc.action), "scenario: "+tc.scenario)
	}
}
