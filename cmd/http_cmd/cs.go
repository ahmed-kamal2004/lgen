package http_cmd

import (
	"generator/load/src/http"

	"github.com/spf13/cobra"
)

func NewCsCommand() *cobra.Command {
	cmd:= &cobra.Command{
		Use: "cs",
		RunE: csHttpExecute,
	}

	var destination string
	var reqnum int
	var workerconc int
	var timeout int
	var maxretries int
	var size int

	cmd.Flags().StringVar(&destination, "destination", "http://localhost:80/", "Full destination including protocol, address, port and url")
	cmd.Flags().IntVar(&reqnum, "reqn", 1, "Number of requests to be done")
	cmd.Flags().IntVar(&workerconc, "conc", 1, "Number of concurrent requests at the same time")
	cmd.Flags().IntVar(&timeout, "timeout", 10, "Maximum number of seconds per request")
	cmd.Flags().IntVar(&maxretries, "maxr", 3, "Maximum number of retries per failed request")
	cmd.Flags().IntVar(&size, "size", 16 * 1024*1024, "Size of the file to be uploaded.")

	cmd.MarkFlagRequired("url")
	cmd.MarkFlagRequired("address")

	return cmd
}


func csHttpExecute(cmd *cobra.Command, args []string) error {
	destination, _ := cmd.Flags().GetString("destination")
	reqnum, _ := cmd.Flags().GetInt("reqn")
	workerconc, _ := cmd.Flags().GetInt("conc")
	timeout, _ := cmd.Flags().GetInt("timeout")
	maxretries, _ := cmd.Flags().GetInt("maxr")
	size, _ := cmd.Flags().GetInt("size")
	println("Here is CS, Connected successfully")

	h := http.GenerateHttpReq(destination, "", reqnum, workerconc, "CS", timeout, maxretries, size)

	h.GenerateCsLoad()
	
	return nil
}