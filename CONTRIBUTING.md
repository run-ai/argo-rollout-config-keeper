# How to contribute to argo-rollout-config-keeper

This project is Apache 2.0 licensed and accepts contributions via GitHub pull requests. This document outlines some of the conventions on commit message formatting, contact points for developers, and other resources to make it easier to get your contribution accepted.

We gratefully welcome improvements to documentation as well as to code.

This project is a regular Kubernetes Operator built using the Operator SDK. Refer to the Operator SDK documentation to understand the basic architecture of this operator.

## Getting started
- Install the Operator SDK CLI by following the [installation guide](https://sdk.operatorframework.io/docs/installation/).
- Fork the repository on GitHub
- Read the [README](README.md) for build and test instructions

## Contribution flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where to base the contribution. This is usually master.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request to operator-framework/operator-sdk.
- The PR must receive a LGTM from two maintainers found in the MAINTAINERS file.

Thanks for contributing!

### Code style

The coding style suggested by the Go community is used in operator-sdk. See the [style doc](https://google.github.io/styleguide/go/decisions) for details.

Please follow this style to make operator-sdk easy to review, maintain and develop.

### Format of the commit message

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
scripts: updating function x

This will solve prblem y and z.

Fixes #38
```