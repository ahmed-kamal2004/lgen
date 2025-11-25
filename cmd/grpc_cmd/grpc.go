package grpc_cmd

import "github.com/spf13/cobra"

func NewGrpcCommand() *cobra.Command {
	return &cobra.Command{
		Use: "grpc",
		RunE: func(cmd *cobra.Command, args []string) error {
			println("Here is Grpc, Connected successfully")
			return nil
		},

	}
}