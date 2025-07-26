module github.com/mwopitz/todo-daemon

go 1.24.0

require (
	github.com/gofrs/flock v0.12.1
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2
	github.com/urfave/cli/v3 v3.3.8
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.6
)

require (
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250721164621-a45f3dfb1074 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.5.1 // indirect
)

tool (
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
)
