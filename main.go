package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Knetic/govaluate"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
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
	SumoLogicCompat  bool
	MetricsOnly      bool
	Verbose          bool
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-processes-check",
			Short:    "Sensu Processes Check",
			Keyspace: "sensu.io/plugins/sensu-processes-check/config",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "search",
			Env:       "PROCESSES_CHECK_SEARCH",
			Argument:  "search",
			Shorthand: "s",
			Default:   "",
			Usage:     `An array of JSON search criteria, fields are "search_string", "severity", "number", "comparison", and "full_cmdline"`,
			Value:     &plugin.Search,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "suppress-ok-output",
			Env:       "PROCESSES_CHECK_SUPPRESS_OK_OUTPUT",
			Argument:  "suppress-ok-output",
			Shorthand: "S",
			Default:   false,
			Usage:     "Aside from overal status, only output failures",
			Value:     &plugin.SuppressOKOutput,
		},
		&sensu.PluginConfigOption[bool]{
			Path:      "metrics-only",
			Env:       "PROCESSES_CHECK_METRICS_ONLY",
			Argument:  "metrics-only",
			Shorthand: "",
			Default:   false,
			Usage:     "Do not alert based on search configuration",
			Value:     &plugin.MetricsOnly,
		},
		/*
			&sensu.PluginConfigOption[bool]{
					Path:      "sumologic-compat",
					Env:       "PROCESSES_CHECK_SUMOLOGIC_COMPAT",
					Argument:  "sumologic-compat",
					Shorthand: "",
					Default:   false,
					Usage:     "Add Sumo Logic compatible \"procstat\" metrics family",
					Value:     &plugin.SumoLogicCompat,
				},
		*/
		&sensu.PluginConfigOption[bool]{
			Path:      "verbose",
			Env:       "PROCESSES_CHECK_SUMOLOGIC_VERBOSE",
			Argument:  "verbose",
			Shorthand: "v",
			Default:   false,
			Usage:     "Verbose output",
			Value:     &plugin.Verbose,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	found := make(map[string][]*process.Process)
	searches, err := parseSearches(plugin.Search)
	if err != nil {
		return sensu.CheckStateCritical, err
	}
	myPid := os.Getpid()
	processes, _ := process.Processes()

	seenPids := make(map[int32]bool)
	for _, p := range processes {
		name, _ := p.Name()
		cmdline, _ := p.Cmdline()
		// Skip myself
		if p.Pid == int32(myPid) {
			if plugin.Verbose {
				fmt.Printf("# Skipping this process: %v %v\n", name, cmdline)
			}
			continue
		}
		// Skip if this Pid has already been tabulated
		if _, ok := seenPids[p.Pid]; ok {
			if plugin.Verbose {
				fmt.Printf("# Skipping duplicate process: %v %v\n", name, cmdline)
			}
			continue
		} else {
			seenPids[p.Pid] = true
		}
		if len(searches) == 0 {
			// if not search configuration is provided, construct metrics for all processes as empty SearchString
			found[""] = append(found[""], p)
		} else {
			for _, search := range searches {
				// skip empty search string, should this be tunable?
				if len(search.SearchString) == 0 {
					if plugin.Verbose {
						fmt.Printf("# Found empty search: %+v\n", search)
					}
					continue
				}
				if !search.FullCmdLine && name == search.SearchString {
					found[search.SearchString] = append(found[search.SearchString], p)
					// break out of search loop if process matches : do not let process match multiple searches
					break
				} else if search.FullCmdLine {
					if strings.Contains(cmdline, search.SearchString) {
						found[search.SearchString] = append(found[search.SearchString], p)
						// break out of search if process matches: do not let process match multiple searches
						break
					}
				}
			}
		}

	}

	// Construct alert severity
	overallSeverity := 0
	if !plugin.MetricsOnly {

		for _, search := range searches {
			// skip empty search string, should this be tunable?
			if len(search.SearchString) == 0 {
				continue
			}
			thisSeverity := 0
			strExpr := fmt.Sprintf("%d %s %d", len(found[search.SearchString]), search.Comparison, search.Number)
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
				fmt.Printf("# %-8s | %d %s %d (found %s required) evaluated %v for %q\n", mapSeverity(thisSeverity), len(found[search.SearchString]), search.Comparison, search.Number, search.Comparison, result.(bool), search.SearchString)
			}

		}
		fmt.Printf("# Status - %s\n\n", mapSeverity(overallSeverity))
	}
	// Construct metrics
	err = generateMetrics(found)
	if err != nil {
		return sensu.CheckStateCritical, nil
	}
	return overallSeverity, nil
}

func parseSearches(searchJSON string) ([]Search, error) {
	searches := []Search{}
	if len(searchJSON) > 0 {
		err := json.Unmarshal([]byte(searchJSON), &searches)
		if err != nil {
			return []Search{}, err
		}
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
