package main

import (
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
}

func TestMapSeverity(t *testing.T) {
	assert := assert.New(t)
	ret := mapSeverity(0)
	assert.Equal("OK", ret)
	ret = mapSeverity(1)
	assert.Equal("WARNING", ret)
	ret = mapSeverity(2)
	assert.Equal("CRITICAL", ret)
	for i := 3; i < 10; i++ {
		ret = mapSeverity(i)
		assert.Equal("UNKNOWN", ret)
	}

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
	assert.NoError(err)

	plugin.Search = `[{"full_cmdline": true, "number": 1, "severity": 1, "search_string": "eventmonitor"}, {"number": 1, "severity": 2, "search_string": "ssh-agent"}, {"number": 2, "severity": 1, "search_string": "non-existent"}]`
	_, err = checkArgs(event)
	assert.NoError(err)
}

func TestExecuteCheckWithSearch(t *testing.T) {
	assert := assert.New(t)
	plugin.Verbose = true
	plugin.Search = `[{"full_cmdline": true, "number": 1, "severity": 1, "search_string": "go"}, {"full_cmdline": true, "number": 2, "severity": 1, "search_string": "go"}]`
	_, err := executeCheck(nil)
	assert.NoError(err)
}
func TestExecuteCheckFullProcessList(t *testing.T) {
	assert := assert.New(t)
	plugin.Verbose = true
	plugin.Search = ``
	_, err := executeCheck(nil)
	assert.NoError(err)
}

func TestExecuteCheckFullProcessListMetricsOnly(t *testing.T) {
	assert := assert.New(t)
	plugin.Verbose = false
	plugin.MetricsOnly = true
	plugin.Search = ``
	_, err := executeCheck(nil)
	assert.NoError(err)
}
