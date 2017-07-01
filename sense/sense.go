/*
/temperature/cookiename
*/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/andersbetner/homeautomation/util"
	"github.com/containous/traefik/log"
	"github.com/prometheus/client_golang/prometheus"

	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/mqtt"
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
	nodeUIDMap    map[string]string
	mqttHost      string
	mqttAdaptor   *mqtt.Adaptor
	updateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_sensor_updates_total",
			Help: "How many times this item has been updated.",
		},
		[]string{"name", "status", "periodicity"},
	)
)

func init() {
	nodeUIDMap = make(map[string]string)
	jsonStr, err := ioutil.ReadFile("id_map.json")
	if err != nil {
		os.Stderr.WriteString("Can't read id_map.json\n")
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	err = json.Unmarshal([]byte(jsonStr), &nodeUIDMap)
	if err != nil {
		os.Stderr.WriteString("Can't unmarshal id_map.json")
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	prometheus.MustRegister(updateCounter)
	mqttHost, _ = os.LookupEnv("SENSE_MQTTHOST")
	if mqttHost == "" {
		fmt.Println("env SENSE_MQTTHOST missing, tcp://mqtt.example.com:1884")
		os.Exit(1)
	}
}

// senseHandler parses data posted from the sen.se API and publishes temperature through MQTT
func senseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	var indata senseData
	postdata, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(postdata, &indata)
	if err != nil {
		log.WithField("error", "post").Error("Error unmarshaling posted value", err)
		updateCounter.WithLabelValues("500", "unknown").Inc()
		return
	}
	response := []byte("ok")
	w.Header().Set("Content-Length", fmt.Sprint(len(response)))
	w.Write(response)
	if name, ok := nodeUIDMap[indata.NodeUID]; ok {
		var temperature = float32(indata.Data.CentidegreeCelsius) / 100
		mqttAdaptor.Publish("temperature/"+name, []byte(fmt.Sprintf("%v", temperature)))
		updateCounter.WithLabelValues("200", name).Inc()
	}
}

func main() {
	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go util.Webserver("Prometheus", ":9100", prometheusMux)

	senseMux := http.NewServeMux()
	senseMux.HandleFunc("/", senseHandler)
	go util.Webserver("sense", ":8080", senseMux)

	mqttAdaptor = mqtt.NewAdaptor(mqttHost, "sense")
	work := func() {
	}

	robot := gobot.NewRobot("sense",
		[]gobot.Connection{mqttAdaptor},
		work,
	)

	robot.Start()
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
