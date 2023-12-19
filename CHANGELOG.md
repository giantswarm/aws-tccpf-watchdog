# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Add `global.podSecurityStandards.enforced` value for PSS migration.
### Changed

- Configure `gsoci.azurecr.io` as the default container image registry.

## [0.2.0] - 2023-07-13

### Fixed

- Add required values for pss policies.

### Added

- Add use of the runtime/default seccompprofile.

### Changed

- Allowed more volumes in the PSP so that the seccompprofile won't stop pods from running.
- Update to Go 1.18.

## [0.1.0] - 2022-04-13

### Added

- First release.

[Unreleased]: https://github.com/giantswarm/aws-tccpf-watchdog/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/giantswarm/aws-tccpf-watchdog/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/giantswarm/aws-tccpf-watchdog/compare/v0.0.0...v0.1.0
