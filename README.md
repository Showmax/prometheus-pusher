Prometheus Pusher Image
pipeline status

Prometheus Pusher Docker image customized for Verrazzano.

EnvironmentVariable Name	Description
PUSHGATEWAY_URL	Prometheus Pushgateway's URL
PUSHGATEWAY_USER	Basic Auth username to push to the pushgateway
PUSHGATEWAY_PASSWORD	Basic Auth password to push to the pushgateway
PULL_HOSTS	Hostname of the source to pull the metrics from (default: localhost)
PULL_PORTS	Port number of the source to pull the metrics from (default: 9102)
METRIC_PATHS	Path specifier to fetch the metrics (default /metrics)
SPLIT_SIZE	integer that says how many metrics to push to prometheus at a time
LOGLEVEL	integer that set the loglevel. 0-panic, 1-fatal, 2-error, 3-warn, 4-info, 5-debug.
INSTANCE_NAME	Prometheus the instance label that helps uniquely identifies the job (defaults to pusher FQDN)
PUSH_INTERVAL	Push interval in seconds
Note - only basic auth is supported for pushing to Prometheus Push Gateway.
