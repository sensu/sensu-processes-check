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
	var processesFamily *dto.MetricFamily
	processesFamily = newMetricFamily("processes", "SumoLogic Compatibility", dto.MetricType_GAUGE)
	families = append(families, processesFamily)
	var procstatFamily *dto.MetricFamily
	procstatFamily = newMetricFamily("procstat", "SumoLogic Compatibility", dto.MetricType_GAUGE)
	families = append(families, procstatFamily)
	totalProcesses := int64(0)
	totalTags := make(map[string]string)
	hostname, err := os.Hostname()
	if err == nil {
		totalTags["host.name"] = hostname
	}
	for searchStr, processes := range found {
		totalProcesses += int64(len(processes))
		//var totalMetrics []string
		//totalTags := make(map[string][]*process.Process)
		for _, p := range processes {
			var fval32 float32
			var fval64 float64
			var ival32 int32
			var ival64 int64

			name, err := p.Name()
			if err != nil {
				continue
			}
			procstatTags := make(map[string]string)
			//Set the labels for the metric
			hostname, err := os.Hostname()
			if err == nil {
				procstatTags["host.name"] = hostname
			}
			procstatTags["field"] = "none"
			procstatTags["search_string"] = searchStr
			procstatTags["process.executable.name"] = name

			//cpu_usage metric
			procstatTags["field"] = "cpu_usage"
			procstatTags["units"] = "percent"
			fval64, _ = p.CPUPercent()
			gauge := newGaugeMetric(procstatTags, float64(fval64), nowMS)
			procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			//memory_usage metric
			procstatTags["field"] = "memory_usage"
			procstatTags["units"] = "percent"
			fval32, _ = p.MemoryPercent()
			gauge = newGaugeMetric(procstatTags, float64(fval32), nowMS)
			procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			mem, err := p.MemoryInfo()
			if err == nil {
				procstatTags["units"] = "bytes"
				procstatTags["field"] = "memory_rss"
				gauge = newGaugeMetric(procstatTags, float64(mem.RSS), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				procstatTags["field"] = "memory_vms"
				gauge = newGaugeMetric(procstatTags, float64(mem.VMS), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				procstatTags["field"] = "memory_swap"
				gauge = newGaugeMetric(procstatTags, float64(mem.Swap), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				procstatTags["field"] = "memory_data"
				gauge = newGaugeMetric(procstatTags, float64(mem.Data), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				procstatTags["field"] = "memory_stack"
				gauge = newGaugeMetric(procstatTags, float64(mem.Stack), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				procstatTags["field"] = "memory_locked"
				gauge = newGaugeMetric(procstatTags, float64(mem.Locked), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			}
			//created_at metric
			procstatTags["field"] = "created_at"
			procstatTags["units"] = "nanoseconds"
			ival64, _ = p.CreateTime()
			gauge = newGaugeMetric(procstatTags, float64(ival64)*1e6, nowMS)
			procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			//num_fds metric
			procstatTags["field"] = "num_fds"
			procstatTags["units"] = "count"
			ival32, _ = p.NumFDs()
			gauge = newGaugeMetric(procstatTags, float64(ival32)*1e6, nowMS)
			procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			//num_threads metric
			procstatTags["field"] = "num_threads"
			procstatTags["units"] = "count"
			ival32, _ = p.NumThreads()
			gauge = newGaugeMetric(procstatTags, float64(ival32)*1e6, nowMS)
			procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			iostats, err := p.IOCounters()
			if err == nil && iostats != nil {
				//read_count metric
				procstatTags["field"] = "read_count"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(iostats.ReadCount), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				//read_bytes metric
				procstatTags["field"] = "read_bytes"
				procstatTags["units"] = "bytes"
				gauge = newGaugeMetric(procstatTags, float64(iostats.ReadBytes), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				//write_count metric
				procstatTags["field"] = "write_count"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(iostats.WriteCount), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				//write_bytes metric
				procstatTags["field"] = "write_bytes"
				procstatTags["units"] = "bytes"
				gauge = newGaugeMetric(procstatTags, float64(iostats.WriteBytes), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			}
			faults, err := p.PageFaults()
			if err == nil && faults != nil {
				//major_fault metric
				procstatTags["field"] = "major_faults"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(faults.MajorFaults), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				//minor_fault metric
				procstatTags["field"] = "minor_faults"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(faults.MinorFaults), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				//child_major_fault metric
				procstatTags["field"] = "child_major_faults"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(faults.ChildMajorFaults), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				//child_minor_fault metric
				procstatTags["field"] = "child_minor_faults"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(faults.ChildMinorFaults), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			}
			switches, err := p.NumCtxSwitches()
			if err == nil && switches != nil {
				//involuntary_context_switches
				procstatTags["field"] = "involuntary_context_switches"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(switches.Involuntary), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
				//voluntary_context_switches
				procstatTags["field"] = "voluntary_context_switches"
				procstatTags["units"] = "count"
				gauge = newGaugeMetric(procstatTags, float64(switches.Voluntary), nowMS)
				procstatFamily.Metric = append(procstatFamily.Metric, gauge)
			}
			rlims, err := p.RlimitUsage(true)
			if err == nil {
				for _, rlim := range rlims {
					procstatTags["units"] = "N/A"
					var name string
					switch rlim.Resource {
					case process.RLIMIT_CPU:
						name = "cpu_time"
					case process.RLIMIT_DATA:
						name = "memory_data"
						procstatTags["units"] = "bytes"
					case process.RLIMIT_STACK:
						name = "memory_stack"
						procstatTags["units"] = "bytes"
					case process.RLIMIT_RSS:
						name = "memory_rss"
						procstatTags["units"] = "bytes"
					case process.RLIMIT_NOFILE:
						name = "num_fds"
						procstatTags["units"] = "count"
					case process.RLIMIT_MEMLOCK:
						name = "memory_locked"
						procstatTags["units"] = "bytes"
					case process.RLIMIT_AS:
						name = "memory_vms"
						procstatTags["units"] = "bytes"
					case process.RLIMIT_LOCKS:
						name = "file_locks"
						procstatTags["units"] = "count"
					case process.RLIMIT_SIGPENDING:
						name = "signals_pending"
						procstatTags["units"] = "count"
					case process.RLIMIT_NICE:
						name = "nice_priority"
					case process.RLIMIT_RTPRIO:
						name = "realtime_priority"
					default:
						continue
					}
					procstatTags["field"] = "rlimit_" + name + "_soft"
					gauge = newGaugeMetric(procstatTags, float64(rlim.Soft), nowMS)
					procstatFamily.Metric = append(procstatFamily.Metric, gauge)
					procstatTags["field"] = "rlimit_" + name + "_hard"
					gauge = newGaugeMetric(procstatTags, float64(rlim.Hard), nowMS)
					procstatFamily.Metric = append(procstatFamily.Metric, gauge)

					switch rlim.Resource {
					case process.RLIMIT_CPU,
						process.RLIMIT_LOCKS,
						process.RLIMIT_SIGPENDING,
						process.RLIMIT_NICE,
						process.RLIMIT_RTPRIO:
						procstatTags["field"] = name
						gauge = newGaugeMetric(procstatTags, float64(rlim.Used), nowMS)
						procstatFamily.Metric = append(procstatFamily.Metric, gauge)
					}

				}
			}
		}
	}
	totalTags["field"] = "total"
	totalTags["units"] = "count"
	gauge := newGaugeMetric(totalTags, float64(totalProcesses), nowMS)
	processesFamily.Metric = append(processesFamily.Metric, gauge)

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
func newGaugeMetric(tags map[string]string, value float64, timestampMS int64) *dto.Metric {
	labels := make([]*dto.LabelPair, 0)
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		name := fmt.Sprintf("%v", key)
		val := fmt.Sprintf("%v", tags[key])
		labels = append(labels, &dto.LabelPair{Name: &name, Value: &val})
	}
	gauge := &dto.Metric{
		Label: labels,
		Gauge: &dto.Gauge{
			Value: &value,
		},
		TimestampMs: &timestampMS,
	}
	return gauge
}
func newCounterMetric(tags map[string]string, value float64, timestampMS int64) *dto.Metric {
	labels := make([]*dto.LabelPair, 0)
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		name := fmt.Sprintf("%v", key)
		val := fmt.Sprintf("%v", tags[key])
		labels = append(labels, &dto.LabelPair{Name: &name, Value: &val})
	}
	counter := &dto.Metric{
		Label: labels,
		Counter: &dto.Counter{
			Value: &value,
		},
		TimestampMs: &timestampMS,
	}
	return counter
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
