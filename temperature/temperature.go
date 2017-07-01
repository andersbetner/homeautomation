package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andersbetner/homeautomation/util"
	"github.com/containous/traefik/log"
	"github.com/prometheus/client_golang/prometheus"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/mqtt"
)

var (
	mqttAdaptor     *mqtt.Adaptor
	mqttHost        string
	temperatureURL  string
	updateInterval  int // minutes default=15
	promUpdateCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ab_update_count",
		Help: "Number of updates performed. (periodicity = updates every x minutes)",
	},
		[]string{"name", "status", "periodicity"})
	promTemperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_temperature",
			Help: "Temperature. (periodicity = updates every x minutes)",
		}, []string{"name", "periodicity"},
	)
)

func init() {
	prometheus.MustRegister(promUpdateCount)
	prometheus.MustRegister(promTemperature)

	exit := false
	mqttHost, _ = os.LookupEnv("TEMPERATURE_MQTTHOST")
	if mqttHost == "" {
		os.Stderr.WriteString("env TEMPERATURE_MQTTHOST missing tcp://example.com:1883\n")
		exit = true
	}
	temperatureURL, _ = os.LookupEnv("TEMPERATURE_URL")
	if temperatureURL == "" {
		os.Stderr.WriteString("env TEMPERATURE_URL missig, eg http://example.com/temp.txt\n")
		exit = true
	}
	_, err := url.ParseRequestURI(temperatureURL)
	if err != nil {
		os.Stderr.WriteString("malformed url: " + temperatureURL + "\n")
		exit = true
	}
	interval, _ := os.LookupEnv("TEMPERATURE_UPDATE_INTERVAL")
	updateInterval = 15
	if interval != "" {
		updateInterval, err = strconv.Atoi(interval)
		if err != nil || updateInterval < 1 {
			os.Stderr.WriteString("env TEMPERATURE_UPDATE_INTERVAL must be an integer > 0\n")
			exit = true
		}
	}
	if exit {
		os.Exit(1)
	}

}

// update gets the outdoor temperature from temperatur.nu
func update() {
	client := &http.Client{
		CheckRedirect: nil,
		Timeout:       time.Second * 10,
	}
	request, _ := http.NewRequest("GET", temperatureURL, nil)
	resp, err := client.Do(request)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "request").Error("Error initiating request")

		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "body").Error("Error getting request body")

		return
	}
	temperature := strings.TrimSpace(string(body))
	_, err = strconv.ParseFloat(temperature, 64)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "value").Error("Error malformed float value", temperature)

		return
	}
	if !mqttAdaptor.Publish("temperature/outdoor", []byte(temperature)) {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "mqtt").Error("Error publishing mqtt")

		return
	}

	promUpdateCount.WithLabelValues("200").Inc()
	promTemperature.SetToCurrentTime()
}

func main() {
	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go util.Webserver("Prometheus", ":9100", prometheusMux)

	mqttAdaptor = mqtt.NewAdaptor(mqttHost, "temperature")
	work := func() {
		update()
		gobot.Every(time.Duration(updateInterval)*time.Minute, update)
	}
	robot := gobot.NewRobot("temperature",
		[]gobot.Connection{mqttAdaptor},
		work,
	)
	robot.Start()

}
