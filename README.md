[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-processes-check)
![Go Test](https://github.com/sensu/sensu-processes-check/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-processes-check/workflows/goreleaser/badge.svg)

# Sensu Processes Check

## Table of Contents
- [Overview](#overview)
- [Output Metrics](#output-metrics)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Usage examples](#usage-examples)
  - [Help output](#help-output)
  - [Search string details](#search-string-details)
  - [Exit severity](#exit-severity)
  - [Annotations](#annotations)
- [Installation from source](#installation-from-source)
- [Contributing](#contributing)

## Overview

The Sensu Processes Check is a [Sensu Check][1] that provides metrics for
processes found in the host process table.

It can optionally restrict the list of processes considered,  using a [search configuration](#search-string-details) specifying multiple
strings to match. The search configuration can also be used to set alert conditions based on the 
number of matching processes on a per-string basis.

### Output Metrics
Metrics output conforms to the Prometheus exposition standard.

### procstat
The `procstat` metric family provides individual per-process metrics for each process matching the configured search criteria.
Each different per-process metric in this metric family is distinguished by the value of the label named `field`.
Each metric is also labeled with `process.executable.name` and `process.executable.pid`  

| Field                          | Units       | Description                         |
|--------------------------------|-------------|-------------------------------------|
| cpu_usage                      | percent     | percent of CPU time used by process
| cpu_time                       | seconds     | total amount of time process has used
| created_at                     | nanoseconds | process creation time since Unix epoch
| memory_usage                   | percent     | percent of memory used by process 
| memory_rss                     | bytes       | process resident set (the number of virtual pages resident in RAM 
| memory_vms                     | bytes       | size of the process's virtual memory (address space)
| memory_swap                    | bytes       | size of the process's swap 
| memory_data                    | bytes       | size of the process's data segment (initialized data, uninitialized data, and heap
| memory_stack                   | bytes       | size of the process stack
| memory_locked                  | bytes       | size of memory locked into RAM
| num_fds                        | count       | number of file descriptors opened by process
| file_locks                     | count       | number of file locks held by process
| num_threads                    | count       | number of process threads
| read_count                     | count       | number or read operations performed
| read_bytes                     | bytes       | bytes read
| write_count                    | count       | number of write operations performed
| writebytes                     | bytes       | bytes written
| major_faults                   | count       | The number of major faults the process has made which have required loading a memory page from disk.
| minor_faults                   | count       | The number of minor faults the process has made which have not required loading a memory page from disk.
| child_major_faults             | count       | The number of major faults that the process's waited-for children have made.
| child_minor_faults             | count       | The number of minor faults that the process's waited-for children have made.
| signals_pending                | count       | number of currently queued signals
| nice_priority                  | N/A         | nice priority
| realtime_priority              | N/A         | realtime priority
| voluntary_context_switches     | count       | number of voluntary context switches
| involuntary_context_switches   | count       | number of involuntary context switches
| rlimit_cpu_time_soft           | seconds     | soft maximum limit for cpu_time
| rlimit_cpu_time_hard           | seconds     | hard maximum limit for cpu_time
| rlimit_core_size_soft          | bytes       | ...
| rlimit_core_size_hard          | bytes       | ... 
| rlimit_memory_data_soft        | bytes       | ...
| rlimit_memory_data_hard        | bytes       | ...
| rlimit_memory_stack_soft       | bytes       | ... 
| rlimit_memory_stack_hard       | bytes       | ...
| rlimit_memory_rss_soft         | bytes       | ...
| rlimit_memory_rss_hard         | bytes       | ...
| rlimit_num_fds_soft            | count       | ...
| rlimit_num_fds_hard            | count       | ...
| rlimit_memory_locked_soft      | bytes       | ...
| rlimit_memory_locked_hard      | bytes       | ...
| rlimit_memory_vms_soft         | bytes       | ...
| rlimit_memory_vms_hard         | bytes       | ...
| rlimit_file_locks_soft         | bytes       | ...
| rlimit_file_locks_hard         | bytes       | ...
| rlimit_signals_pending_soft    | count       | ...
| rlimit_signals_pending_hard    | count       | ...
| rlimit_nice_priority_soft      | count       | ...
| rlimit_nice_priority_hard      | count       | ...
| rlimit_realtime_priority_soft  | count       | ...
| rlimit_realtime_priority_hard  | count       | ...

#### processes
The `processes` metric family provides summary metrics derived from the list of processes matching the configured search criteria.
Each different metric in this metric family is distinguished by the value of the label named `field`.

| Field             | Units | Description                         |
|-------------------|-------|-------------------------------------|
| total             | count | total number of processes tabulated |
| total_threads     | count | total number of threads
| sleeping          | count | number of processes in sleeping state
| unknown           | count | number of processes in unknown state
| parked            | count | number of processes in parked state
| blocked           | count | number of processes in blocked state
| zombies           | count | number of processes in zombie state
| stopped           | count | number of processes in stopped state
| running           | count | number of processes in running state
| wait              | count | number of processes in wait state
| dead              | count | number of processes in dead state
| idle              | count | number of processes in idle state

## Configuration

### Asset registration

[Sensu Assets][2] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add sensu/sensu-processes-check
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][4].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-processes-check
  namespace: default
spec:
  command: >
    sensu-processes-check
    --search
    '[{"search_string": "sshd"}]'
  subscriptions:
  - system
  runtime_assets:
  - sensu/sensu-processes-check
```

## Usage examples

### Help output

```
Sensu Processes Check

Usage:
  sensu-processes-check [flags]
  sensu-processes-check [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -h, --help                 help for sensu-processes-check
  -s, --search string        An array of JSON search criteria, fields are "search_string", "severity", "number", "comparison", and "full_cmdline"
  -S, --suppress-ok-output   Aside from overal status, only output failures

Use "sensu-processes-check [command] --help" for more information about a command.
```

### Search string details

The search string is JSON array of processes to search for.  Each JSON object
in the array is made up of the following attributes:

|Attribute     |Default |Explanation                                                                                             |
|--------------|--------|--------------------------------------------------------------------------------------------------------|
|search_string |N/A     |The string to search for                                                                                |
|number        |1       |The number of occurrences to compare against                                                            |
|comparison    |>=      |The comparison between matches found and number above                                                   |
|severity      |2       |The severity should the string not be found                                                             |
|full_cmdline  |false   |Boolean as to whether or not to include full command line (path to command plus all args) in the search |

#### Process name caveat

By default (when `full_cmdline` is false) the search string is matched against
the process name only, without any pathing.  You need to understand the impact
this may have on your searches.

For example, if you use the following search string to make sure that `sshd`
is running on a Linux server, the following output may be produced:

```
sensu-processes-check -s '[{"search_string": "sshd"}]'
OK       | 3 >= 1 (found >= required) evaluated true for "sshd"
Status - OK
```

If you compare the output of `ps -e` and `ps -ef` you will see the 3 matches it
found.

```
ps -e | grep sshd
   1174 ?        00:00:00 sshd
1361376 ?        00:00:00 sshd
1361385 ?        00:00:00 sshd

ps -ef | grep sshd
root        1174       1  0 Oct29 ?        00:00:00 /usr/sbin/sshd -D -o [...extraneous output deleted]
root     1361376    1174  0 Nov02 ?        00:00:00 sshd: todd [priv]
todd     1361385 1361376  0 Nov02 ?        00:00:00 sshd: todd@pts/0

```

But maybe what you are really wanting is to make sure the parent listening
process is running.  The best bet would be to set `full_cmdline` to `true`
and set `search_string` to `/usr/sbin/sshd`.

```
sensu-processes-check -s '[{"search_string": "/usr/sbin/sshd", "full_cmdline": true}]'
OK       | 1 >= 1 (found >= required) evaluated true for "/usr/sbin/sshd"
Status - OK
```

#### Supported comparisons

When comparing the number of matches found with the requested number, the
following comparisons are supported:

* `==`
* `>=`
* `<=`
* `>`
* `<`

The comparison is always evaluated as number of matching processes found
`operator` number specified in the search criteria.

#### Example

Search criteria:
* At least 1 process named `ssh-agent` and report critical (2) if not found (defaults for `number`, `comparison`, `severity`, and `full_cmdline`)
* At least 2 processes named  `webapp1` and report warning (1) if not found (defaults for `comparison` and `full_cmdline`)
* Only 1 process with `eventmonitor` as part of the command line and report warning (1) if not found (defaults for `number`)

```
sensu-processes-check -s '[{"search_string": "ssh-agent"}, {"number": 2, "severity": 1, "search_string": "webapp1"}, {"full_cmdline": true, "comparison": "==", "severity": 1, "search_string": "eventmonitor"}]'
OK       | 1 >= 1 (found >= required) evaluated true for "ssh-agent"
OK       | 3 >= 2 (found >= required) evaluated true for "webapp1"
OK       | 1 == 1 (found == required) evaluated true for "eventmonitor"
Status - OK
```

### Exit severity

The process will exit with the highest severity encountered across all searches.
Given the same search criteria as the example above, if zero (0) instances of
"ssh-agent" were found, and only one (1) instance of "webapp1" were found, the
output would like similar to the below and the exit status would be 2
(critical).

```
sensu-processes-check -s '[{"search_string": "ssh-agent"}, {"number": 2, "severity": 1, "search_string": "webapp1"}, {"full_cmdline": true, "comparison": "==", "severity": 1, "search_string": "eventmonitor"}]'
CRITICAL | 0 >= 1 (found >= required) evaluated false for "ssh-agent"
WARNING  | 1 >= 2 (found >= required) evaluated false for "webapp1"
OK       | 1 == 1 (found == required) evaluated true for "eventmonitor"
Status - CRITICAL
```

### Annotations

The arguments for this check are tunable on a per entity basis.  The annotations
keyspace for this check is `sensu.io/plugins/sensu-processes-check/config`.
Here is how you would set the search string for an entity to override the
existing check definition:

```yaml
type: Entity
api_version: core/v2
metadata:
  annotations:
    sensu.io/plugins/sensu-processes-check/config/search: '[{"search_string":"qmgr"},{"search_string":"pickup"},{"search_string":"chronyd"}]'
[...remaining lines deleted]
```

It should be noted that annotations completely override the existing argument.
Should you want to use [check token substitution][3] with an annotation, you
will need to use a different annotation key.  Also, when using check token
substitution it becomes necessary to escape the JSON that makes up the
argument to the `--search` option both for the command definition and in the
annotation itself.  The examples below show the necessary changes.

Here is the relevant portion of the check definition:

```yaml
type: Check
api_version: core/v2
metadata:
  name: processes-check
  namespace: default
spec:
  check_hooks: null
  command: |
    sensu-processes-check --search "{{ .annotations.sensu_processes_check_search | default `[{\"search_string\": \"qmgr\"}, {\"search_string\": \"pickup\"}]` }}"
[...remaining lines deleted]
```

And here is the relevant portion of the entity annotation:

```yaml
type: Entity
api_version: core/v2
metadata:
  annotations:
    sensu_processes_check_search: '[{\"search_string\":\"qmgr\"},{\"search_string\":\"pickup\"},{\"search_string\":\"gssproxy\"}]'
[...remaining lines deleted]
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-processes-check repository:

```
go build
```

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[2]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[3]: https://docs.sensu.io/sensu-go/latest/observability-pipeline/observe-schedule/checks/#check-token-substitution
[4]: https://bonsai.sensu.io/assets/sensu/sensu-processes-check
