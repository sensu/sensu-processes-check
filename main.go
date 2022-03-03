package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
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
	SumoLogicCompat  bool
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
			Env:       "PROCESSES_CHECK_SUPPRESS_OK_OUTPUT",
			Argument:  "suppress-ok-output",
			Shorthand: "S",
			Default:   false,
			Usage:     "Aside from overal status, only output failures",
			Value:     &plugin.SuppressOKOutput,
		},
		{
			Path:      "sumologic-compat",
			Env:       "PROCESSES_CHECK_SUMOLOGIC_COMPAT",
			Argument:  "sumologic-compat",
			Shorthand: "",
			Default:   false,
			Usage:     "Add Sumo Logic compatible \"procstat\" metrics family",
			Value:     &plugin.SumoLogicCompat,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	//if len(plugin.Search) == 0 {
	//	return sensu.CheckStateUnknown, fmt.Errorf("--search is required")
	//}
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
	for _, p := range processes {
		// Skip myself
		if p.Pid == int32(myPid) {
			continue
		}
		name, _ := p.Name()
		cmdline, _ := p.Cmdline()
		if len(searches) == 0 {
			found[""] = append(found[""], p)
			// if not search configuration is provided, construct metrics for all processes
		} else {
			for _, search := range searches {
				// skip duplicate search string to prevent duplicate counts
				if len(found[search.SearchString]) > 0 {
					continue
				}
				// skip empty search string, should this be tunable?
				if len(search.SearchString) == 0 {
					continue
				}
				if !search.FullCmdLine && name == search.SearchString {
					found[search.SearchString] = append(found[search.SearchString], p)
					break
				} else if search.FullCmdLine {
					if strings.Contains(cmdline, search.SearchString) {
						found[search.SearchString] = append(found[search.SearchString], p)
						break
					}
				}
			}
		}

	}

	// Construct alert severity
	overallSeverity := 0
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
	fmt.Printf("# Status - %s\n", mapSeverity(overallSeverity))

	// Construct metrics
	nowMS := time.Now().UnixMilli()
	families := make([]*dto.MetricFamily, 0)
	//var processesFamily *dto.MetricFamily
	//processesFamily = newMetricFamily("processes", "SumoLogic Compatibility", dto.MetricType_GAUGE)
	//families = append(families, processesFamily)
	var procstatFamily *dto.MetricFamily
	procstatFamily = newMetricFamily("procstat", "SumoLogic Compatibility", dto.MetricType_GAUGE)
	families = append(families, procstatFamily)
	for searchStr, processes := range found {
		//var totalMetrics []string
		//totalTags := make(map[string][]*process.Process)
		for _, p := range processes {
			name, _ := p.Name()
			labels := make([]*dto.LabelPair, 0)
			procstatTags := make(map[string]string)
			//Set the labels for the metric
			procstatTags["field"] = "none"
			procstatTags["search_string"] = searchStr
			procstatTags["process.executable.name"] = name

			//CPU usage metric
			procstatTags["field"] = "cpu_usage"
			value, _ := p.CPUPercent()
			keys := make([]string, 0, len(procstatTags))
			for k := range procstatTags {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				name := fmt.Sprintf("%v", key)
				val := fmt.Sprintf("%v", procstatTags[key])
				labels = append(labels, &dto.LabelPair{Name: &name, Value: &val})
			}
			gauge := &dto.Metric{
				Label: labels,
				Gauge: &dto.Gauge{
					Value: &value,
				},
				TimestampMs: &nowMS,
			}
			procstatFamily.Metric = append(procstatFamily.Metric, gauge)

		}
	}

	var buf bytes.Buffer
	for _, family := range families {
		buf.Reset()
		encoder := expfmt.NewEncoder(&buf, expfmt.FmtText)
		err = encoder.Encode(family)
		if err != nil {
			return sensu.CheckStateCritical, err
		}

		fmt.Print(buf.String())
	}

	return overallSeverity, nil
}

func newMetricFamily(name, help string, metricType dto.MetricType) *dto.MetricFamily {
	return &dto.MetricFamily{
		Name:   &name,
		Help:   &help,
		Type:   &metricType,
		Metric: []*dto.Metric{},
	}
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
