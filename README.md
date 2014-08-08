go-git-http
===========

A Smart Git Http server library in Go (golang)

### Example

```go
package main

import (
    "log"
    "net/http"

    "github.com/AaronO/go-git-http"
)

func main() {
    // Get git handler to serve a directory of repos
    git := githttp.New("/Users/aaron/git")

    // Attach handler to http server
    http.Handle("/", git)

    // Start HTTP server
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
```
