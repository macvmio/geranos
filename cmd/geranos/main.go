package main

import (
	"github.com/tomekjarosik/geranos/cmd/geranos/cmd"
)

func main() {
	cmd.Execute(cmd.InitializeCommands())
}
