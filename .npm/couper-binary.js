const { existsSync, mkdirSync, unlinkSync, chmodSync, copyFileSync } = require("fs")
const axios = require("axios")
const tar = require("tar")
const crypto = require("crypto")
const unzip = require("unzip-stream")
const { join } = require("path")
const { spawnSync } = require("child_process")

const osMap = {
	"Linux":  "linux",
	"Darwin": "macos",
	"Windows_NT": "windows"
}

const archMap = {
	"x64": "amd64",
	"arm64": "arm64"
}

function getPlatform() {
	const os = require('os')
	const type = osMap[os.type()]
	const arch = archMap[os.arch()]

	if (!type || !arch) {
		throw `Sorry, ${this.name} is not available for your platform: ${os.type()}/${os.arch()}`
	}
	const binary  = type === "windows" ? "couper.exe" : "couper"
	const archive = type === "linux"   ? "tar.gz" : "zip"

	return {os: type, arch: arch, binary: binary, archive: archive}
}

class CouperBinary {

	constructor() {
		const { version } = require("./package.json")
		this.name = "couper"
		this.platform = getPlatform()
		// require from package.json fails for older node versions!
		this.url = "https://github.com/coupergateway/couper/releases/download/" +
				   `v${version}/${this.name}-v${version}-` +
				   `${this.platform.os}-${this.platform.arch}.${this.platform.archive}`

		this.targetDirectory = join(__dirname, "bin")
		this.binary = join(this.targetDirectory, this.platform.binary)
	}

	install() {
		mkdirSync(this.targetDirectory, { recursive: true })
	    const hash = crypto.createHash('sha256').setEncoding('hex')
		hash.on('finish', (() => {
		    hash.end()
			this.verify(hash)

			// executable permisson lost due to streaming
			// https://github.com/EvanOxfeld/node-unzip/issues/123
			chmodSync(this.binary, 0o755)

			// start via wrapper script on Windows
			if (this.platform.binary === "couper.exe") {
				const wrapper = join(__dirname, "bin", "couper")
				copyFileSync(wrapper + ".js", wrapper)
			}
		}).bind(this))

		console.log(`Downloading release from ${this.url}...`)
		return axios({
			url: this.url,
			responseType: "stream"
		})
		.then(response => {
			response.data.pipe(hash)
			if (this.platform.archive === "tar.gz") {
				return response.data.pipe(tar.x({
					strip: 0,
					cwd: this.targetDirectory
				}))
			} else if (this.platform.archive === "zip") {
				return response.data.pipe(unzip.Extract({path: this.targetDirectory}))
			}
			else {
				// FIXME handle plain files, too?
				throw new Error("Invalid archive format: " + this.platform.archive)
			}
		})
		.catch(e => {
			console.error(`Error installing release: ${e}`)
			process.exit(1)
		})
	}

	verify(hash) {
		console.log(`Verifying checksum...`)
		const url = this.url + ".sha256"
		return axios({
			url: url,
			responseType: "text"
		})
		.then(response => {
			const checksum = response.data.trim()
			if (hash.read() !== checksum) {
				unlinkSync(this.binary)
				throw new Error("Bad checksum!")
			}
			console.log(`${this.name} successfully installed: ${this.binary}`)
		}).catch(e => {
			console.error(`Error downloading checksum ${url}: ${e}`)
			process.exit(1)
		})
	}

	run() {
		if (!existsSync(this.binary)) {
			console.error(`Please install ${this.name} first: npm install`)
			process.exit(1)
		}

		const [, , ...args] = process.argv
		const options = { cwd: process.cwd(), stdio: "inherit" }
		const result = spawnSync(this.binary, args, options)
		if (result.error) {
			console.error(result.error)
		}

		process.exit(result.status)
	}
}

module.exports.CouperBinary = CouperBinary
