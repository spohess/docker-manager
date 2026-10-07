package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type Options struct {
	Project string
	Config  string
	Profile string
	Full    bool
	Build   bool
	DryRun  bool
}

type Invocation struct {
	Command string
	Args    []string
	Options
}

var forwardingFlags = map[string]bool{
	"p":       true,
	"project": true,
	"config":  true,
	"dry-run": true,
}

func Parse(args []string, passthrough func(string) bool) (Invocation, error) {
	var inv Invocation
	fs := newFlagSet(&inv.Options)
	flags := split(fs, args, passthrough, &inv)
	if err := fs.Parse(flags); err != nil {
		return inv, err
	}
	if fs.NArg() > 0 {
		return inv, fmt.Errorf("argumento inesperado: %q", fs.Arg(0))
	}
	return inv, nil
}

func newFlagSet(opts *Options) *flag.FlagSet {
	fs := flag.NewFlagSet("docker-manager", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	fs.StringVar(&opts.Project, "project", "", "")
	fs.StringVar(&opts.Project, "p", "", "")
	fs.StringVar(&opts.Config, "config", "", "")
	fs.StringVar(&opts.Profile, "profile", "", "")
	fs.BoolVar(&opts.Full, "full", false, "")
	fs.BoolVar(&opts.Full, "f", false, "")
	fs.BoolVar(&opts.Build, "build", false, "")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "")
	return fs
}

func split(fs *flag.FlagSet, args []string, passthrough func(string) bool, inv *Invocation) []string {
	var flags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			inv.Args = append(inv.Args, args[i+1:]...)
			return flags
		}
		forwarding := inv.Command != "" && passthrough(inv.Command)
		name, inline, isFlag := parseFlag(arg)
		if !isFlag {
			if inv.Command == "" {
				inv.Command = arg
				continue
			}
			if forwarding {
				inv.Args = append(inv.Args, args[i:]...)
				return flags
			}
			inv.Args = append(inv.Args, arg)
			continue
		}
		if forwarding && !forwardingFlags[name] {
			inv.Args = append(inv.Args, args[i:]...)
			return flags
		}
		flags = append(flags, arg)
		if takesValue(fs, name) && !inline && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags
}

func parseFlag(arg string) (name string, inline bool, ok bool) {
	if len(arg) < 2 || arg[0] != '-' {
		return "", false, false
	}
	name = strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
	name, _, inline = strings.Cut(name, "=")
	return name, inline, name != ""
}

func takesValue(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	boolFlag, ok := f.Value.(interface{ IsBoolFlag() bool })
	return !ok || !boolFlag.IsBoolFlag()
}
