package grpc_cmd

import (
	"strings"

	"generator/load/src/grpc"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/spf13/cobra"
)

func NewGrpcCommand() *cobra.Command {
	grpcCmd := &cobra.Command{
		Use: "grpc",
		RunE: grpcExecute,
	}

	var proto_path string
	var destination string
	var req_num int
	var targetMethod string
	var timeout int

	grpcCmd.Flags().StringVar(&destination, "destination", "", "Destination Address")
	grpcCmd.Flags().StringVar(&targetMethod, "tarm", "", "Target method to test on it")
	grpcCmd.Flags().StringVar(&proto_path, "proto", "", "Path to the target proto file")
	grpcCmd.Flags().IntVar(&req_num, "reqn", 10 , "Number of requests")
	grpcCmd.Flags().IntVar(&timeout, "timeout", 5, "Timeout for the requests")

	return grpcCmd
}


func grpcExecute(cmd *cobra.Command, args []string) error {

	dest, _ := cmd.Flags().GetString("destination")
	req_num, _ := cmd.Flags().GetInt("reqn")
	timeout, _ := cmd.Flags().GetInt("timeout")

	method := getMethod(cmd)

	if method == nil {
		return nil
	}

	grpc_req := grpc.GenerateGrpcReq(dest, method, req_num, timeout)

	if grpc_req != nil {
		grpc_req.GenerateGenericLoad()
	}
	return nil
}

func getMethod(cmd *cobra.Command)( m *desc.MethodDescriptor) {
	// Get Flags 
	filepath, _ := cmd.Flags().GetString("proto")
	targetMethod, _ := cmd.Flags().GetString("tarm")

	// Parse proto
	parser := protoparse.Parser{}
	fds, err := parser.ParseFiles(filepath)
	if err != nil {
		return nil
	}

	var method *desc.MethodDescriptor = nil
	services := fds[0].GetServices()
	for _, svc := range services {
		for _, m := range svc.GetMethods() {
			if  strings.ToLower(m.GetName()) == strings.ToLower(targetMethod) {
				method = m
			}
		}
	}

	return method
}