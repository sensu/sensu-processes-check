package main

import (
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
}

func TestParseSearches(t *testing.T) {
	assert := assert.New(t)
	searches := `[{"full_cmdline": true, "number": 1, "severity": 1, "search_string": "eventmonitor"}, {"number": 1, "severity": 2, "search_string": "ssh-agent"}, {"number": 2, "severity": 1, "search_string": "non-existent"}]`
	s, err := parseSearches(searches)
	assert.NoError(err)
	assert.Equal(3, len(s))
}

func TestCheckArgs(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")
	_, err := checkArgs(event)
	assert.Error(err)

	plugin.Search = `[{"full_cmdline": true, "number": 1, "severity": 1, "search_string": "eventmonitor"}, {"number": 1, "severity": 2, "search_string": "ssh-agent"}, {"number": 2, "severity": 1, "search_string": "non-existent"}]`
	_, err = checkArgs(event)
	assert.NoError(err)
}
