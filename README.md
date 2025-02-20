## Introduction
Small package for throttling net.Listener connections

## Features

- Setting a global bandwidth limit for all connections
- Setting an individual connection bandwidth limit for all connections
- Applying changes of the limits to existing connections in runtime

## Usage

To use the Throttling library in your Go project, import it as follows:

```go
import "github.com/mlshvsk/netlistener"
```

To create a throttled net.Listener, you can use the `netlistener.NewThrottledListener` function. Here's an example:

```go
package main

import (
    "fmt"
    "net"
    "time"

    "github.com/mlshvsk/netlistener"
)

func main() {
    // Create a regular net.Listener
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        fmt.Println("Failed to create listener:", err)
        return
    }

    // Create a throttled net.Listener with a global bandwidth limit of 1MB/s and perConn bandwith to 500kB/s
    throttledLn := netlistener.NewThrottledListener(ln, 1024*1024, 512*1024/2)

    go func() {
        time.Sleep(10 * time.Second)
        throttledLn.SetLimits(limit, limit) // Update limits dynamically
    }()

    // Start accepting connections
    for {
        conn, err := throttledLn.Accept()
        if err != nil {
            fmt.Println("Failed to accept connection:", err)
            return
        }

        // Handle the connection
        go handleConnection(conn)
    }

}

func handleConnection(conn net.Conn) {
    //do something
}
```

## Testing

To run tests:

```go
go test .
```