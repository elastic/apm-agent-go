## CI/CD

There are 5 main stages that run on GitHub actions

* Lint
* Test
* Coverage
* Benchmarking
* Release

The whole process should look like:

`Checkout` -> `Lint` -> `Test` -> `Coverage` -> `Benchmark` -> `Release`

There are some other stages that run for every push on the main branches:

* [Snapshoty](./snapshoty.yml)
* [Microbenchmark](./microbenchmark.yml)

### Scenarios

* Matrix compatibility runs on branches, tags and PRs basis.
* Pull Requests that are only affecting the docs files should not trigger any test or similar stages that are not required.
* Builds do not get triggered automatically for Pull Requests from contributors that are not Elasticians when need to access to any GitHub Secrets.

### Compatibility matrix

Go agent supports compatibility to different Go versions, those are defined in:

* Go [versions](https://github.com/elastic/apm-agent-go/blob/main/.github/workflows/ci.yml) for all the `*nix` builds.

### How to interact with the CI?

#### On a PR basis

Once a PR has been opened then there are two different ways you can trigger builds in the CI:

1. Commit based
1. UI based, any Elasticians can force a build through the GitHub UI

#### Branches

Every time there is a merge to main or any release branches the whole workflow will compile and test every entry in the compatibility matrix for Linux, Windows and MacOS.

#### Release process

This process has been fully automated and it gets triggered when a tag release has been created, Continuous Deployment based, aka no input approval required.

### OpenTelemetry

There is a GitHub workflow in charge to populate what the workflow run in terms of jobs and steps. Those details can be seen in [here](https://ela.st/oblt-ci-cd-stats) (**NOTE**: only available for Elasticians).
