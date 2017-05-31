# prometheus-pusher
> `prometheus-pusher` aggregates [Prometheus](https://prometheus.io/) metrics from different endpoints and pushes them to [pushgateway](https://github.com/prometheus/pushgateway)

## Architecture
IMG

`prometheus-pusher` fetches metrics data from configured resources in specified interval and does inverse multiplexing on each metric, where destination for each one is decided by the prefix of metric name specified in route map file.

## Installation
```
$ go get -u github.com/ShowMax/prometheus-pusher
```

## Usage
See `-help`.

## Configuration

- `push_interval`
  - Valid sections: `[config]`
  - Default: `60`
  - interval of scraping in seconds
- `pushgateway_url`
  - Valid sections: `[config]`, `[<resource>]`
  - Default: ``
  - URL of the pushgateway. If you want to use inverse multiplexing by metric name, you have to include `%s` in the string. That place will be used by the resolved route destination either from route map file or default_route. Can be configured both in `[config]` section and separately for each resource.
- `route_map`
  - Valid sections: `[config]`, `[<resource>]`
  - Default: n/a
  - Absolute path to the route map file. Can be configured both in `[config]` section and separately for each resource. **Mandatory when using inverse multiplexing**
- `default_route`
  - Valid sections: `[config]`, `[<resource>]`
  - Default: n/a
  - Default route for metrics with unnamed prefixes. Can include multiple strings separated by `,` (without spaces). Metrics will be pushed to all the named destinations. Can be configured both in `[config]` section and separately for each resource. **Mandatory when using inverse multiplexing**
- `host`
  - Valid sections: `[<resource>]`
  - Default: `localhost`
  - Hostname of the resource
- `port` **mandatory option**
  - Valid sections: `[<resource>]`
  - Default: `0`
  - Port of the resource.
- `path`
  - Valid sections: `[<resource>]`
  - Default: `/metrics`
  - The path part of the resource URL.
- `ssl`
  - Valid sections: `[<resource>]`
  - Default: `false`
  - Whether the endpoint is encrypted (HTTPS).


### Example config

```
[config]
push_interval = 60                 # Default (in seconds)
pushgateway_url = "http://%s.somedomain.com:9092" # Default
route_map = "/path/to/route1.map"
default_route = "prometheus1,prometheus2"

[resource1]
host = "localhost" # Default
path = "/metrics"  # Default
ssl = false        # Default
port = 9111

[resource2]
pushgateway_url = "http://%s.somedomain.com:9091/"
route_map = "/path/to/route2.map"
default_route = "prometheus"
port = 9112
```

### Example route map
```
go_ prometheus1
go_debug_ prometheus
mem_ prometheus1
```


## Logging
`prometheus-pusher` uses [logrus](https://github.com/Sirupsen/logrus/) with [sockrus](https://github.com/ShowMax/sockrus) wrapper for logging.

## Contributing
PRs which enhance, but don't break functionality are welcome. Tests are requires whenever possible.
