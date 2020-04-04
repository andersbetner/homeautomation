/*
mqtt topics
ica/update (Will update on whatever message)
ica/availableamount
ica/all

type, topic, status
*/
package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/andersbetner/homeautomation/ica"
	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	mqttHost          string
	icaUser           string
	icaPassword       string
	agent             *ag.Agent
	updateInterval    int // Minutes default = 30
	promUpdateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_sensor_updates_total",
			Help: "How many times this item has been updated.",
		},
		[]string{"status", "type", "topic"},
	)
	promAmount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_ica",
			Help: "ICA data",
		}, []string{"topic"},
	)
)

// update gets the latest account funds from ica.se
func update() {
	icaClient := &ica.IcaClient{}
	err := icaClient.Login(icaUser, icaPassword)
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "login").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "login"}).Error("Error logging in")

		return
	}
	resp, err := icaClient.GetAccount()
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "account").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "account"}).Error("Error getting account")

		return

	}
	icaData := ica.New()
	icaData, err = ica.ParseAccount(resp, icaData)
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "parse").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "parse"}).Error("Error parsing response body")

		return
	}
	resp, err = icaClient.GetTransactions()
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "transactions").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "transactions"}).Error("Error getting transactions")

		return

	}
	icaData, err = ica.ParseTransactions(resp, icaData)
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "parse").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "parse"}).Error("Error parsing transactions")

		return
	}

	err = agent.Publish("ica/availableamount", true, strconv.Itoa(int(icaData.Available)))
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "publish").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "availableamount"}).Error("Error publishing ica/availableamount")

		return
	}
	promUpdateCounter.WithLabelValues("200", "ica", "availableamount").Inc()
	promAmount.WithLabelValues("availableamount").Set(icaData.Available)

	b, err := json.Marshal(icaData)
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "json").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "json"}).Error("Error marshalling json")

		return
	}
	err = agent.Publish("ica/all", true, string(b))
	if err != nil {
		promUpdateCounter.WithLabelValues("500", "ica", "publish").Inc()
		log.WithFields(log.Fields{"error": err,
			"type":  "ica",
			"topic": "all"}).Error("Error publishing ica/all")

		return
	}
	promUpdateCounter.WithLabelValues("200", "ica", "all").Inc()
	log.WithField("amount", icaData.Available).Debug("Update published")

}

// updateHandler acts on mqtt messages to ica/update
func updateHandler(client mqtt.Client, msg mqtt.Message) {
	command := string(msg.Payload())
	log.WithFields(log.Fields{"command": command}).Debug("Update requested through mqtt")
	update()
}

func init() {
	log.SetLevel(log.DebugLevel)
	prometheus.MustRegister(promUpdateCounter)
	prometheus.MustRegister(promAmount)

	exit := false
	flag.StringVar(&mqttHost, "mqtthost", "", "address and port for mqtt server eg tcp://example.com:1883")
	flag.IntVar(&updateInterval, "updateinterval", 30, "integer > 0")
	flag.Parse()
	if mqttHost == "" {
		os.Stderr.WriteString("--mqtthost missing eg --mqtthost=tcp://example.com:1883\n")
		exit = true
	}
	icaUser, _ = os.LookupEnv("ICA_USER")
	if icaUser == "" {
		os.Stderr.WriteString("env ICA_USER missing\n")
		exit = true
	}
	icaPassword, _ = os.LookupEnv("ICA_PASSWORD")
	if icaPassword == "" {
		os.Stderr.WriteString("env ICA_PASSWORD missing\n")
		exit = true
	}
	if exit {
		os.Exit(1)
	}

}

func main() {
	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go util.Webserver("prometheus", ":9100", prometheusMux)

	agent = ag.NewAgent(mqttHost, "ica")
	err := agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	agent.Subscribe("ica/update", updateHandler)
	go func() {
		done := make(chan os.Signal)
		signal.Notify(done, os.Interrupt)
		<-done
		log.Info("Shutting down ica")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	for !agent.IsTerminated() {
		update()
		time.Sleep(time.Duration(updateInterval) * time.Minute)
	}

}
