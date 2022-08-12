import grammar from "./hcl.tmLanguage.json" assert {type: "json"}

const name = grammar.name
const scopeName = grammar.scopeName
const comment = grammar.comment
const fileTypes = grammar.fileTypes
const patterns = grammar.patterns
const repository = grammar.repository

export {
	grammar as default,
	comment,
	fileTypes,
	name,
	patterns,
	repository,
	scopeName
}
