package http

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type HttpReq struct {
	destination string // full destination including protocol, address, port and url and query parameters for get requests.
	requestBody string // body to be sent with the request "if POST request".
	reqNum int // number of requests to be done.
	workerConc int // number of concurrent requests at the same time.
	httpMethod string // GET, POST, PUT, DELETE 
	timeout int // maximum number of seconds per request.
	maxRetries int // maximum number of retries per failed request.
	fileSize int // size of the file to be uploaded
}

type requestStat struct {
	latency float64
	method string
	successful bool
}

type sseRequestStat struct {
	latency float32
	events int
	successful bool
}

type csRequestStat struct {
	latency float64
	successful bool
}

type requestStats struct {
	request_stats []requestStat
	throughput float64
	total_time float64
	successful_requests int
	failed_requests int
}	


////////////////////////// Exported Methods /////////////////////////

func GenerateHttpReq(destination string, requestbody_path string, reqNum int, workerConc int, httpMethod string, timeout int, maxRetries int, fileSize int) *HttpReq {

	var err error
	var requestBodyBytes []byte
	if httpMethod == "POST" || httpMethod == "PUT" {
		requestBodyBytes, err = os.ReadFile(requestbody_path)
		if err != nil {
			println("Error reading request body file:", err.Error())
			requestBodyBytes = []byte{}
		}
	}

	return &HttpReq{
		destination: destination,
		requestBody: string(requestBodyBytes),
		reqNum: reqNum,
		workerConc: workerConc,
		httpMethod: httpMethod,
		timeout: timeout,
		maxRetries: maxRetries,
		fileSize: fileSize,
	}
}

func (h *HttpReq) GenerateSseLoad(){
	client := h.generateClient(true) // timeout for SSE
	var wg sync.WaitGroup
	var result_collector sync.WaitGroup
	output := make(chan sseRequestStat)
	for i:= 0; i < h.reqNum; i++ {
		wg.Add(1)
		go h.generate_one_sse_load(client, &wg, output)
	}
	go func (ch <- chan sseRequestStat, wg *sync.WaitGroup)  {
		wg.Add(1)
		defer wg.Done()
		var total_latency float32 = 0
		var total_count int = 0
		var successful int = 0
		for {
			select {
			case val, ok := <-ch:
				if !ok {
					fmt.Printf("Average Latency: %.3f\n", total_latency/float32(total_count))
					fmt.Printf("Total Success percent: %.2f%%\n", float32(successful)/float32(total_count)*100)
					return
				}
				println("Latency: ", val.latency)
				println("Events Received: ", val.events)
				println("Successful: ", val.successful)
				total_latency += val.latency
				total_count ++
				if val.successful {
					successful ++
				}
			}
		}
	}(output, &result_collector)


	wg.Wait()
	close(output)
	result_collector.Wait()
}


func (h *HttpReq) GenerateGenericLoad() {
	client := h.generateClient(false) // no timeout for Generic Unary
	var wg sync.WaitGroup
	var result_collector sync.WaitGroup
	output := make(chan requestStat)
	for i:= 0; i < h.reqNum; i++ {
		wg.Add(1)
		go h.generate_one_generic_load(client, &wg, output)
	}
	go func (ch <- chan requestStat, wg *sync.WaitGroup)  {
		wg.Add(1)
		defer wg.Done()
		var total_latency float32 = 0
		var total_count int = 0
		var successful int = 0
		for {
			select {
			case val, ok := <-ch:
				if !ok {
					fmt.Printf("Average Latency: %.3f\n", total_latency/float32(total_count))
					fmt.Printf("Total Success percent: %.2f%%\n", float32(successful)/float32(total_count)*100)
					return
				}
				println("Latency: ", val.latency)
				println("Successful: ", val.successful)
				total_latency += float32(val.latency)
				total_count ++
				if val.successful {
					successful ++
				}
			}
		}
	}(output, &result_collector)


	wg.Wait()
	close(output)
	result_collector.Wait()
}

func (h *HttpReq) GenerateCsLoad() {
	client := h.generateClient(false)

	// Generate file
	var err error
	var filepath string  = ""
	filepath, err = os.Getwd()
	if err != nil {
		return
	}
	filepath = filepath + "/demo.txt"

	err = os.WriteFile(filepath, make([]byte, h.fileSize), os.FileMode(777))
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	var result_collector sync.WaitGroup
	output := make(chan csRequestStat)
	for i:= 0; i < h.reqNum; i++ {
		wg.Add(1)
		go h.generate_one_cs_load(client, &wg, output)
	}
	go func (ch <- chan csRequestStat, wg *sync.WaitGroup)  {
		wg.Add(1)
		defer wg.Done()
		var total_latency float32 = 0
		var total_count int = 0
		var successful int = 0
		for {
			select {
			case val, ok := <-ch:
				if !ok {
					fmt.Printf("Average Latency: %.3f\n", total_latency/float32(total_count))
					fmt.Printf("Total Success percent: %.2f%%\n", float32(successful)/float32(total_count)*100)
					return
				}
				println("Latency: ", val.latency)
				println("Successful: ", val.successful)
				total_latency += float32(val.latency)
				total_count ++
				if val.successful {
					successful ++
				}
			}
		}
	}(output, &result_collector)


	wg.Wait()
	close(output)
	result_collector.Wait()

	// Delete the generated file
	os.Remove(filepath)

	return
}

///////////////////////// Internal Methods /////////////////////////

func (h *HttpReq) generateClient(need_timeout bool) *http.Client {
	var timeout int = 0 
	if need_timeout { // for unary, else for SSE no timeout
		timeout = h.timeout
	}
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second  ,
	}
	return client
}


func (h *HttpReq) generate_one_generic_load(client * http.Client, wg *sync.WaitGroup, ch chan requestStat) {
	defer wg.Done()
	var successful_request bool = false
	var err error = nil
	var resp *http.Response
	

	time_before := time.Now()

	for attempt := 0; attempt < h.maxRetries; attempt++ {
		resp, err = client.Post(h.destination, "application/json", strings.NewReader(h.requestBody))
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			successful_request = true
			resp.Body.Close()
			break
		}else if resp != nil {
			resp.Body.Close()
		}
	}
	time_after := time.Now()
	request_latency := time_after.Sub(time_before).Seconds()


	ch <- requestStat{
		latency: request_latency,
		method: h.httpMethod,
		successful: successful_request}

	return
}


func (h *HttpReq) generate_one_sse_load(client * http.Client, wg *sync.WaitGroup, ch chan sseRequestStat) {
	defer wg.Done()

	time_before := time.Now()

	req, err := http.NewRequest("GET", h.destination, nil)
	// req.Header.Set("Accept", "text/event-stream") I think no need for it, right now at least
	if err != nil {
		ch <- sseRequestStat{
			latency: 0,
			events: 0,
			successful: false}
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		ch <- sseRequestStat{
			latency: 0,
			events: 0,
			successful: false}
		return
	}
	defer resp.Body.Close()
	
	attempts := 0
	reader := bufio.NewReader(resp.Body)
	events := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			if attempts >= h.maxRetries{
				break
			} else {
				attempts ++
				continue
			}
		}
		if len(line) > 6 && line[:5] == "data:" {
			events ++
		} else if len(line) == 0 {
			break
		}
	}

	time_after := time.Now()
	request_latency := time_after.Sub(time_before).Seconds()

	ch <- sseRequestStat{
		latency: float32(request_latency),
		events: events,
		successful: true}
}


func (h *HttpReq) generate_one_cs_load(client * http.Client, wg *sync.WaitGroup, ch chan csRequestStat){
	defer wg.Done()
	filepath, err := os.Getwd()
	filepath = filepath + "/demo.txt"

	file, err := os.Open(filepath)
	if err != nil {
		println("Unable to open file: ", filepath)
		ch <- csRequestStat{}
		return
	}

	req, err := http.NewRequest(http.MethodPost, h.destination, file)
	req.Header.Set("Content-Type", "application/octet-stream")

	before_time := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		println("Unable to send Client streaming request")
		ch <- csRequestStat{}
		return
	}

	after_time := time.Now()

	latency := after_time.Sub(before_time).Seconds()

	defer resp.Body.Close()
	ch <- csRequestStat{
		latency: latency,
		successful: true,
	}
	return 
}