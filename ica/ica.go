/*
mqtt topics
ica/update (Will update on whatever message)
ica/availableamount
ica/all
*/
package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	mqttHost        string
	icaUser         string
	icaPassword     string
	agent           *ag.Agent
	updateInterval  int // Minutes default = 30
	promUpdateCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ab_update_count",
		Help: "Number of updates performed.",
	},
		[]string{"status"})
	promUpdateTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ab_succsessful_update_timestamp",
		Help: "Timestamp of last update.",
	})
)

type accountInfo struct {
	AvailableAmount float64
}

// update gets the latest account funds from ica.se
func update() {
	client := &http.Client{
		CheckRedirect: nil,
		Timeout:       time.Second * 10,
	}
	request, _ := http.NewRequest("GET", "https://api.ica.se/api/login/", nil)
	request.SetBasicAuth(icaUser, icaPassword)
	resp, err := client.Do(request)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", err).Error("Error in basic auth")
		return
	}
	defer resp.Body.Close()

	authTicket := resp.Header.Get("AuthenticationTicket")
	request, _ = http.NewRequest("GET", "https://api.ica.se/api/user/minasidor/", nil)
	request.Header.Add("AuthenticationTicket", authTicket)
	resp, err = client.Do(request)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", err).Error("Error in authticket")

		return
	}
	defer resp.Body.Close()
	jsonBlob, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", err).Error("Error parsing response body")

		return
	}
	ica := accountInfo{}
	err = json.Unmarshal(jsonBlob, &ica)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", err).Error("Error unmarshal json")
		return
	}

	err = agent.Publish("ica/availableamount", true, strconv.Itoa(int(ica.AvailableAmount)))
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "mqtt").Error("Error publishing ica/availableamount")

		return
	}
	err = agent.Publish("ica/all", true, string(jsonBlob))
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", err).Error("Error publishing ica/all")

		return
	}

	promUpdateCount.WithLabelValues("200").Inc()
	promUpdateTimestamp.SetToCurrentTime()

}

// updateHandler acts on mqtt messages to ica/update
func updateHandler(client mqtt.Client, msg mqtt.Message) {
	command := string(msg.Payload())
	log.WithFields(log.Fields{"command": command}).Debug("Update requested through mqtt")
	update()
}

func init() {
	prometheus.MustRegister(promUpdateCount)
	prometheus.MustRegister(promUpdateTimestamp)

	exit := false
	flag.StringVar(&mqttHost, "mqtthost", "", "address and port for mqtt server eg tcp://example.com:1883")
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
	flag.IntVar(&updateInterval, "updateinterval", 30, "integer > 0")
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
