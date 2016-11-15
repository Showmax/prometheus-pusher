package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ShowMax/go-fqdn"
	"github.com/achun/tom-toml"
	"github.com/mgutz/logxi/v1"
)

type pusherConfig struct {
	PushGatewayURL string
	PushInterval   time.Duration
	Metrics        []metricConfig
}

type metricConfig struct {
	Name string
	URL  string
}

var logger log.Logger

func main() {
	path := flag.String("config", "/etc/prometheus-pusher/conf.d", "Config file or directory. If directory is specified then all files in the directory will be loaded.")
	dummy := flag.Bool("dummy", false, "Do not post the metrics, just print them to stdout")
	flag.Parse()

	logger = log.New("prometheus-pusher")

	pusher, err := parseConfig(*path)
	if err != nil {
		logger.Error("Error parsing configuration", err.Error())
	}

	hostname := fqdn.Get()
	logger.Info("Starting prometheus-pusher", "instance_name", hostname)

	for _, metric := range pusher.Metrics {
		go getAndPush(metric.Name, metric.URL, pusher.PushGatewayURL, hostname, dummy)
	}
	for _ = range time.Tick(pusher.PushInterval) {
		pusher, err := parseConfig(*path)
		if err != nil {
			logger.Error("Error parsing configuration", err.Error())
		}

		for _, metric := range pusher.Metrics {
			go getAndPush(metric.Name, metric.URL, pusher.PushGatewayURL, hostname, dummy)
		}
	}
}

func getConfigFiles(path string) []string {
	var files []string

	pathCheck, err := os.Open(path)
	if err != nil {
		logger.Fatal("Unable to open configuration file(s)", "error", err.Error())
	}

	pathInfo, err := pathCheck.Stat()
	if err != nil {
		logger.Fatal("Unable to stat configuration file(s)", "error", err.Error())
	}

	if pathInfo.IsDir() {
		dir, _ := pathCheck.Readdir(-1)
		for _, file := range dir {
			if strings.HasSuffix(file.Name(), ".toml") && (file.Mode().IsRegular()) {
				files = append(files, path+"/"+file.Name())
			}
		}
	} else {
		files = []string{path}
	}
	return files
}

func parseConfig(path string) (pusherConfig, error) {
	conf := pusherConfig{
		PushGatewayURL: "http://localhost:9091",
		PushInterval:   time.Duration(60 * time.Second),
		Metrics:        []metricConfig{},
	}

	for _, file := range getConfigFiles(path) {
		tomlFile, err := toml.LoadFile(file)
		if err != nil {
			return conf, err
		}

		metrics, _ := tomlFile.TableNames()
		for _, metric := range metrics {

			if metric == "config" {

				if tomlFile["config.pushgateway_url"].IsValue() {
					conf.PushGatewayURL = tomlFile["config.pushgateway_url"].String()
				}

				if tomlFile["config.push_interval"].IsValue() {
					interval := tomlFile["config.push_interval"].Int()
					conf.PushInterval = time.Duration(interval) * time.Second
				}

			} else {

				var port int
				host := "localhost"
				path := "/metrics"
				scheme := "http"

				if tomlFile[metric+".host"].IsValue() {
					host = tomlFile[metric+".host"].String()
				}

				if tomlFile[metric+".path"].IsValue() {
					path = tomlFile[metric+".path"].String()
				}

				if tomlFile[metric+".ssl"].IsValue() {
					if tomlFile[metric+".ssl"].Boolean() {
						scheme = "https"
					}
				}

				if tomlFile[metric+".port"].IsValue() {
					port = tomlFile[metric+".port"].Integer()
				}

				if port == 0 {
					logger.Fatal("Port is not defined", "config_section", metric)
				}

				conf.Metrics = append(conf.Metrics, metricConfig{
					Name: metric,
					URL:  fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path),
				})
			}
		}
	}

	return conf, nil
}

func getMetrics(metricURL string) []byte {
	logger.Info("Getting Node Exporter metrics", "url", metricURL)

	resp, err := http.Get(metricURL)
	if err != nil {
		logger.Error(err.Error(), "error", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err.Error(), "error", err)
		return nil
	}
	return body
}

func pushMetrics(metricName string, pushgatewayURL string, instance string, metrics []byte, dummy *bool) {
	postURL := fmt.Sprintf("%s/metrics/job/%s/instance/%s", pushgatewayURL, metricName, instance)
	if *dummy {
		fmt.Println(string(metrics))
	} else {
		logger.Info("Pushing Node exporter metrics", "endpoint", postURL)

		data := bytes.NewReader(metrics)
		resp, err := http.Post(postURL, "text/plain", data)
		if err != nil {
			logger.Error(err.Error(), "error", err)
			return
		}
		defer resp.Body.Close() // FIXME: no need to close on error?
	}
}

func addTimestamps(metrics []byte) (timestampedMetrics []byte) {
	/* if the metrics are missing timestams and the pusher stops sending
	 * for a while, pushgateway will report the same values every time
	 * prometheus collects it, which results into flat non-zero values which
	 * are confusing */

	/* Note that this is not optimal and the producers of the data should
	 * be appending timestamps by themselves. And we will honor them. But
	 * most of the exporters do not do that unfortunately. */

	currentTime := strconv.Itoa(int(time.Now().UnixNano() / int64(time.Millisecond)))
	lines := strings.Split(string(metrics), "\n")
	for i := 0; i < len(lines); i++ {
		// skip comments and empty lines
		if (len(lines[i]) == 0) || (lines[i][0] == '#') {
			continue
		}
		// find closing curly bracket - metrics that have labels
		lastCBIndex := strings.LastIndex(lines[i], "}")
		// some metrics do not have labels and curly braces
		if lastCBIndex == -1 {
			lastCBIndex = 0
		}
		// we'll have "} <value>" for untimestamped but labelled metrics
		// and "<metric_name> <value>" for untimestamped and unlabelled metrics
		dataFields := strings.Fields(lines[i][lastCBIndex:])
		// skip lines that (probably) already have timestamps
		if len(dataFields) > 2 {
			continue
		}
		lines[i] += " " + currentTime
	}
	timestampedMetrics = []byte(strings.Join(lines, "\n"))
	return
}

func getAndPush(metricName string, metricURL string, pushgatewayURL string, instance string, dummy *bool) {
	if metrics := getMetrics(metricURL); metrics != nil {
		pushMetrics(metricName, pushgatewayURL, instance, addTimestamps(metrics), dummy)
	}
}
