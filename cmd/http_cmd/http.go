package http_cmd

import (
	"generator/load/src/http"

	"github.com/spf13/cobra"
)



func NewHttpCommand() *cobra.Command {
	cmd:= &cobra.Command{
		Use: "http",
		RunE: httpExecute,

	}

	var destination string
	var requestbody_path string
	var reqnum int
	var workerconc int
	var httpmethod string
	var timeout int
	var maxretries int

	cmd.AddCommand(NewSseCommand())
	cmd.AddCommand(NewCsCommand())

	cmd.Flags().StringVar(&destination, "destination", "http://localhost:80/", "Full destination including protocol, address, port and url")
	cmd.Flags().StringVar(&requestbody_path, "reqb_path", "", "Path to the file containing the request body for POST requests")
	cmd.Flags().IntVar(&reqnum, "reqn", 1, "Number of requests to be done")
	cmd.Flags().IntVar(&workerconc, "conc", 1, "Number of concurrent requests at the same time")
	cmd.Flags().StringVar(&httpmethod, "method", "POST", "HTTP method to use: GET, POST, PUT, DELETE")
	cmd.Flags().IntVar(&timeout, "timeout", 5, "Maximum number of seconds per request")
	cmd.Flags().IntVar(&maxretries, "maxr", 3, "Maximum number of retries per failed request")

	cmd.MarkFlagRequired("destination")

	return cmd
}


func httpExecute(cmd *cobra.Command, args []string) error {
	println("Here is HTTP, Connected successfully")
	destination, _ := cmd.Flags().GetString("destination")
	reqnum, _ := cmd.Flags().GetInt("reqn")
	workerconc, _ := cmd.Flags().GetInt("conc")
	timeout, _ := cmd.Flags().GetInt("timeout")
	maxretries, _ := cmd.Flags().GetInt("maxr")
	reqBody, _ := cmd.Flags().GetString("reqb_path")
	reqMethod , _ := cmd.Flags().GetString("method")

	h := http.GenerateHttpReq(destination, reqBody, reqnum, workerconc, reqMethod, timeout, maxretries, 0)

	h.GenerateGenericLoad()
	
	return nil
}