//go:generate protoc -I . -I $GOPATH/src -I $GOPATH/src/github.com/kubernetes/cri-api/pkg/apis/runtime/v1 --go_out=. --go-grpc_out=. api.proto

package main

import (
	"github.com/tomekjarosik/geranos/cmd/geranos/cmd"
)

func main() {
	cmd.Execute(cmd.InitializeCommands())
}
