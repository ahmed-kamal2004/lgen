package http_cmd

import (
	"generator/load/src/http"

	"github.com/spf13/cobra"
)

func NewSseCommand() *cobra.Command {
	cmd:= &cobra.Command{
		Use: "sse",
		RunE: sseHttpExecute,
	}

	var destination string
	var reqnum int
	var workerconc int
	var timeout int
	var maxretries int

	cmd.Flags().StringVar(&destination, "destination", "http://localhost:80/", "Full destination including protocol, address, port and url")
	cmd.Flags().IntVar(&reqnum, "reqn", 1, "Number of requests to be done")
	cmd.Flags().IntVar(&workerconc, "conc", 1, "Number of concurrent requests at the same time")
	cmd.Flags().IntVar(&timeout, "timeout", 10, "Maximum number of seconds per request")
	cmd.Flags().IntVar(&maxretries, "maxr", 3, "Maximum number of retries per failed request")

	cmd.MarkFlagRequired("url")
	cmd.MarkFlagRequired("address")

	return cmd
}


func sseHttpExecute(cmd *cobra.Command, args []string) error {
	destination, _ := cmd.Flags().GetString("destination")
	reqnum, _ := cmd.Flags().GetInt("reqn")
	workerconc, _ := cmd.Flags().GetInt("conc")
	timeout, _ := cmd.Flags().GetInt("timeout")
	maxretries, _ := cmd.Flags().GetInt("maxr")
	println("Here is SSE, Connected successfully")

	println(destination, reqnum, workerconc, timeout, maxretries)
	h := http.GenerateHttpReq(destination, "", reqnum, workerconc, "SSE", timeout, maxretries, 0)

	h.GenerateSseLoad()

	return nil
}