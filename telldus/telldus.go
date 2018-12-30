package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	mqttHost          string
	agent             *ag.Agent
	promUpdateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_agent_updates_total",
			Help: "How many times this item has been updated.",
		},
		[]string{"type"},
	)
	promErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_agent_updates_total",
			Help: "How many times errors has occured.",
		},
		[]string{"type", "topic"},
	)
)

func topic(data []string) (string, string) {
	house := data[1]
	unit := data[2]
	group := data[3]
	method := data[4]
	remoteMap := viper.GetStringMap("remotes")
	remote, ok := remoteMap[house]
	if !ok {
		return "", ""
	}
	topic := fmt.Sprintf("remote/%s/", remote)
	if group == "1" {
		topic += "g"
	} else {
		topic += unit
	}
	if method == "turnon" {
		return topic, "on"
	}
	if method == "turnoff" {
		return topic, "off"
	}

	return "", ""
}
func listener() {

	for {
		log.Debug("Connect telldus unix socket")

		telldusSocket, err := net.Dial("unix", "/tmp/TelldusEvents")
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Connect to telldus unix socket")
			promErrorCounter.WithLabelValues("telldus", "connect").Inc()
			time.Sleep(5 * time.Second)
			continue
		}
		defer telldusSocket.Close()
		buf := make([]byte, 1024)
		re := regexp.MustCompile(`TDRawDeviceEvent.*class:command;protocol:arctech;model:selflearning;house:(\d+);unit:(\d+);group:(\d+);method:(\w+)`)
		for {
			if telldusSocket != nil {
				data, err := telldusSocket.Read(buf[:])
				if err != nil {
					promErrorCounter.WithLabelValues("telldus", "read").Inc()
					log.WithField("error", err).Error("Read from socket")
					break
				}

				telldusData := string(buf[0:data])
				res := re.FindStringSubmatch(telldusData)
				if res != nil {
					topic, data := topic(res)
					if topic != "" {
						err := agent.Publish(topic, true, data)
						if err != nil {
							promErrorCounter.WithLabelValues("telldus", "publish").Inc()
							log.WithFields(log.Fields{"error": err,
								"type":  "telldus",
								"topic": topic}).Error("Error publishing telldus")
							continue
						}
						promUpdateCounter.WithLabelValues("telldus").Inc()
						log.WithFields(log.Fields{"topic": topic, "data": data}).Debug("Sent topic")

					}
				}
			}
		}
		time.Sleep(5 * time.Second)
	}

}

func init() {
	log.SetLevel(log.DebugLevel)
	prometheus.MustRegister(promUpdateCounter)
	viper.SetConfigName("telldusagent")
	viper.AddConfigPath("/etc/telldus")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	exit := false
	mqttHost = viper.GetString("mqtthost")

	if mqttHost == "" {
		log.Error("mqtthost missing in config")
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

	agent = ag.NewAgent(mqttHost, "telldus")
	err := agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	go func() {
		done := make(chan os.Signal)
		signal.Notify(done, os.Interrupt)
		<-done
		log.Debug("Shutting down telldus")
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	go listener()
	for !agent.IsTerminated() {

		time.Sleep(time.Duration(1) * time.Minute)
	}
}
