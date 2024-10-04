package main

import (
	"github.com/macvmio/geranos/cmd/geranos/cmd"
)

func main() {
	cmd.Execute(cmd.InitializeCommands())
}
