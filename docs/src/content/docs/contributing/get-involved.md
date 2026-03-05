---
title: "Get Involved"
description: "Feature requests, bug reports, and contributions are all welcome."
category: "contributing"
order: 2
lastUpdated: 2026-03-05
---

# Get Involved

Hatch is open source and actively developed. Whether you want to report a bug, suggest a feature, or submit code, your input is welcome.

## Feature requests

Have an idea for something Hatch should do? [Open a feature request](https://github.com/httphatch/hatch/issues/new?labels=enhancement&template=feature_request.md) on GitHub. Include as much context as you can: what problem it solves, how you expect it to work, and any alternatives you considered.

Good feature requests are specific. "Support for Windows" is a feature request. "Make it better" is not.

## Bug reports

If something is broken, [open a bug report](https://github.com/httphatch/hatch/issues/new?labels=bug&template=bug_report.md). Include:

- Your macOS version
- Your Hatch version (`hatch version`)
- Steps to reproduce the issue
- What you expected vs. what happened
- Relevant log output from `~/.hatch/logs/hatch.log`

## Contributing code

Pull requests are welcome for bug fixes, new features, documentation improvements, and test coverage. Before starting work on a large change, open an issue first to discuss the approach.

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Run tests: `go test ./...`
5. Run static analysis: `go vet ./...`
6. Open a pull request

See the [Development Setup](/docs/contributing/development-setup) guide for build instructions and project structure.

## Documentation

Found a typo? Missing a guide? Documentation lives in the `docs/` directory of the repository. Every docs page has an "Edit on GitHub" link at the bottom. Small fixes can go straight to a PR. For larger additions, open an issue to discuss scope first.

## Spread the word

If Hatch has been useful to you, star the [GitHub repository](https://github.com/httphatch/hatch) and share it with other developers working on local development.
