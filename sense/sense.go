/*
/temperature/cookiename
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	"github.com/prometheus/client_golang/prometheus"
)

// senseData holds info posted from the sen.se API
// the struct doesn't define all posted attributes
type senseData struct {
	NodeUID string `json:"nodeUid"`
	Data    struct {
		CentidegreeCelsius int `json:"centidegreeCelsius"`
	} `json:"data"`
}

var (
	nodeUIDMap        map[string]string
	mqttHost          string
	agent             *ag.Agent
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

// senseHandler parses data posted from the sen.se API and publishes temperature through MQTT
func senseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	var indata senseData
	postdata, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.WithField("error", err).Error("Error reading body", err)
		promUpdateCounter.WithLabelValues("500", "temperature", "parse").Inc()

		return
	}
	err = json.Unmarshal(postdata, &indata)
	if err != nil {
		log.WithField("error", err).Error("Error unmarshaling posted value", err)
		promUpdateCounter.WithLabelValues("500", "temperature", "parse").Inc()

		return
	}
	sensor, ok := nodeUIDMap[indata.NodeUID]
	if !ok {
		promUpdateCounter.WithLabelValues("400", "temperature", "unknown").Inc()
		log.WithField("error", fmt.Sprintf("%#v", indata)).Error("Unknown sensor")

		return
	}
	var temperature = float64(indata.Data.CentidegreeCelsius) / 100
	agent.Publish("temperature/"+sensor, true, fmt.Sprintf("%v", temperature))
	promUpdateCounter.WithLabelValues("200", "temperature", sensor).Inc()
	promTemperature.WithLabelValues(sensor).Set(temperature)
	log.WithFields(log.Fields{"topic": sensor, "value": temperature}).Debug("Published")

	return
}

func init() {
	prometheus.MustRegister(promUpdateCounter)
	prometheus.MustRegister(promTemperature)

	var configFile string
	flag.StringVar(&mqttHost, "mqtthost", "", "address and port for mqtt server eg tcp://example.com:1883")
	flag.StringVar(&configFile, "config", "", "full path to configfile eg --config=/etc/id_map.json ")
	flag.Parse()
	exit := false
	if mqttHost == "" {
		os.Stderr.WriteString("--mqtthost missing eg --mqtthost=tcp://example.com:1883\n")
		exit = true
	}
	if configFile == "" {
		os.Stderr.WriteString("--config missing eg --config=/etc/id_map.json\n")
		exit = true
	}

	if exit {
		os.Exit(1)
	}

	nodeUIDMap = make(map[string]string)
	jsonStr, err := ioutil.ReadFile(configFile)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Can't read %s\n", configFile))
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	err = json.Unmarshal([]byte(jsonStr), &nodeUIDMap)
	if err != nil {
		os.Stderr.WriteString("Can't unmarshal id_map.json")
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

}

func main() {
	log.SetLevel(log.DebugLevel)

	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go util.Webserver("Prometheus", ":9100", prometheusMux)

	senseMux := http.NewServeMux()
	senseMux.HandleFunc("/", senseHandler)
	go util.Webserver("sense", ":8080", senseMux)

	agent = ag.NewAgent(mqttHost, "sense")
	err := agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}

	for !agent.IsTerminated() {
		time.Sleep(time.Second * 2)
	}

}

//{
//     "feedUid": "XXX",
//     "nodeUid": "XXXX",
//     "type": "alert",
//     "profile": "DoorStandard",
//     "dateEvent": "2015-01-31T17:09:15.871376",
//     "dateServer": "2015-01-31T17:09:15.871382",
//     "expiresAt": null,
//     "gatewayNodeUid": "XXX",
//     "geometry": {
//         "type": "Point",
//         "coordinates": [
//             48.830126299999996,
//             2.2466682
//         ]
//     },
//     "signal": 3,
//     "data": {
//          "centidegreeCelsius": 3730
//       }
// }
