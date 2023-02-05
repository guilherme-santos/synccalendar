package main

import "strings"

type Strings []string

func (i *Strings) String() string {
	return strings.Join(*i, ", ")
}

func (i *Strings) Set(value string) error {
	*i = append(*i, value)
	return nil
}
