# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic
Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased
### Added
- Sumo Logic Dashboard compatible metric output

### Changed
- Allow for empty search configuration to match full process list
- Added metrics only flag to disable search based alerts
- Converted search alert text output into metric comment strings
- Update to sensu plugin sdk 0.15.0
- Changed types import to corev2
- Minor README fix

## [0.2.0] - 2020-11-09

### Changed
- Add suppot for suppressing OK output lines
- Updated README to include information on annotations and check token substitution
- If a search string is "" then skip it, do not fail it

## [0.1.0] - 2020-11-04

### Added
- Initial release
