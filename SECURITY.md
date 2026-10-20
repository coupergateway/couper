# Security Policy

## Supported Versions

Security fixes are released for the current minor release line only.

| Version | Supported |
|---------|-----------|
| 1.14.x  | Yes       |
| < 1.14  | No        |

Please upgrade to the latest 1.14.x release before you report an issue.
Backports to older release lines are not provided.

## Reporting a Vulnerability

Please do not open a public issue for security problems.

Use GitHub's private vulnerability reporting for this repository:

https://github.com/coupergateway/couper/security/advisories/new

A report should contain:

- the affected Couper version and how you run it (binary, container, `edge`)
- a minimal configuration (HCL) that reproduces the problem
- the exact request or steps to trigger it, for example a `curl` command
- the observed and the expected behavior
- an assessment of the impact, if you have one

## What to Expect

- We confirm receipt of a report within 5 working days.
- We keep you informed while we analyze and fix the issue.
- We coordinate the disclosure date with you. By default we publish the
  advisory together with the fixed release.
- We credit the reporter in the advisory unless you ask us not to.

## Scope

In scope are the Couper binary and container images built from this
repository. Out of scope are example configurations, third-party backends
and problems in dependencies that are already reported upstream.
