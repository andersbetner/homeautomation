/*
mqtt topics
ica/update (Will update on whatever message)
ica/availableamount
ica/all
*/
package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/andersbetner/homeautomation/webserver"
	"github.com/prometheus/client_golang/prometheus"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/mqtt"
)

var (
	mqttAdaptor     *mqtt.Adaptor
	mqttHost        string
	icaUser         string
	icaPassword     string
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

func init() {
	prometheus.MustRegister(promUpdateCount)
	prometheus.MustRegister(promUpdateTimestamp)

	exit := false
	mqttHost, _ = os.LookupEnv("ICA_MQTTHOST")
	if mqttHost == "" {
		os.Stderr.WriteString("env ICA_MQTTHOST missing tcp://example.com:1883\n")
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
	interval, _ := os.LookupEnv("ICA_UPDATE_INTERVAL")
	if interval != "" {
		var err error
		updateInterval, err = strconv.Atoi(interval)
		if err != nil || updateInterval < 1 {
			os.Stderr.WriteString("env ICA_UPDATE_INTERVAL must be integer > 0\n")
			exit = true
		}
	}
	if exit {
		os.Exit(1)
	}
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
		log.WithField("error", "basicauth").Error("Error in basic auth")
		return
	}
	defer resp.Body.Close()

	authTicket := resp.Header.Get("AuthenticationTicket")
	request, _ = http.NewRequest("GET", "https://api.ica.se/api/user/minasidor/", nil)
	request.Header.Add("AuthenticationTicket", authTicket)
	resp, err = client.Do(request)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "authticket").Error("Error in authticket")

		return
	}
	defer resp.Body.Close()
	jsonBlob, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "body").Error("Error parsing response body")

		return
	}
	ica := accountInfo{}
	err = json.Unmarshal(jsonBlob, &ica)
	if err != nil {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "unmarshal").Error("Error unmarshal json")
		return
	}

	if !mqttAdaptor.Publish("ica/availableamount", []byte(strconv.Itoa(int(ica.AvailableAmount)))) ||
		!mqttAdaptor.Publish("ica/all", []byte(string(jsonBlob))) {
		promUpdateCount.WithLabelValues("500").Inc()
		log.WithField("error", "mqtt").Error("Error publishing mqtt")

		return
	}

	promUpdateCount.WithLabelValues("200").Inc()
	promUpdateTimestamp.SetToCurrentTime()

}

// updateHandler acts on mqtt messages to ica/update
func updateHandler(msg mqtt.Message) {
	command := string(msg.Payload())
	log.WithFields(log.Fields{"command": command}).Debug("Update requested through mqtt")
	update()
}

func main() {
	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go webserver.Webserver("Prometheus", ":9100", prometheusMux)

	mqttAdaptor = mqtt.NewAdaptor(mqttHost, "ica")
	work := func() {
		update()
		mqttAdaptor.On("ica/update", updateHandler)
		gobot.Every(time.Duration(updateInterval)*time.Minute, update)
	}
	robot := gobot.NewRobot("ica",
		[]gobot.Connection{mqttAdaptor},
		work,
	)
	robot.Start()
}
