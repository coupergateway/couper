# Couper Changelog

<a name="v0.2"></a>
## [v0.2](https://github.com/avenga/couper/compare/v0.1...v0.2)

> 2020-10-08

### Added

* Add health check ([#29](https://github.com/avenga/couper/issues/29))
* Add Basic-Auth support ([#19](https://github.com/avenga/couper/issues/19))
* Add post (form) parsing for use in config variables ([#26](https://github.com/avenga/couper/issues/26))
* Add more documentation

### Fix

* Fix wildcard path join with trailing slash and respect req path ([#45](https://github.com/avenga/couper/pull/45))
* Fix env var mapping ([#35](https://github.com/avenga/couper/pull/35))
* Fix JWT HMAC keys ([#32](https://github.com/avenga/couper/pull/32))


<a name="0.1"></a>
## 0.1

> 2020-09-11

### Add

* Parse and load from given *HCL* configuration file
* Config structs for blocks: `server, api, endpoint, files, spa, definitions, jwt`
* HTTP handler implementation for `api backends, files, spa` and related config mappings
* CORS handling for `api` endpoints
* Access control configuration for all blocks
* Access control type `jwt` with claim validation
* _Access_ und _backend_ logs
* Configurable error templates with a fallback to our [defaults](./assets/files)
* Github actions for our continuous integration workflows
* [Dockerfile](./Dockerfile)
* [Documentation](./docs)
