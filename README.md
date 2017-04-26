# prometheus-pusher
> `prometheus-pusher` aggregates [Prometheus](https://prometheus.io/) metrics from different endpoints and pushes them to [pushgateway](https://github.com/prometheus/pushgateway)

## Installation
```
$ go get github.com/ShowMax/prometheus-pusher
```

## Usage
```
  -config string
    	Config file or directory. If directory is specified then all files in the directory will be loaded. (default "/etc/prometheus-pusher/conf.d")
  -dummy
    	Do not post the metrics, just print them to stdout
  -http-timeout duration
    	Timeout for HTTP requests (default 30s)
  -verbosity uint
    	Set logging verbosity. (default 1)

```

## Configuration

_TODO: write some info about how the dynamic routing works_

### Example config file
```
[config]
pushgateway_url = "http://localhost:9091" # Default
push_interval = 60                        # Default (in seconds)

[resource1]
host = "localhost" # Default
path = "/metrics"  # Default
ssl = false        # Default
port = 9111

[resource2]
pushgateway_url = "http://%s:9091/"
default_route = "prometheus"
route_map = "path/to/route.map"
port = 9112
```

### Example route map
```
go_ prometheus1.somedomain.com
go_debug_ prometheus2.somedomain.com
mem_ prometheus1.somedomain.com
```


## Logging
`prometheus-pusher` uses [logrus](https://github.com/Sirupsen/logrus/) with [sockrus](https://github.com/ShowMax/sockrus) wrapper for logging.

## Contributing
PRs which enhance, but don't break functionality are welcome. Tests are requires whenever possible.
