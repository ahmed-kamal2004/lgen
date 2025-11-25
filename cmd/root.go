package cmd

import (
	. "generator/load/cmd/grpc_cmd"
	. "generator/load/cmd/http_cmd"

	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "load-generator",
	}

	cmd.AddCommand(NewGrpcCommand())
	cmd.AddCommand(NewHttpCommand())

	return cmd
}



