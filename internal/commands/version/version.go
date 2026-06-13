package version

import "fmt"

var Version = "dev"

type Cmd struct{}

func (c *Cmd) Run() error {
	fmt.Println(Version)
	return nil
}
