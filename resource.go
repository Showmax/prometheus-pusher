package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

type resources struct {
	wg     *sync.WaitGroup
	ticker *time.Ticker
	sig    chan os.Signal
	exit   chan struct{}
	rs     map[string]*resource
}

func createResources(cfg *pusherConfig, grm *routeMap) *resources {
	rs := make(map[string]*resource)

	for name := range cfg.resources {
		rs[name] = newResource(name, cfg, grm)
	}

	return &resources{
		rs:     rs,
		ticker: time.NewTicker(cfg.pushInterval),
		sig:    make(chan os.Signal, 1),
		exit:   make(chan struct{}, 1),
		wg:     &sync.WaitGroup{},
	}
}

func (rs *resources) process() {
	for _, r := range rs.rs {
		rs.wg.Add(1)
		go r.getAndPush(rs.wg)
	}
	rs.wg.Wait()
}

func (rs *resources) run() <-chan time.Time {
	return rs.ticker.C
}

func (rs *resources) stop() <-chan struct{} {
	return rs.exit
}

func (rs *resources) shutdown() {
	rs.ticker.Stop()
	rs.exit <- struct{}{}
}

type resource struct {
	name           string
	pushGatewayURL string
	resURL         string
	routes         *routeMap
	httpClient     *http.Client
}

// creates new instance of resource
//
func newResource(name string, cfg *pusherConfig, grm *routeMap) *resource {
	var pushgatewayURL string
	if cfg.resources[name].pushGatewayURL != "" {
		pushgatewayURL = cfg.resources[name].pushGatewayURL
	} else if cfg.pushGatewayURL != "" {
		pushgatewayURL = cfg.pushGatewayURL
	} else {
		logger.Fatalf("No pushgateway_url derived from config for resource '%s'", name)
	}

	defaultRoute := cfg.defaultRoute
	if cfg.resources[name].defaultRoute != "" {
		defaultRoute = cfg.resources[name].defaultRoute
	}

	var rm *routeMap
	if cfg.resources[name].routeMap != "" {
		rm = newRouteMap(cfg.resources[name].routeMap, defaultRoute)
	} else {
		rm = newRouteMap(cfg.routeMap, defaultRoute)
	}

	return &resource{
		name:           name,
		pushGatewayURL: pushgatewayURL,
		resURL:         cfg.resources[name].resURL,
		routes:         rm,
		httpClient: &http.Client{
			Timeout: httpClientTimeout,
		},
	}
}

// retrieve metrics of a resource
//
func (r *resource) getMetrics() []byte {
	logger.WithFields(logrus.Fields{
		"resource_name": r.name,
		"resource_url":  r.resURL,
	}).Debug("Getting metrics")

	resp, err := r.httpClient.Get(r.resURL)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":         err.Error(),
			"resource_name": r.name,
			"resource_url":  r.resURL,
		}).Error("Failed to get metrics.")
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":         err.Error(),
			"resource_name": r.name,
			"resource_url":  r.resURL,
		}).Error("Failed to read response body.")
		return nil
	}
	return body
}

// push metrics into given destination
//
func (r *resource) pushMetrics(metrics []byte, dst string, wg *sync.WaitGroup) {
	defer wg.Done()

	postURL := fmt.Sprintf(r.pushGatewayURL, dst) + fmt.Sprintf("/job/%s/instance/%s", r.name, hostname)
	if dummy {
		fmt.Println(string(metrics))
		return
	}

	logger.WithFields(logrus.Fields{
		"endpoint_url":  postURL,
		"resource_name": r.name,
	}).Debug("Pushing metrics.")

	data := bytes.NewReader(metrics)
	resp, err := r.httpClient.Post(postURL, "text/plain", data)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"endpoint_url": postURL,
			"error":        err.Error(),
		}).Error("Failed to push metrics.")
		return
	}
	resp.Body.Close()
	return
}

// gets metrics, does inverse-multiplexing on the data
// by metrics names and route definitions and pushes the
// data into promethei
//
func (r *resource) getAndPush(wgImux *sync.WaitGroup) {
	defer wgImux.Done()
	wgPush := &sync.WaitGroup{}
	if metricsBytes := r.getMetrics(); metricsBytes != nil {
		m := newMetrics(metricsBytes)
		for dst, body := range m.imux(r.routes) {
			wgPush.Add(1)
			go r.pushMetrics(body, dst, wgPush)
		}
	}
}
