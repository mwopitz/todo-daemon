# Go Daemon

This repository contains a proof of concept for a [daemon][daemon]-like service
created with [Go][go] that provides both a [command-line interface][cli] (CLI)
as well as a [REST][rest] API. It's designed to run user scoped (not as `root`) on
a desktop client.

[cli]: https://en.wikipedia.org/wiki/Command-line_interface
[daemon]: https://en.wikipedia.org/wiki/Daemon_(computing)
[go]: https://go.dev/
[rest]: https://en.wikipedia.org/wiki/REST

## Architecture

The Go Daemon has one primary process, the daemon process, and zero or more
secondary command processes.

The daemon process runs two servers:

* An HTTP server that provides a REST API to other applications.
* A [gRPC](https://grpc.io/) server that is used for internal communication
  between the daemon process and the command processes.

## Getting started

1.  [Install Go](https://go.dev/doc/install)
2.  Build the go-daemon:
    
    ```shs
    go build
    ```
3.  Start it:
    
    ```sh
    ./go-daemon run
    ```

## Compiling the gRPC components

1.  [Install the protobuf compiler](https://protobuf.dev/installation/)
2.  Install the Go plugins for the protobuf compiler:
    
    ```sh
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    ```
3.  Execute the compiler:
    
    ```sh
    protoc --proto_path=./api/protos --go_out module=github.com/mwopitz/go-daemon:. --go-grpc_out module=github.com/mwopitz/go-daemon:. daemon.proto
    ```
