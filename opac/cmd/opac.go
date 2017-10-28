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

	log "github.com/Sirupsen/logrus"
	"github.com/andersbetner/homeautomation/opac"
	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	"github.com/prometheus/client_golang/prometheus"
)

type user struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
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
	promOpac = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_opac_books_count",
			Help: "Library books.",
		}, []string{"type", "topic"},
	)
	promOpacDue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_opac_due",
			Help: "Library books.",
		}, []string{"topic"},
	)
	promOpacReservationPickup = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_opac_reservation_pickup",
			Help: "Library books.",
		}, []string{"topic"},
	)
	promOpacFee = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_opac_fee",
			Help: "Library books fees in SEK",
		}, []string{"topic"},
	)
)

func update() {
	for _, user := range users {
		var err error
		topic := strings.ToLower(user.Name)
		c, err := opac.NewClient(user.User, user.Password)
		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "opac",
				"topic": topic}).Error("Error create client")
			continue
		}
		if err = c.Login(); err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "opac",
				"topic": topic}).Error("Error login")
			continue
		}
		o := opac.New(user.Name)

		o, err = opac.Parse(c, o)

		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "opac",
				"topic": topic}).Error("Error parsing opac body")
			continue
		}
		bookCount := float64(len(o.Books))
		reservationCount := float64(len(o.Reservations))
		promOpac.WithLabelValues("loan", topic).Set(bookCount)
		promOpac.WithLabelValues("reservation", topic).Set(reservationCount)
		dueTime := float64(0)
		if bookCount > 0 {
			dueTime = float64(o.FirstDue().DateDue.Unix())
		}
		promOpacDue.WithLabelValues(topic).Set(dueTime)
		reservationPickup := float64(0)
		if o.ReservationPickup() {
			reservationPickup = float64(1)
		}
		promOpacReservationPickup.WithLabelValues(topic).Set(reservationPickup)
		promOpacFee.WithLabelValues(topic).Set(o.Fee)
		out, err := json.Marshal(o)
		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "opac",
				"topic": topic}).Error("Error marshal json")
		}
		err = agent.Publish("opac/"+topic, true, string(out))
		if err != nil {
			log.WithFields(log.Fields{"error": err,
				"type":  "opac",
				"topic": topic}).Error("Error publish opac")
		} else {
			log.WithFields(log.Fields{
				"topic": topic}).Debug("Publish opac")
		}
	}
}

func init() {
	log.SetLevel(log.DebugLevel)
	prometheus.MustRegister(promUpdateCounter)
	prometheus.MustRegister(promOpac)
	prometheus.MustRegister(promOpacDue)
	prometheus.MustRegister(promOpacReservationPickup)
	prometheus.MustRegister(promOpacFee)

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
