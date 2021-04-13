package command

import (
	"flag"
	"strings"
)

type Args []string

func NewArgs(args []string) Args {
	return args[1:]
}

// Filter returns all command line arguments which will match the given flag.FlagSet.
func (a Args) Filter(set *flag.FlagSet) Args {
	if set == nil {
		return a
	}
	var args Args
	for i, arg := range a {
		name := arg[1:]
		if idx := strings.Index(name, "="); idx > -1 {
			name = name[:idx]
		}
		if f := set.Lookup(name); f != nil {
			if name == arg[1:] {
				if iFn, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
					if iFn.IsBoolFlag() {
						args = append(args, a[i:i+1]...)
						continue
					}
				}

				if len(a[i:]) > 1 {
					args = append(args, a[i:i+2]...)
				} else {
					args = append(args, a[i:i+1]...)
				}

				continue
			}
			args = append(args, a[i])
		}
	}
	return args
}
