[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-processes-check)
![Go Test](https://github.com/sensu/sensu-processes-check/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-processes-check/workflows/goreleaser/badge.svg)

# Sensu Processes Check

## Table of Contents
- [Overview](#overview)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Usage examples](#usage-examples)
  - [Help output](#help-output)
  - [Search string details](#search-string-details)
- [Installation from source](#installation-from-source)
- [Contributing](#contributing)

## Overview

The Sensu Processes Check is a [Sensu Check][1] that searches for certain
running processes (or other strings in a command line).  It can search for
multiple processes and, on a per-string basis, set the number of processes
expected, severity if the number of processes is not met, and whether not
to search the full command line for the requested string.

## Configuration

### Asset registration

[Sensu Assets][2] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add sensu/sensu-processes-check
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/sensu/sensu-processes-check].

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
  -h, --help            help for sensu-processes-check
  -s, --search string   An array of JSON search criteria, fields are "search_string", "severity", "number", "comparison", and "full_cmdline"

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
