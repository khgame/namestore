## Installation

### Prerequisites

- Go 1.18 or higher (for generics support)

### Using go get

```bash
go get github.com/khicago/namestore
```

### Verifying Installation

Create a simple test file to verify the installation:

```go
// test.go
package main

import (
    "context"
    "fmt"

    "github.com/khicago/namestore"
)

func main() {
    client := namestore.New[string]("test", "demo")
    client.Set(context.Background(), "hello", []byte("world"), 0)

    data, _ := client.Get(context.Background(), "hello")
    fmt.Println(string(data)) // Output: world
}
```

Run it:

```bash
go run test.go
```

If you see `world` printed, the installation is successful!