package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/shirou/gopsutil/process"
)

type Search struct {
	// Include a bool for case insensitive searches?
	FullCmdLine  bool   `json:"full_cmdline"`
	Number       int    `json:"number"`
	Severity     int    `json:"severity"`
	Comparison   string `json:"comparison"`
	SearchString string `json:"search_string"`
}

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	Search           string
	SuppressOKOutput bool
	Zombie           bool
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-processes-check",
			Short:    "Sensu Processes Check",
			Keyspace: "sensu.io/plugins/sensu-processes-check/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		{
			Path:      "search",
			Env:       "",
			Argument:  "search",
			Shorthand: "s",
			Default:   "",
			Usage:     `An array of JSON search criteria, fields are "search_string", "severity", "number", "comparison", and "full_cmdline"`,
			Value:     &plugin.Search,
		},
		{
			Path:      "suppress-ok-output",
			Env:       "",
			Argument:  "suppress-ok-output",
			Shorthand: "S",
			Default:   false,
			Usage:     "Aside from overal status, only output failures",
			Value:     &plugin.SuppressOKOutput,
		},
		{
			Path:      "zombie",
			Env:       "",
			Argument:  "zombie",
			Shorthand: "z",
			Default:   false,
			Usage:     "Check for zombie processes",
			Value:     &plugin.Zombie,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	if len(plugin.Search) == 0 {
		return sensu.CheckStateUnknown, fmt.Errorf("--search is required")
	}
	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	found := make(map[string]int)
	searches, err := parseSearches(plugin.Search)
	if err != nil {
		return sensu.CheckStateCritical, err
	}

	myPid := os.Getpid()
	processes, _ := process.Processes()
	for _, proc := range processes {
		// Skip myself
		if proc.Pid == int32(myPid) {
			continue
		}

		// Check for zombie processes if --zombie or -z flag is set
		if plugin.Zombie {
			status, _ := proc.Status()
			if status == "Z" { // "Z" status is for Zombie processes
				fmt.Printf("Zombie process found with PID: %d\n", proc.Pid)
				return sensu.CheckStateCritical, nil
			}
		}

		name, _ := proc.Name()
		cmdline, _ := proc.Cmdline()
		for _, search := range searches {
			// skip empty search string, should this be tunable?
			if len(search.SearchString) == 0 {
				continue
			}
			if !search.FullCmdLine && name == search.SearchString {
				found[search.SearchString]++
				break
			} else if search.FullCmdLine {
				if strings.Contains(cmdline, search.SearchString) {
					found[search.SearchString]++
					break
				}
			}
		}
	}

	overallSeverity := 0
	for _, search := range searches {
		// skip empty search string, should this be tunable?
		if len(search.SearchString) == 0 {
			continue
		}
		thisSeverity := 0
		strExpr := fmt.Sprintf("%d %s %d", found[search.SearchString], search.Comparison, search.Number)
		expression, err := govaluate.NewEvaluableExpression(strExpr)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Unable to create expression %s: %v", strExpr, err)
		}
		result, err := expression.Evaluate(nil)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Unable to evalute expression %s: %v", strExpr, err)
		}

		if !result.(bool) && overallSeverity < search.Severity {
			overallSeverity = search.Severity
			thisSeverity = search.Severity
		} else if !result.(bool) {
			thisSeverity = search.Severity
		}

		if (!plugin.SuppressOKOutput && thisSeverity == 0) || thisSeverity > 0 {
			fmt.Printf("%-8s | %d %s %d (found %s required) evaluated %v for %q\n", mapSeverity(thisSeverity), found[search.SearchString], search.Comparison, search.Number, search.Comparison, result.(bool), search.SearchString)
		}

	}

	fmt.Printf("Status - %s\n", mapSeverity(overallSeverity))
	return overallSeverity, nil
}

func parseSearches(searchJSON string) ([]Search, error) {
	searches := []Search{}
	err := json.Unmarshal([]byte(searchJSON), &searches)
	if err != nil {
		return []Search{}, err
	}

	return searches, nil
}

func mapSeverity(sev int) string {
	switch sev {
	case sensu.CheckStateOK:
		return "OK"
	case sensu.CheckStateWarning:
		return "WARNING"
	case sensu.CheckStateCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Set the defaults for searching
func (s *Search) UnmarshalJSON(data []byte) error {
	type searchAlias Search
	search := &searchAlias{
		Number:       1,
		Severity:     2,
		Comparison:   ">=",
		SearchString: "",
	}

	_ = json.Unmarshal(data, search)

	*s = Search(*search)
	return nil
}
