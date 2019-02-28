package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	fqdn "github.com/Showmax/go-fqdn"
	"github.com/Showmax/sockrus"
	"github.com/sirupsen/logrus"
)

// global vars
//
var (
	cfgPath           string
	dummy             bool
	verbose           uint
	hostname          string
	httpClientTimeout time.Duration
	logger            *logrus.Entry
	defaultConfPath   = "/etc/prometheus-pusher/conf.d"
	defaultLogSocket  = "/run/showmax/socket_to_amqp.sock"
	serviceName       = "prometheus-pusher"
	version           string
	versionFlag       bool
	printMutex        = &sync.Mutex{}
)

func init() {
	// parse arguments
	flag.StringVar(&cfgPath, "config", defaultConfPath,
		"Config file or directory. If directory is specified then all "+
			"files in the directory will be loaded.")
	flag.BoolVar(&dummy, "dummy", false,
		"Do not post the metrics, just print them to stdout")
	flag.UintVar(&verbose, "verbosity", 1, "Set logging verbosity.")
	flag.DurationVar(&httpClientTimeout, "http-timeout", 30*time.Second, "Timeout for HTTP requests")
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")
	flag.Parse()

	if versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	// set log level
	var logLevel logrus.Level
	switch verbose {
	case 0:
		logLevel = logrus.ErrorLevel
	case 1:
		logLevel = logrus.InfoLevel
	default:
		logLevel = logrus.DebugLevel
	}

	hostname = fqdn.Get()

	// create logger instance
	_, logger = sockrus.NewSockrus(sockrus.Config{
		LogLevel:       logLevel,
		Service:        serviceName,
		SocketAddr:     defaultLogSocket,
		SocketProtocol: "unix",
	})
}

func main() {
	logger.Info("Starting prometheus-pusher")

	// read config files
	cfgData, err := concatConfigFiles(cfgPath)
	if err != nil {
		logger.Fatalf("Failed to read config files - %s", err.Error())
	}

	// parse config data
	pusherCfg, err := parseConfig(cfgData)
	if err != nil {
		logger.Fatalf("Failed to parse config data - %s", err.Error())
	}

	// prepare global route map if there is any
	var globalRouteMap *routeMap
	if pusherCfg.defaultRoute != "" && pusherCfg.routeMap != "" {
		globalRouteMap = newRouteMap(pusherCfg.routeMap, pusherCfg.defaultRoute)
	}

	// spawn resources
	resources := createResources(pusherCfg, globalRouteMap)

	// handle signals for clean shutdown
	signal.Notify(resources.sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for s := range resources.sig {
			logger.Infof("Received %s signal, will shut down", s)
			resources.shutdown()
			return
		}
	}()

	resources.process(pusherCfg)

	for {
		select {
		case <-resources.run():
			resources.process(pusherCfg)
		case <-resources.stop():
			logger.Info("Resources processing stopped")
			os.Exit(0)
		}
	}
}
