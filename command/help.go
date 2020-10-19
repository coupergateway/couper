package command

// Help shows available commands and options.
func Help() { // TODO: generate from command and options list
	println(`couper usage:

couper <cmd> <options>

global options:

	-f		couper hcl configuration file
	-log-format	format option for json or common logs

available commands:

	run		starts the server
`)
}
