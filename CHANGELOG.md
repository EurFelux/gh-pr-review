# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed

- Add `PENDING` to allowed review states in `review view --states` filter.
- Upgrade golangci-lint-action to v7 for golangci-lint v2 support.

## [2.2.0] - 2026-02-04

### Changed

- **BREAKING:** Change `review preview` filter from `--comment-id` to `--thread-id`.
- Add golangci-lint v2 config and fix lint issues.
- Update review workflow to verify each comment immediately after adding.

## [2.1.0] - 2026-02-04

### Added

- Add `--comment-id` filter to `review preview` command.

## [2.0.0] - 2026-02-04

### Changed

- **BREAKING:** Split `review` command from flags to subcommands (`start`, `add-comment`, `edit-comment`, `delete-comment`, `submit`, `preview`, `view`).

[Unreleased]: https://github.com/EurFelux/gh-pr-review/compare/v2.2.0...HEAD
[2.2.0]: https://github.com/EurFelux/gh-pr-review/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/EurFelux/gh-pr-review/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/EurFelux/gh-pr-review/compare/v1.7.0...v2.0.0
