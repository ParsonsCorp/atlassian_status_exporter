package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	address       = flag.String("svc.address", "0.0.0.0", "assign an IP address for this service to listen on")
	debug         = flag.Bool("debug", false, "enable the service debug output")
	enableColLogs = flag.Bool("enable-color-logs", false, "when developing in debug mode, prettier to set this for visual colors")
	help          = flag.Bool("help", false, "help will display this helpful dialog output")
	port          = flag.String("svc.port", "9997", "set the port that this service will listen on")
	protocal      = flag.String("app.protocal", "https", "set the protocol used to interact with the application")
	scrapeTimeout = flag.Int("svc.timeout", 10, "set the timeout this service will allow to check the url. by default prometheus scrape timeout is 10 second. if you know the scrape may take longer, this can be adjusted.")
	url           = flag.String("app.url", "", "REQUIRED: provide the application url to be monitored (ie. <bitbucket|confluence|jira>.domain.com)")

	baseURL      string
	disCol       = true
	namespace    = "atlassian_status"
	usageMessage = "The Atlassian Status Exporter is used to reach out and collect the info from\n" +
		"the /status page, then turn that into a collectable metric.\n" +
		"\nUsage: " + namespace + "_exporter [Arguments...]\n" +
		"\nArguments:\n"
)

var client = http.Client{
	Timeout: time.Duration(*scrapeTimeout) * time.Second,
}

// usage is used to display this binaries usage description and then exit the program.
var usage = func() {
	fmt.Println(usageMessage)
	flag.PrintDefaults()
	os.Exit(0)
}

// statusEndpoint defines the expected response json structure found at /status.
type statusEndpoint struct {
	State string `json:"state"`
}

// statusCollector is the structure of our prometheus collector containing it descriptors.
type statusCollector struct {
	scrapeUpMetric     *prometheus.Desc
	stateMetric        *prometheus.Desc
	stateRuntimeMetric *prometheus.Desc
}

// newStatusCollector is the constructor for our collector used to initialize the metrics.
func newStatusCollector() *statusCollector {
	return &statusCollector{
		scrapeUpMetric: prometheus.NewDesc(
			namespace+"_scrape_url_up",
			"metric shows the status of the connection to the atlassian application endpoint",
			[]string{
				"httpcode",
				"url",
			},
			nil,
		),
		stateMetric: prometheus.NewDesc(
			namespace+"_state",
			"metric returns the state of the monitored atlassian application",
			[]string{
				"state",
				"httpcode",
				"description",
				"url",
			},
			nil,
		),
		stateRuntimeMetric: prometheus.NewDesc(
			namespace+"_collect_duration_seconds",
			"metric keeps track of how long the exporter took to collect metrics",
			[]string{
				"url",
			},
			nil,
		),
	}
}

// Describe is required by prometheus to add out metrics to the default prometheus desc channel
func (collector *statusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.scrapeUpMetric
	ch <- collector.stateMetric
	ch <- collector.stateRuntimeMetric
}

// Collect implements required collect function for all prometheus collectors
func (collector *statusCollector) Collect(ch chan<- prometheus.Metric) {

	startTime := time.Now()

	log.Debug("get url ", baseURL)
	resp, err := client.Get(baseURL)
	if err != nil {
		log.Warn("client.Get base URL returned an error: ", err)
		ch <- prometheus.MustNewConstMetric(collector.scrapeUpMetric, prometheus.GaugeValue, 0, "", *url)
		return
	}
	defer resp.Body.Close()

	log.Debug("set scrape_url_up metric")
	ch <- prometheus.MustNewConstMetric(collector.scrapeUpMetric, prometheus.GaugeValue, 1, strconv.Itoa(resp.StatusCode), *url)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("ioutil.ReadAll returned an error: ", err)
	}

	// remove the trailing \n and any whitespace before checking if we got an empty body
	if strings.TrimSuffix(strings.Replace(string(body), " ", "", -1), "\n") == "" {
		log.Debug(*url, " response entity empty")
		ch <- prometheus.MustNewConstMetric(collector.stateMetric, prometheus.GaugeValue, stateMetricValue(""), "", strconv.Itoa(resp.StatusCode), stateDesc(""), *url)
		return
	}

	m := unmarshal(body)
	log.Debug("the returned body map: ", m)

	log.Debug("set state metric")
	ch <- prometheus.MustNewConstMetric(
		collector.stateMetric,
		prometheus.GaugeValue,
		stateMetricValue(m.State),
		m.State,
		strconv.Itoa(resp.StatusCode),
		stateDesc(m.State),
		*url,
	)

	finishTime := time.Now()
	elapsedTime := finishTime.Sub(startTime)
	log.Debug("set collect_duration_seconds metric")
	ch <- prometheus.MustNewConstMetric(collector.stateRuntimeMetric, prometheus.GaugeValue, elapsedTime.Seconds(), *url)
	log.Debug("collect finished")
}

// unmarshal takes a http body btye slice and unmarshals it into the /status structure.
func unmarshal(body []byte) statusEndpoint {

	log.Debug("create the json map for endpoint")
	var m statusEndpoint

	log.Debug("unmarshal (turn unicode back into a string) request body into map structure")
	err := json.Unmarshal(body, &m)
	if err != nil {
		log.Error("error Unmarshalling: ", err)
		log.Info("Problem unmarshalling the following string: ", string(body))
	}

	return m
}

// stateMetricValue takes in the state response entity and returns a code we'll use for the metric value.
func stateMetricValue(state string) float64 {
	switch state {
	case "RUNNING":
		return 0
	case "ERROR":
		return 1
	case "STARTING":
		return 2
	case "STOPPING":
		return 3
	case "FIRST_RUN":
		return 4
	case "":
		return 5
	default:
		return 6
	}
}

// stateDesc takes in the state response entity and returns the description that matches.
func stateDesc(state string) string {
	switch state {
	case "RUNNING":
		return "Running normally"
	case "ERROR":
		return "An error state"
	case "STARTING":
		return "Application is starting"
	case "STOPPING":
		return "Application is stopping"
	case "FIRST_RUN":
		return "Application is running for the first time and has not yet been configured"
	case "":
		return "Application failed to start up in an unexpected way (the web application failed to deploy)"
	default:
		return "Unknown Response, go look at the Atlassian Application"
	}
}

func main() {
	flag.Parse()

	// check if help has been passed
	if *help {
		usage()
	}

	// check for required arguments
	if *url == "" {
		fmt.Printf("-app.url must be provided\n\n")
		usage()
	}

	// adjust the logrus logger. Disable colors by default (adjustable with enable-color-logs argument). Enable full time-stamps by default
	if *enableColLogs {
		disCol = false
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		DisableColors: disCol,
	})

	// check for debug argument, adjust if set
	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("set log level: debug")
	}

	log.Debug("set status url from given argument: ", *url)
	baseURL = *protocal + "://" + *url + "/status"

	// Create a new instance of the statusCollector and then
	// register it with the prometheus client.
	exporter := newStatusCollector()
	prometheus.MustRegister(exporter)

	srv := http.Server{
		Addr: *address + ":" + *port,
	}

	// This will run metrics endpoint by the prometheus http handler.
	http.Handle("/metrics", promhttp.Handler())

	// make a channel to wait for os signals
	ch := make(chan os.Signal, 1)
	// define what signals we are going to wait for
	signal.Notify(ch, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// start the http server in a go routine
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal("ListenAndServe Error:", err)
		}
	}()

	log.Info("serving "+namespace+"_exporter on ", *address+":"+*port)

	// block waiting for channel. serivce will stay running waiting for a defined signal, once the signal comes, it will continue.
	s := <-ch
	log.Info("Got SIGNAL:", s)

	log.Debug("close channel")
	close(ch)

	log.Info("shutdown http server")
	err := srv.Shutdown(context.Background())
	if err != nil {
		// Error from closing listeners, or context timeout
		log.Fatal("Shutdown error:", err)
	}

	log.Info("Supposed Graceful Shutdown")

}
