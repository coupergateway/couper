const { existsSync, mkdirSync, unlinkSync, chmodSync, copyFileSync } = require("fs")
const https = require("https")
const crypto = require("crypto")
const unzip = require("unzip-stream")
const { join } = require("path")
const { spawnSync, spawn } = require("child_process")

const osMap = {
	"Linux":  "linux",
	"Windows_NT": "windows"
}

const archMap = {
	"x64": "amd64",
	"arm64": "arm64"
}

function download(url) {
	return new Promise((resolve, reject) => {
		https.get(url, (res) => {
			if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
				return download(res.headers.location).then(resolve, reject)
			}
			if (res.statusCode !== 200) {
				return reject(new Error(`Download failed: HTTP ${res.statusCode} for ${url}`))
			}
			resolve(res)
		}).on("error", reject)
	})
}

function downloadText(url) {
	return download(url).then((res) => {
		return new Promise((resolve, reject) => {
			let data = ""
			res.on("data", (chunk) => data += chunk)
			res.on("end", () => resolve(data))
			res.on("error", reject)
		})
	})
}

function getPlatform() {
	const os = require('os')
	const type = os.type()
	const arch = archMap[os.arch()]

	if (type === "Darwin") {
		console.error(
			"Couper is not available via npm on macOS.\n" +
			"Install via Homebrew:  brew install coupergateway/couper/couper\n" +
			"Or build from source:  go install github.com/coupergateway/couper@latest"
		)
		process.exit(1)
	}

	const mapped = osMap[type]
	if (!mapped || !arch) {
		console.error(`Sorry, couper is not available for your platform: ${type}/${os.arch()}`)
		process.exit(1)
	}

	const binary  = mapped === "windows" ? "couper.exe" : "couper"
	const archive = mapped === "linux"   ? "tar.gz" : "zip"

	return {os: mapped, arch: arch, binary: binary, archive: archive}
}

class CouperBinary {

	constructor() {
		const { version } = require("./package.json")
		this.name = "couper"
		this.platform = getPlatform()
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
		return download(this.url)
		.then(response => {
			response.pipe(hash)
			if (this.platform.archive === "tar.gz") {
				const which = spawnSync('tar', ['--version'])
				if (which.error) {
					console.error('Error: "tar" command not found. Please install tar (e.g., apk add tar).')
					process.exit(1)
				}
				const tarProcess = spawn('tar', ['xzf', '-', '-C', this.targetDirectory])
				response.pipe(tarProcess.stdin)
				return new Promise((resolve, reject) => {
					tarProcess.on('close', (code) => {
						if (code !== 0) reject(new Error(`tar extraction failed with exit code ${code}`))
						else resolve()
					})
					tarProcess.on('error', reject)
				})
			} else if (this.platform.archive === "zip") {
				return response.pipe(unzip.Extract({path: this.targetDirectory}))
			}
			else {
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
		return downloadText(url)
		.then(data => {
			const checksum = data.trim()
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
