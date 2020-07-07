package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"strconv"

	log "github.com/Sirupsen/logrus"

	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ShowMax/go-fqdn"
)

type vzPusherConfig struct {
	PushGatewayURL string
	PushInterval   time.Duration
	Metrics        []metricConfig
}

type metricConfig struct {
	Name string
	URL  string
}

var (
	defConfigPath                = "/etc/prometheus-pusher/conf.d"
	defLogSocket                 = "/run/showmax/socket_to_amqp.sock"
	servicename                  = "prometheus-pusher"
	defaultHTTPClientTimeout     = 30 * time.Second
	splitSize                int = 1000
	pushGatewayUser              = os.Getenv("PUSHGATEWAY_USER")
	pushGatewayPassword          = os.Getenv("PUSHGATEWAY_PASSWORD")
	pushGatewayPasswordFile      = os.Getenv("PUSHGATEWAY_PASSWORD_FILE")
	pushInterval                 = os.Getenv("PUSH_INTERVAL")
	logger 					 	 = log.New()
)

const envCaCertFile = "PROM_CERT"

func init() {

	s := os.Getenv("SPLIT_SIZE")
	if len(s) > 0 {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Errorf("could not parse split size: %s", err)
			return
		}
		splitSize = int(i)
	}

}

func main() {
	log.Infoln("Starting Prometheus Pusher...")

	log.SetLevel(log.InfoLevel)
	if s := os.Getenv("LOGLEVEL"); len(s) > 0 {
		level, err := strconv.Atoi(s)
		level = 5
		if err != nil {
			log.Errorf("could not parse log level (%s): %s", s, err)
		} else {
			log.SetLevel(log.Level(level))
		}
	}
	log.Infoln("MYLOGLEVEL=" + log.GetLevel().String())

	dummy := flag.Bool("dummy", false,
		"Do not post the metrics, just print them to stdout")
	flag.Parse()

	log.SetOutput(os.Stdout)

	instanceLabel := getInstanceName()

	pusher, err := vzParseConfig()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal(fmt.Sprintf("Error parsing configuration: %s", err))
	}

	labels := getEnvVarsWithPrefix("log_field_", os.Environ())

	log.Infoln("instanceLabel=" + instanceLabel)
	log.Infoln("PushGateway=" + pusher.PushGatewayURL)
	log.Infoln("Push Interval=" + pusher.PushInterval.String())
	log.Infoln("Metric:")
	for _, metric := range pusher.Metrics {
		log.Infof("  Name=%s URL:%s", metric.Name, metric.URL)
	}
	log.Infoln("Labels:")
	for _, label := range labels {
		log.Infoln("  " + label)
	}

	log.Infoln("Starting threads to scrape metrics, then push them to the gateway")

	for _, metric := range pusher.Metrics {
		go getAndPush(metric, pusher.PushGatewayURL, instanceLabel, dummy, labels)
	}
	for _ = range time.Tick(pusher.PushInterval) {
		for _, metric := range pusher.Metrics {
			go getAndPush(metric, pusher.PushGatewayURL, instanceLabel, dummy, labels)
		}
	}
}

func getPullURLS(env []string) map[string]string {
	results := make(map[string]string)
	prefix := "PULL_URL_"
	for _, s := range env {
		if !strings.HasPrefix(s, prefix) {
			continue
		}
		s = s[len(prefix):]
		parts := strings.SplitN(s, "=", 2)
		log.Debugf("processing %s", s)
		if len(s) < 2 {
			continue
		}
		results[parts[0]] = parts[1]
	}
	return results
}

func vzParseConfig() (vzPusherConfig, error) {
	gateway := "http://localhost:9091"
	if s := os.Getenv("PUSHGATEWAY_URL"); len(s) > 0 {
		gateway = s
	}

	interval := time.Second * 60
	if pushInterval != "" {
		val, err := strconv.Atoi(pushInterval)
		if err != nil {
			log.Warningf("Error parsing pushInterval as an integer. Defaulting to 60 seconds")
		} else {
			interval = time.Second * time.Duration(val)
		}
	}
	conf := vzPusherConfig{
		PushGatewayURL: gateway,
		PushInterval:   interval,
		Metrics:        []metricConfig{},
	}

	endpoints := getPullURLS(os.Environ())
	for metric, s := range endpoints {
		conf.Metrics = append(conf.Metrics, metricConfig{
			Name: metric,
			URL:  s,
		})
	}

	return conf, nil
}

func getMetrics(metric metricConfig) []byte {
	log.WithFields(log.Fields{
		"metric_name": metric.Name,
		"metric_url":  metric.URL,
	}).Debug("Getting metrics")

	client := &http.Client{
		Timeout: defaultHTTPClientTimeout,
	}
	resp, err := client.Get(metric.URL)
	if err != nil {
		log.WithFields(log.Fields{
			"error":       err.Error(),
			"metric_name": metric.Name,
			"metric_url":  metric.URL,
		}).Error("Failed to get metrics.")
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error":       err.Error(),
			"metric_name": metric.Name,
			"metric_url":  metric.URL,
		}).Error("Failed to read response body.")
		return nil
	}
	return body
}

func removeTimestamps(metrics []byte) []byte {
	lines := strings.Split(string(metrics), "\n")
	newLines := []string{}
	for _, line := range lines {
		line = strings.Replace(line, " summary", " untyped", 1)
		if strings.HasPrefix(line, "#") {
			newLines = append(newLines, line)
			continue
		}
		if len(line) == 0 {
			continue // don't append empty lines
		}
		fields := strings.Fields(line)
		l := len(fields)
		if l <= 2 {
			newLines = append(newLines, line)
			continue
		}
		// if last two fields are numeric, last one is likely a timestamp, strip it
		if _, err := strconv.ParseFloat(fields[l-2], 64); err != nil {
			newLines = append(newLines, line)
			continue
		}
		if _, err := strconv.ParseInt(fields[l-1], 10, 64); err != nil {
			newLines = append(newLines, line)
			continue
		}
		newLines = append(newLines, strings.Join(fields[:l-1], " "))
	}
	return []byte(strings.Join(newLines, "\n"))
}

func pushMetrics(metricName string, pushgatewayURL string, instance string, metrics []byte, dummy *bool) {
	postURL := fmt.Sprintf("%s/metrics/job/%s/instance/%s", pushgatewayURL, metricName, instance)
	log.Debugf("Post url: %v", postURL)
	if *dummy {
		fmt.Println(string(metrics))
	} else {
		log.WithFields(log.Fields{
			"endpoint_url": postURL,
			"metric_name":  metricName,
		}).Debug("Pushing metrics.")

		// Get a new HTTP client each time in case the cert changed
		client := getHttpClient()

		// start pushing collected metrics in batches
		started := time.Now()
		n := 0
		for s, part := splitMetrics(metrics, splitSize); part != nil; part = s.Read() {
			// post one batch
			n += 1
			data := bytes.NewReader(part)
			request, err := http.NewRequest("POST", postURL, data)
			if err != nil {
				log.Errorf("could not create request: %s", err)
				return
			}
			if len(pushGatewayUser) > 0 {
				if pushGatewayPassword == "" && pushGatewayPasswordFile != "" {
					b, err := ioutil.ReadFile(pushGatewayPasswordFile)
					if err != nil {
						log.Errorf("could not read password file: %s", err)
						return
					}
					pushGatewayPassword = string(b)
				}
				request.SetBasicAuth(pushGatewayUser, pushGatewayPassword)
			}
			resp, err := client.Do(request)
			if err != nil {
				log.WithFields(log.Fields{
					"endpoint_url": postURL,
					"error":        err.Error(),
				}).Error("Failed to push metrics.")
				// dump the failed payload to disk to allow debugging
				ioutil.WriteFile("/tmp/metrics_error_dump.txt", part, 0600)
				return
			}
			if resp.StatusCode >= 300 {
				log.Errorf("got response code: %s(%d) ", resp.Status, resp.StatusCode)
				body, _ := ioutil.ReadAll(resp.Body)
				if body != nil {
					log.Errorf("got error body %s", string(body))
				}
				// dump the failed payload to disk to allow debugging
				ioutil.WriteFile(fmt.Sprintf("/tmp/metrics_error_response_%d.txt", len(body)), part, 0600)
			}
			log.Debugf("posted %d bytes", len(part))
			resp.Body.Close()
		}
		log.Debugf("took %s to post all metrics using %d calls", time.Since(started), n)
	}
}

func getAndPush(metric metricConfig, pushgatewayURL string, instance string, dummy *bool, labels []string) {
	if metrics := getMetrics(metric); metrics != nil {
		pushMetrics(metric.Name, pushgatewayURL, instance, removeTimestamps(addLabels(metrics, labels)), dummy)
	}
}

func getInstanceName() string {
	if len(os.Getenv("INSTANCE_NAME")) > 0 {
		return os.Getenv("INSTANCE_NAME")
	}
	return fqdn.Get()
}

// Get client used to call keycloak
func getHttpClient() *http.Client {
	// Get the cert
	caData := getCACert()
	if caData == nil || len(caData) == 0 {
		log.Debugln("Using HTTP client wihout cert ")
		return &http.Client{Timeout: defaultHTTPClientTimeout}
	}

	log.Debugln("Using cert with HTTP client ")

	// Create Transport object
	tr := &http.Transport{
		TLSClientConfig:       &tls.Config{RootCAs: rootCertPool(caData)},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{Transport: tr, Timeout: defaultHTTPClientTimeout}
	return client
}

// get the ca.crt from secret "<vz-env-name>-secret" in namespace "verrazzano-system"
func getCACert() []byte {

	fName := os.Getenv(envCaCertFile)

	if len(fName) == 0 {
		return nil
	}
	f, err := os.Open(fName)
	if err != nil {
		log.Error("Unable to open cert file " + fName)
		return nil
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Error("Error reading the cert file " + fName)
		return nil
	}
	return b
}

func rootCertPool(caData []byte) *x509.CertPool {
	if len(caData) == 0 {
		return nil
	}
	// if we have caData, use it
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caData)
	return certPool
}
