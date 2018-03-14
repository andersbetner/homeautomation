package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/andersbetner/homeautomation/otraf"
	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type user struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	Tab      string `json:"tab"`
}

var (
	mqttHost          string
	agent             *ag.Agent
	updateInterval    int
	users             []user
	promUpdateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_sensor_updates_total",
			Help: "How many times this item has been updated.",
		},
		[]string{"status", "type", "topic"},
	)
	promOtrafAmount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_otraf_amount",
			Help: "Amount on card",
		}, []string{"type", "topic"},
	)
	promOtrafCardStart = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_otraf_card_start",
			Help: "Timestamp when monthly och day pass starts 0 if no pass",
		}, []string{"type", "topic"},
	)
	promOtrafCardEnd = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_otraf_card_end",
			Help: "Timestamp when monthly och day pass ends 0 if no pass",
		}, []string{"type", "topic"},
	)
)

func update() {
	for _, user := range users {
		var err error
		var topic string
		if user.Tab == "" {
			topic = strings.ToLower(user.Name)
		} else {
			topic = strings.ToLower(user.Tab)
		}

		o := otraf.New(user.Name)
		resp, err := otraf.GetHTML(user.User, user.Password, user.Tab)
		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "otraf",
				"topic": topic}).Error("Error getting html")
			continue
		}
		o, err = otraf.Parse(resp, o)
		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "otraf",
				"topic": topic}).Error("Error parsing json")
			continue
		}
		out, err := json.Marshal(o)
		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "otraf",
				"topic": topic}).Error("Error marshal json")
			continue
		}
		err = agent.Publish("otraf/"+topic, true, string(out))
		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "otraf",
				"topic": topic}).Error("Error publish otraf")
			continue
		}
		promUpdateCounter.WithLabelValues("200", "otraf", topic).Inc()
		promOtrafAmount.WithLabelValues("otraf", topic).Set(float64(o.Amount))
		promOtrafCardStart.WithLabelValues("otraf", topic).Set(float64(o.CardStart.Unix()))
		promOtrafCardEnd.WithLabelValues("otraf", topic).Set(float64(o.CardEnd.Unix()))
		log.WithFields(log.Fields{
			"topic": topic}).Debug("Publish otraf")

	}
}
func init() {
	log.SetLevel(log.DebugLevel)
	prometheus.MustRegister(promUpdateCounter)
	prometheus.MustRegister(promOtrafAmount)
	prometheus.MustRegister(promOtrafCardStart)
	prometheus.MustRegister(promOtrafCardEnd)
	var configFile string
	flag.StringVar(&mqttHost, "mqtthost", "", "address and port for mqtt server eg tcp://example.com:1883")
	flag.StringVar(&configFile, "config", "", "full path to configfile eg --config=/etc/users.json ")
	flag.IntVar(&updateInterval, "updateinterval", 30, "integer > 0")
	flag.Parse()
	exit := false
	if mqttHost == "" {
		os.Stderr.WriteString("--mqtthost missing eg --mqtthost=tcp://example.com:1883\n")
		exit = true
	}
	if configFile == "" {
		os.Stderr.WriteString("--config missing eg --config=/etc/users.json\n")
		exit = true
	}

	if exit {
		os.Exit(1)
	}

	jsonStr, err := ioutil.ReadFile(configFile)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Can't read %s\n", configFile))
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	err = json.Unmarshal([]byte(jsonStr), &users)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Can't unmarshal json in %s\n", configFile))
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

}
func main() {
	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go util.Webserver("prometheus", ":9100", prometheusMux)
	host, err := os.Hostname()
	if err != nil {
		host = "opac"
	}
	agent = ag.NewAgent(mqttHost, "opac-"+host)
	err = agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	// agent.Subscribe("opac/update", updateHandler)
	go func() {
		done := make(chan os.Signal)
		signal.Notify(done, os.Interrupt)
		<-done
		log.Info("Shutting down opac")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	for !agent.IsTerminated() {
		update()
		time.Sleep(time.Duration(updateInterval) * time.Minute)
	}

}
