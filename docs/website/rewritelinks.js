const fs = require('fs')
const process = require('process')
const { red, green } = require('colorette')

const filename = process.argv[2]
const DUMMY_HOST = "http://localhost:1234"
const baseURI = DUMMY_HOST + process.argv[3] + "/"

const markdown = fs.readFileSync(filename, {encoding: "utf8"})
const newMarkdown = markdown.replaceAll(/\[(.*?)\]\(([a-z0-9._-].*?)\)/gi, (match, text, uri) => {
	let hostRelativeURI
	try {
		url = new URL(uri, baseURI)
		hostRelativeURI = url.href.replace(DUMMY_HOST, "")
		if (hostRelativeURI != uri) {
			console.log(green(`ℹ Rewriting ${uri} → ${hostRelativeURI}`))
		}
	} catch (e) {
		console.error(red(`‼ ${e.message}: ${uri}, ${baseURI}`))
		hostRelativeURI = uri
	}

	return `[${text}](${hostRelativeURI})`
})

fs.writeFileSync(filename, newMarkdown)
