<img width="1024" height="1024" alt="lgen" src="https://github.com/user-attachments/assets/5b24a0a3-a221-4dfe-8075-5ad19e5d199b" />

# lgen
Benchmarking tool for gRPC & HTTP protocols-based web applications
> Currently lgen is in the PoC period, not production ready (you can use it, as long as, you are aware of it's limitations)

## features
- lgen utilizes `cobra` package to have a CLI toolset to generate load using your specific configurations
### gRPC
- Currently, lgen dynamically parses the `proto` file and generate random data based on the data type for each field
- lgen supports `Unary`, `Client Streaming` and `Server Streaming` ![gRPC operations](https://grpc.io/docs/what-is-grpc/core-concepts/).
#### Unary gRPC
`go run main.go grpc --proto /home/ahmed-kamal/Downloads/services.proto --destination localhost:50051 --reqn 100000 --tarm sendmessage`
#### Client Streaming gRPC
Client streaming is tested by file uploading over chunks, file size can be changed, check `go run main.go grpc help` for details.
`go run main.go grpc --proto /home/ahmed-kamal/Downloads/services.proto --destination localhost:50051 --reqn 100000 --tarm uploadfile`
#### Server Streaming gRPC
The main metric that should be cared for during Server streaming is the `number of sent events`
`go run main.go grpc --proto /home/ahmed-kamal/Downloads/services.proto --destination localhost:50051 --reqn 10000 --tarm getnotifications --timeout 10`

##### Output example
<img width="1857" height="279" alt="Screenshot from 2025-11-30 01-12-58" src="https://github.com/user-attachments/assets/f44e2888-d9d2-4b5c-9fc2-8f477adeb2b8" />

### HTTP
#### Unary (Currently POST requests are the only supported HTTP request type for now)
for POST requests, the request body should be specified.
`go run main.go http --destination "http://localhost:8000/SendMessage" --reqn 100 --method POST --reqb_path test-scripts/body.json`
#### Client Streaming (Simulated using File upload)
`go run main.go http cs --destination "http://localhost:8000/upload" --reqn 100 --size 2147483648`
#### Server-Sent-Events
`go run main.go http sse --destination "http://localhost:8000/GetNotifications?user_id=u1" --reqn 100`

## Contribution
lgen is and will be always OSS, lgen is always open to OS contribution, feel free to open PR, add issue or even discuss detials within github discussions (slack/discord can considered if the community became bigger).
