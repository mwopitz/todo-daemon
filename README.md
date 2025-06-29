# To-do Daemon

This repository contains a proof of concept for a [daemon][daemon]-like service
created with [Go][go] that provides both a [command-line interface][cli] (CLI)
as well as a [REST][rest] API. It's designed to run user scoped (not as `root`)
on a desktop client.

The To-do Daemon does not behave like a traditional [SysV daemon][sysv-daemon];
it doesn't call `fork` to detach itself from the parent process. Instead, the
To-do Daemon behaves more like a [new-style daemon][systemd-daemon], as defined
by systemd.

[cli]: https://en.wikipedia.org/wiki/Command-line_interface
[daemon]: https://en.wikipedia.org/wiki/Daemon_(computing)
[go]: https://go.dev/
[rest]: https://en.wikipedia.org/wiki/REST
[sysv-daemon]: https://www.freedesktop.org/software/systemd/man/latest/daemon.html#New-Style%20Daemons
[systemd-daemon]: https://www.freedesktop.org/software/systemd/man/latest/daemon.html#New-Style%20Daemons

## Architecture

![Architecture diagram](docs/architecure.svg)

The To-do Daemon has one primary process, the daemon process, and zero or more
secondary command processes.

The daemon process runs two servers:

* An HTTP server that provides a REST API to other applications.
* A [gRPC](https://grpc.io/) server that is used for internal communication
  between the daemon process and the command processes.

The command processes provide a command-line interface (CLI) for interacting
with the daemon process.

## Getting started

1.  [Install Go](https://go.dev/doc/install)
1.  Build the go-daemon:
    
    ```sh
    go build
    ```
1.  Start it:
    
    ```sh
    ./go-daemon run
    ```

## Compiling the gRPC components

1.  [Install the protobuf compiler](https://protobuf.dev/installation/).
    
    For example, on Linux:
    
    ```sh
    pb_rel=https://github.com/protocolbuffers/protobuf/releases
    pb_ver=31.1
    curl -LO "$pb_rel/download/v$pb_ver/protoc-$pb_ver-linux-x86_64.zip"
    unzip "protoc-$pb_ver-linux-x86_64.zip" -d "$HOME/.local"
    ```
1.  Install the Go plugins for the protobuf compiler:
    
    ```sh
    go tool install
    ```
1.  Execute the compiler:
    
    ```sh
    protoc --proto_path=./api/proto \
      --go_out module=github.com/mwopitz/todo-daemon:. \
      --go-grpc_out module=github.com/mwopitz/todo-daemon:. \
      ./api/proto/todo_daemon.proto
    ```
