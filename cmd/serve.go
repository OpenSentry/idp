package cmd

import (
	"fmt"
)

type ServeCmd struct {
	// Version   bool `short:"v" long:"version" description:"Display version"`
}

func (v *ServeCmd) Execute(args []string) error {
	fmt.Println("ServeCmd")
	fmt.Printf("%#v\n", v)
	fmt.Printf("%#v\n", Application.Config)

	return nil
}
