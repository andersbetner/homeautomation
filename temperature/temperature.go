package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	mqttHost          string
	agent             *ag.Agent
	temperatureURL    string
	updateInterval    int // minutes default=15
	promUpdateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_sensor_updates_total",
			Help: "How many times this item has been updated.",
		},
		[]string{"status", "type", "topic"},
	)
	promTemperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_temperature",
			Help: "Temperature.",
		}, []string{"topic"},
	)
)

// update gets the outdoor temperature from temperatur.nu
func update() {
	client := &http.Client{
		CheckRedirect: nil,
		Timeout:       time.Second * 10,
	}
	request, _ := http.NewRequest("GET", temperatureURL, nil)
	resp, err := client.Do(request)
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "temperature", "request").Inc()
		log.WithField("error", err).Error("Error initiating request")

		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "temperature", "parse").Inc()
		log.WithField("error", err).Error("Error getting request body")

		return
	}
	temperature, err := strconv.ParseFloat(strings.TrimSpace(string(body)), 64)
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "temperature", "parse").Inc()
		log.WithField("error", string(body)).Error("Error malformed float value", string(body))

		return
	}
	err = agent.Publish("temperature/outdoor", true, fmt.Sprintf("%v", temperature))
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "temperature", "publish").Inc()
		log.WithField("error", err).Error("Error publishing mqtt")

		return
	}

	promUpdateCounter.WithLabelValues("200", "temperature", "outdoor").Inc()
	promTemperature.WithLabelValues("outdoor").Set(temperature)
	log.WithField("temperature", temperature).Debug("Outdoor temp")
}

func init() {
	log.SetLevel(log.DebugLevel)
	prometheus.MustRegister(promUpdateCounter)
	prometheus.MustRegister(promTemperature)

	exit := false
	flag.StringVar(&mqttHost, "mqtthost", "", "address and port for mqtt server eg tcp://example.com:1883")
	flag.StringVar(&temperatureURL, "url", "", "url for temperature server eg http://example.com/temp.txt")
	flag.IntVar(&updateInterval, "interval", 15, "integer > 0")
	flag.Parse()
	if mqttHost == "" {
		os.Stderr.WriteString("--mqtthost missing eg --mqtthost=tcp://example.com:1883\n")
		exit = true
	}

	if temperatureURL == "" {
		os.Stderr.WriteString("--url missig, eg --url=http://example.com/temp.txt\n")
		exit = true
	}
	_, err := url.ParseRequestURI(temperatureURL)
	if err != nil {
		os.Stderr.WriteString("malformed url: " + temperatureURL + "\n")
		exit = true
	}

	if exit {
		os.Exit(1)
	}

}

func main() {
	log.SetLevel(log.DebugLevel)
	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go util.Webserver("Prometheus", ":9100", prometheusMux)

	agent = ag.NewAgent(mqttHost, "temperature")
	err := agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	go func() {
		done := make(chan os.Signal)
		signal.Notify(done, os.Interrupt)
		<-done
		log.Info("Shutting down temperature")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	for !agent.IsTerminated() {
		update()
		time.Sleep(time.Duration(updateInterval) * time.Minute)
	}

}
