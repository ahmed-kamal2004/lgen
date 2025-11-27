package grpc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	dpb "google.golang.org/protobuf/types/descriptorpb"
)

type grpcReq struct {
	destination string
	method *desc.MethodDescriptor
	req_num int
	timeout int
}


type reqStat struct {
	latency float32
	successful bool
}

/// API

func GenerateGrpcReq(dest string, method *desc.MethodDescriptor, reqn int, timeout int) *grpcReq {
	return &grpcReq{
		destination: dest,
		method: method,
		req_num: reqn,
		timeout: timeout,
	}
}

func (g *grpcReq) GenerateLoad() {
	output := make(chan reqStat)
	var wg sync.WaitGroup
	var result_collector sync.WaitGroup

	time_before := time.Now()
	// Shared between requests
	conn, err := grpc.Dial(g.destination, grpc.WithInsecure())
	if err != nil {
		return
	}
	defer conn.Close()
	for i:= 0; i< g.req_num;i++ {
		wg.Add(1)
		if !g.method.IsClientStreaming() && !g.method.IsServerStreaming() {
			go g.generate_one_generic_load(conn, &wg, output)
		} else if g.method.IsClientStreaming() && !g.method.IsServerStreaming() {
			//
		} else if g.method.IsServerStreaming() && !g.method.IsClientStreaming() {
			go g.generate_one_servers_load(conn, &wg, output)
		}
	}
	
	go func (ch <- chan reqStat, wg *sync.WaitGroup) {
		wg.Add(1)
		defer wg.Done()
		var total_latency float32
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
				// println("Latency: ", val.latency)
				// println("Successful: ", val.successful)
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

	time_after := time.Now()
	total_time_taken := time_after.Sub(time_before).Seconds()

	fmt.Printf("Total time taken: %.4f\n", float32(total_time_taken))
	fmt.Printf("Total throughput: %.4f\n", float32(g.req_num)/float32(total_time_taken))


}

/// Internal

func randString(n int) string {
	var letters = []rune("0123")
    rand.Seed(uint64(time.Now().UnixNano()))
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}


func (g *grpcReq) generate_one_generic_load(conn *grpc.ClientConn, wg *sync.WaitGroup, ch chan reqStat){
	defer wg.Done()
	fullMethodName := fmt.Sprintf(
		"/%s.%s/%s",
		g.method.GetService().GetFile().GetPackage(),
		g.method.GetService().GetName(),
		g.method.GetName(),
	)
	req := dynamic.NewMessage(g.method.GetInputType())
	fields := req.GetKnownFields()
	for _ , field := range fields {
		if field.GetType() == dpb.FieldDescriptorProto_TYPE_STRING {
			req.SetFieldByName(field.GetName(), randString(2)) // Random Generation, to be fixed
		}
	}
	time_before := time.Now()
	resp := dynamic.NewMessage(g.method.GetOutputType())
	err := grpc.Invoke(
			context.Background(),
			fullMethodName,
			req,
			resp,
			conn,
			)	
	time_after := time.Now()
	if err != nil {
		ch <- reqStat{}
		return
	}

	duration := time_after.Sub(time_before).Seconds()
	ch <- reqStat{
		latency: float32(duration),
		successful: true,
	}
	return
}


func (g *grpcReq) generate_one_servers_load(conn *grpc.ClientConn, wg *sync.WaitGroup, ch chan reqStat){
	defer wg.Done()
	fullMethodName := fmt.Sprintf(
		"/%s.%s/%s",
		g.method.GetService().GetFile().GetPackage(),
		g.method.GetService().GetName(),
		g.method.GetName(),
	)
	req := dynamic.NewMessage(g.method.GetInputType())
	fields := req.GetKnownFields()
	for _ , field := range fields {
		if field.GetType() == dpb.FieldDescriptorProto_TYPE_STRING {
			req.SetFieldByName(field.GetName(), randString(2)) // Random Generation, to be fixed
		}
	}

	time_before := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(g.timeout) * time.Second)
	defer cancel()
	stream, err := conn.NewStream(
        ctx,
        &grpc.StreamDesc{
            ServerStreams: true,
            ClientStreams: false,
        },
        fullMethodName,
    )
	if err != nil {
		ch <- reqStat{}
		return
	}

	if err := stream.SendMsg(req); err != nil {
        ch <- reqStat{}
        return
    }
	if err := stream.CloseSend(); err != nil {
        ch <- reqStat{}
        return
    }
	resp := dynamic.NewMessage(g.method.GetOutputType())
	for {
        err := stream.RecvMsg(resp)
        if err == io.EOF{
            break
        }
        if err != nil {
			st, _ := status.FromError(err)
			if st.Code() == 4 { // Deadline exceeded
				break
			}
            ch <- reqStat{}
            return
        }
    }
	time_after := time.Now()

	duration := time_after.Sub(time_before).Seconds()
	ch <- reqStat{
		latency: float32(duration),
		successful: true,
	}
	return
}