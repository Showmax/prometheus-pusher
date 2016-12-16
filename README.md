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
```

## Example configuration
```
[config]
pushgateway_url = "http://localhost:9091" # Default
push_interval = 60                        # Default (in seconds)

[metric_name]
host = "localhost" # Default
path = "/metrics"  # Default
ssl = false        # Default
port = 9111

[second_metric]
port = 9112
```


## Logging
`prometheus-pusher` uses [logrus](https://github.com/Sirupsen/logrus/) with [sockrus](https://github.com/ShowMax/sockrus) hook for logging. There are some limitations we're aware of. PRs which enhance, but don't break, functionality are welcome.
