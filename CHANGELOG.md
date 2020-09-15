
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
