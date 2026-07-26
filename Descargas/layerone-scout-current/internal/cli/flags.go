package cli

import (
	"flag"
)

type FlagSet struct {
	*flag.FlagSet
	touched map[string]bool
}

func NewFlagSet(name string) *FlagSet {
	return &FlagSet{
		FlagSet: flag.NewFlagSet(name, flag.ExitOnError),
		touched: make(map[string]bool),
	}
}

func (f *FlagSet) Parse(args []string) error {
	if err := f.FlagSet.Parse(args); err != nil {
		return err
	}
	f.Visit(func(fl *flag.Flag) {
		f.touched[fl.Name] = true
	})
	return nil
}

func (f *FlagSet) WasSet(name string) bool {
	return f.touched[name]
}

func (f *FlagSet) StringVar(p *string, name, value, usage string) {
	f.FlagSet.StringVar(p, name, value, usage)
}

func (f *FlagSet) IntVar(p *int, name string, value int, usage string) {
	f.FlagSet.IntVar(p, name, value, usage)
}

func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string) {
	f.FlagSet.BoolVar(p, name, value, usage)
}

func (f *FlagSet) OptionalBool(name, usage string) *bool {
	var v bool
	f.BoolVar(&v, name, false, usage)
	return &v
}
