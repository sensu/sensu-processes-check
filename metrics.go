package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/shirou/gopsutil/process"
)

func generateMetrics(found map[string][]*process.Process) error {
	totalFamilies := make([]*dto.MetricFamily, 0)
	processesFamily := newMetricFamily("processes", "summary metrics", dto.MetricType_GAUGE)
	totalFamilies = append(totalFamilies, processesFamily)

	statFamilies := make([]*dto.MetricFamily, 0)
	procstatFamily := newMetricFamily("procstat", "per-process metrics", dto.MetricType_GAUGE)
	statFamilies = append(statFamilies, procstatFamily)

	nowMS := time.Now().UnixMilli()
	totalProcesses := int64(0)
	totalThreads := int64(0)
	totalStatusMap := map[string]int64{
		"parked":   0,
		"wait":     0,
		"blocked":  0,
		"zombies":  0,
		"dead":     0,
		"stopped":  0,
		"running":  0,
		"sleeping": 0,
		"idle":     0,
		"unknown":  0,
		"other":    0,
	}
	totalTags := make(map[string]string)
	hostname, err := os.Hostname()
	if err == nil {
		totalTags["host.name"] = hostname
	}
	for searchStr, processes := range found {
		totalProcesses += int64(len(processes))
		for _, p := range processes {
			var fval32 float32
			var fval64 float64
			var ival32 int32
			var ival64 int64

			name, err := p.Name()
			if err != nil {
				continue
			}
			status, err := p.Status()
			if err != nil {
				if len(status) > 0 {
					switch status[0] {
					case 'P':
						totalStatusMap["parked"] += 1
					case 'W':
						totalStatusMap["wait"] += 1
					case 'U', 'D', 'L':
						totalStatusMap["blocked"] += 1
						// Also known as uninterruptible sleep or disk sleep
					case 'Z':
						totalStatusMap["zombies"] += 1
					case 'X':
						totalStatusMap["dead"] += 1
					case 'T':
						totalStatusMap["stopped"] += 1
					case 'R':
						totalStatusMap["running"] += 1
					case 'S':
						totalStatusMap["sleeping"] += 1
					case 'I':
						totalStatusMap["idle"] += 1
					case '?':
						totalStatusMap["unknown"] += 1
					default:
						totalStatusMap["other"] += 1
					}
				}
			}
			metricTags := make(map[string]string)
			//Set the labels for the metric
			hostname, err := os.Hostname()
			if err == nil {
				metricTags["host.name"] = hostname
			}
			metricTags["field"] = "none"
			metricTags["search_string"] = searchStr
			metricTags["process.executable.name"] = name
			metricTags["process.executable.pid"] = fmt.Sprintf("%v", (p.Pid))

			//cpu_usage metric
			metricTags["field"] = "cpu_usage"
			metricTags["units"] = "percent"
			fval64, _ = p.CPUPercent()
			addGaugeMetric(statFamilies, metricTags, float64(fval64), nowMS)
			//memory_usage metric
			metricTags["field"] = "memory_usage"
			metricTags["units"] = "percent"
			fval32, _ = p.MemoryPercent()
			addGaugeMetric(statFamilies, metricTags, float64(fval32), nowMS)
			mem, err := p.MemoryInfo()
			if err == nil {
				metricTags["units"] = "bytes"
				metricTags["field"] = "memory_rss"
				addGaugeMetric(statFamilies, metricTags, float64(mem.RSS), nowMS)
				metricTags["field"] = "memory_vms"
				addGaugeMetric(statFamilies, metricTags, float64(mem.VMS), nowMS)
				metricTags["field"] = "memory_swap"
				addGaugeMetric(statFamilies, metricTags, float64(mem.Swap), nowMS)
				metricTags["field"] = "memory_data"
				addGaugeMetric(statFamilies, metricTags, float64(mem.Data), nowMS)
				metricTags["field"] = "memory_stack"
				addGaugeMetric(statFamilies, metricTags, float64(mem.Stack), nowMS)
				metricTags["field"] = "memory_locked"
				addGaugeMetric(statFamilies, metricTags, float64(mem.Locked), nowMS)
			}
			//created_at metric
			metricTags["field"] = "created_at"
			metricTags["units"] = "nanoseconds"
			ival64, _ = p.CreateTime()
			addGaugeMetric(statFamilies, metricTags, float64(ival64)*1e6, nowMS)
			//num_fds metric
			metricTags["field"] = "num_fds"
			metricTags["units"] = "count"
			ival32, _ = p.NumFDs()
			addGaugeMetric(statFamilies, metricTags, float64(ival32), nowMS)
			//num_threads metric
			metricTags["field"] = "num_threads"
			metricTags["units"] = "count"
			ival32, _ = p.NumThreads()
			totalThreads += int64(ival32)
			addGaugeMetric(statFamilies, metricTags, float64(ival32), nowMS)
			iostats, err := p.IOCounters()
			if err == nil && iostats != nil {
				//read_count metric
				metricTags["field"] = "read_count"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(iostats.ReadCount), nowMS)
				//read_bytes metric
				metricTags["field"] = "read_bytes"
				metricTags["units"] = "bytes"
				addGaugeMetric(statFamilies, metricTags, float64(iostats.ReadBytes), nowMS)
				//write_count metric
				metricTags["field"] = "write_count"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(iostats.WriteCount), nowMS)
				//write_bytes metric
				metricTags["field"] = "write_bytes"
				metricTags["units"] = "bytes"
				addGaugeMetric(statFamilies, metricTags, float64(iostats.WriteBytes), nowMS)
			}
			faults, err := p.PageFaults()
			if err == nil && faults != nil {
				//major_fault metric
				metricTags["field"] = "major_faults"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(faults.MajorFaults), nowMS)
				//minor_fault metric
				metricTags["field"] = "minor_faults"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(faults.MinorFaults), nowMS)
				//child_major_fault metric
				metricTags["field"] = "child_major_faults"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(faults.ChildMajorFaults), nowMS)
				//child_minor_fault metric
				metricTags["field"] = "child_minor_faults"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(faults.ChildMinorFaults), nowMS)
			}
			switches, err := p.NumCtxSwitches()
			if err == nil && switches != nil {
				//involuntary_context_switches
				metricTags["field"] = "involuntary_context_switches"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(switches.Involuntary), nowMS)
				//voluntary_context_switches
				metricTags["field"] = "voluntary_context_switches"
				metricTags["units"] = "count"
				addGaugeMetric(statFamilies, metricTags, float64(switches.Voluntary), nowMS)
			}
			rlims, err := p.RlimitUsage(true)
			if err == nil {
				for _, rlim := range rlims {
					metricTags["units"] = "N/A"
					var name string
					switch rlim.Resource {
					case process.RLIMIT_CPU:
						name = "cpu_time"
						metricTags["units"] = "seconds"
					case process.RLIMIT_DATA:
						name = "memory_data"
						metricTags["units"] = "bytes"
					case process.RLIMIT_STACK:
						name = "memory_stack"
						metricTags["units"] = "bytes"
					case process.RLIMIT_RSS:
						name = "memory_rss"
						metricTags["units"] = "bytes"
					case process.RLIMIT_NOFILE:
						name = "num_fds"
						metricTags["units"] = "count"
					case process.RLIMIT_MEMLOCK:
						name = "memory_locked"
						metricTags["units"] = "bytes"
					case process.RLIMIT_AS:
						name = "memory_vms"
						metricTags["units"] = "bytes"
					case process.RLIMIT_LOCKS:
						name = "file_locks"
						metricTags["units"] = "count"
					case process.RLIMIT_SIGPENDING:
						name = "signals_pending"
						metricTags["units"] = "count"
					case process.RLIMIT_NICE:
						name = "nice_priority"
					case process.RLIMIT_RTPRIO:
						name = "realtime_priority"
					default:
						continue
					}
					metricTags["field"] = "rlimit_" + name + "_soft"
					addGaugeMetric(statFamilies, metricTags, float64(rlim.Soft), nowMS)
					metricTags["field"] = "rlimit_" + name + "_hard"
					addGaugeMetric(statFamilies, metricTags, float64(rlim.Hard), nowMS)

					switch rlim.Resource {
					case process.RLIMIT_CPU,
						process.RLIMIT_LOCKS,
						process.RLIMIT_SIGPENDING,
						process.RLIMIT_NICE,
						process.RLIMIT_RTPRIO:
						metricTags["field"] = name
						addGaugeMetric(statFamilies, metricTags, float64(rlim.Used), nowMS)
					}

				}
			}

		}
	}
	totalTags["field"] = "total"
	totalTags["units"] = "count"
	addGaugeMetric(totalFamilies, totalTags, float64(totalProcesses), nowMS)
	totalTags["field"] = "total_threads"
	totalTags["units"] = "count"
	addGaugeMetric(totalFamilies, totalTags, float64(totalThreads), nowMS)
	for status, count := range totalStatusMap {
		totalTags["field"] = status
		totalTags["units"] = "count"
		addGaugeMetric(totalFamilies, totalTags, float64(count), nowMS)
	}
	statFamilies = append(statFamilies, totalFamilies...)
	var buf bytes.Buffer
	for _, family := range statFamilies {
		if len(family.Metric) > 0 {
			buf.Reset()
			encoder := expfmt.NewEncoder(&buf, expfmt.FmtText)
			err = encoder.Encode(family)
			if err != nil {
				return err
			}

			fmt.Print(buf.String())
		}
	}
	return nil

}
func newMetricFamily(name, help string, metricType dto.MetricType) *dto.MetricFamily {
	return &dto.MetricFamily{
		Name:   &name,
		Help:   &help,
		Type:   &metricType,
		Metric: []*dto.Metric{},
	}
}

func addGaugeMetric(families []*dto.MetricFamily, tags map[string]string, value float64, timestampMS int64) {
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
	for _, family := range families {
		family.Metric = append(family.Metric, gauge)
	}
}
