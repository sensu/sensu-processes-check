package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
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
	Search string
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
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if len(plugin.Search) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--search is required")
	}
	return sensu.CheckStateOK, nil
}

func executeCheck(event *types.Event) (int, error) {
	found := make(map[string]int)
	searches, err := parseSearches(plugin.Search)
	if err != nil {
		return sensu.CheckStateCritical, err
	}

	myPid := os.Getpid()
	processes, _ := process.Processes()
	for _, process := range processes {
		// Skip myself
		if process.Pid == int32(myPid) {
			continue
		}
		name, _ := process.Name()
		cmdline, _ := process.Cmdline()
		for _, search := range searches {
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

	sev := 0
	for _, search := range searches {
		mySev := 0
		strExpr := fmt.Sprintf("%d %s %d", found[search.SearchString], search.Comparison, search.Number)
		expression, err := govaluate.NewEvaluableExpression(strExpr)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Unable to create expression %s: %v", strExpr, err)
		}
		result, err := expression.Evaluate(nil)
		if err != nil {
			return sensu.CheckStateCritical, fmt.Errorf("Unable to evalute expression %s: %v", strExpr, err)
		}

		if !result.(bool) && sev < search.Severity {
			sev = search.Severity
			mySev = search.Severity
		} else if !result.(bool) {
			mySev = search.Severity
		}
		// Add a --verbose option for this? Or only report failures in the conditional above?
		fmt.Printf("%-8s | %d %s %d (found %s required) evaluated %v for %q\n", mapSeverity(mySev), found[search.SearchString], search.Comparison, search.Number, search.Comparison, result.(bool), search.SearchString)
	}
	fmt.Printf("Status - %s\n", mapSeverity(sev))
	return sev, nil
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
		Number: 1,
		Severity: 2,
		Comparison: ">=",
		SearchString: "",
	}

	_ = json.Unmarshal(data, search)

	*s = Search(*search)
	return nil
}

