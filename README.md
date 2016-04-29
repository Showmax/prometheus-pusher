# prometheus-pusher
> `prometheus-pusher` aggregates [Prometheus](https://prometheus.io/) metrics from different endpoints and push them to [pushgateway](https://github.com/prometheus/pushgateway)

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
`prometheus-pusher` uses [logxi](https://github.com/mgutz/logxi#configuration) for logging.

The format may be set via LOGXI_FORMAT environment variable. Valid values are "happy", "text", "JSON", "LTSV".

By default logxi logs entries whose level is LevelWarn or above when using a terminal.

To change the level, use LOGXI environment variable. Valid values are "DBG", "INF", "WRN", "ERR", "FTL".

```
LOGXI=* prometheus-pusher
# the above statement is equivalent to this
LOGXI=*=DBG prometheus-pusher
# now using json format instead
LOGXI_FORMAT=JSON prometheus-pusher
```
