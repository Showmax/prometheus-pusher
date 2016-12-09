# Socket hook for [logrus](https://github.com/Sirupsen/logrus) <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:" />

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](http://godoc.org/github.com/ShowMax/sockrus)

Use this hook to send the logs over UDP, TCP or Unix socket.

Output format is JSON, formatted for Logstash/ElasticSearch.

## Usage

```go
package main

import (
        "github.com/Sirupsen/logrus"
        "github.com/ShowMax/sockrus"
)

func main() {
        log := logrus.New()
        hook, err := sockrus.NewHook("unixpacket", "/tmp/log.sock")
        if err != nil {
                log.Fatal(err)
        }
        log.Hooks.Add(hook)
        ctx := log.WithFields(logrus.Fields{
          "method": "main",
        })
        ...
        ctx.Info("Hello World!")
}
```

This is how it will look like:

```json
{
  "@timestamp": "2016-04-15T12:49:36Z",
  "@version": 1,
  "level": "info",
  "message": "Hello World!",
  "method": "main"
}
```
