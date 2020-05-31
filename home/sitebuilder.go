package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/andersbetner/homeautomation/util"
	ag "github.com/andersbetner/mqttagent"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// env public path MQTTHOST

var (
	mqttHost      string
	publicPath    string
	templates     = make(map[string]*template.Template)
	page          = newPageData()
	updateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_sensor_updates_total",
			Help: "How many times this item has been updated.",
		},
		[]string{"status", "type", "name"},
	)
)

type pageData struct {
	Ica         int64
	Temperature util.Temperature
	Users       []string
	Opacs       map[string]*util.Opac
	Otrafs      map[string]*util.Otraf
}

func newPageData() *pageData {
	p := &pageData{}
	p.Ica = -99
	p.Temperature = *util.NewTemperature()
	p.Users = []string{"anders", "anna", "lowe", "malva", "vega"}
	p.Opacs = make(map[string]*util.Opac)
	for _, user := range p.Users {
		p.Opacs[user] = new(util.Opac)
	}
	p.Otrafs = make(map[string]*util.Otraf)
	for _, user := range p.Users {
		p.Otrafs[user] = new(util.Otraf)
	}

	return p
}

func libraryName(name string) string {
	switch name {
	case "Kungsbergsskolan":
		return "(Kungsberget)"
	case "Ekkälleskolan":
		return "(Ekkällan)"
	case "Blästadskolan":
		return "(Blästad)"
	}
	return ""
}

func (p *pageData) OpacsSlice() []*util.Opac {
	var ret []*util.Opac
	for _, user := range p.Users {
		ret = append(ret, p.Opacs[user])
	}

	return ret
}

func (p *pageData) OtrafsSlice() []*util.Otraf {
	var ret []*util.Otraf
	for _, user := range p.Users {
		ret = append(ret, p.Otrafs[user])
	}

	return ret
}

func copyStaticFiles() error {
	err := os.Mkdir(path.Join(publicPath, "css"), 0775)
	if err != nil && !os.IsExist(err) {
		return err
	}
	err = os.Mkdir(path.Join(publicPath, "images"), 0775)
	if err != nil && !os.IsExist(err) {
		return err
	}
	err = os.Mkdir(path.Join(publicPath, "js"), 0775)
	if err != nil && !os.IsExist(err) {
		return err
	}
	err = util.CopyFile(path.Join(publicPath, "css", "minimal.css"), "public/css/minimal.css")
	if err != nil {
		return err
	}
	err = util.CopyFile(path.Join(publicPath, "js", "minimal.js"), "public/js/minimal.js")
	if err != nil {
		return err
	}
	err = util.CopyDir(path.Join(publicPath, "images"), "public/images")
	if err != nil {
		return err
	}

	return nil
}

func render(template string) error {
	outFile, err := ioutil.TempFile(publicPath, "tmp")
	if err != nil {
		return err
	}
	os.Chmod(outFile.Name(), 0644)
	err = templates[template].ExecuteTemplate(outFile, "layout", page)
	if err != nil {
		return err
	}
	err = os.Rename(outFile.Name(), path.Join(publicPath, template))
	if err != nil {
		return err
	}

	return nil
}
func updateTemperature(client mqtt.Client, msg mqtt.Message) {
	value, err := strconv.ParseFloat(string(msg.Payload()), 64)
	sensor := path.Base(strings.Replace(msg.Topic(), "/state", "", 1))

	if err != nil {
		log.WithFields(log.Fields{"error": err,
			"type": "temperature",
			"name": sensor}).Error("Error converting float value")
		updateCounter.WithLabelValues("500", "temperature", sensor).Inc()

		return
	}
	err = page.Temperature.Set(sensor, value)
	if err != nil {
		log.WithFields(log.Fields{"error": err,
			"type": "temperature",
			"name": sensor}).Error("Unknown sensor")
		updateCounter.WithLabelValues("500", "temperature", sensor).Inc()

		return

	}
	err = render("index.html")
	if err != nil {
		log.WithFields(log.Fields{"error": err,
			"type": "temperature",
			"name": sensor}).Error("Error rendering")
		updateCounter.WithLabelValues("500", "temperature", sensor).Inc()

		return
	}
	updateCounter.WithLabelValues("200", "temperature", sensor).Inc()
}

func updateIca(client mqtt.Client, msg mqtt.Message) {
	value, err := strconv.ParseInt(string(msg.Payload()), 10, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"type":  "ica",
			"name":  "amount",
			"value": msg.Payload()}).Error("Error converting int value")
		updateCounter.WithLabelValues("500", "ica", "amount").Inc()

		return
	}
	page.Ica = value
	render("index.html")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"type":  "ica",
			"name":  "amount"}).Error("Error rendering ica")
		updateCounter.WithLabelValues("500", "ica", "amount").Inc()

		return
	}
	updateCounter.WithLabelValues("200", "ica", "amount").Inc()
}

func updateOpac(client mqtt.Client, msg mqtt.Message) {
	user := path.Base(msg.Topic())
	opac := new(util.Opac)
	err := json.Unmarshal(msg.Payload(), &opac)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"type":  "opac",
			"name":  user,
			"value": string(msg.Payload())}).Error("Error unmarshal json")
		updateCounter.WithLabelValues("500", "opac", user).Inc()

		return
	}

	page.Opacs[user] = opac

	err = render("library.html")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"type":  "opac",
			"name":  user}).Error("Error rendering opac")
		updateCounter.WithLabelValues("500", "opac", user).Inc()

		return
	}
	updateCounter.WithLabelValues("200", "opac", user).Inc()
}

func updateOtraf(client mqtt.Client, msg mqtt.Message) {
	user := path.Base(msg.Topic())
	otraf := new(util.Otraf)
	err := json.Unmarshal(msg.Payload(), &otraf)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"type":  "opac",
			"name":  user,
			"value": msg.Payload()}).Error("Error unmarshal json")
		updateCounter.WithLabelValues("500", "otraf", user).Inc()

		return
	}

	page.Otrafs[user] = otraf

	err = render("bus.html")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"type":  "opac",
			"name":  user}).Error("Error rendering otraf")
		updateCounter.WithLabelValues("500", "otraf", user).Inc()

		return
	}
	updateCounter.WithLabelValues("200", "otraf", user).Inc()
}

func init() {
	prometheus.MustRegister(updateCounter)

	flag.StringVar(&mqttHost, "mqtthost", "", "address and port for mqtt server eg tcp://example.com:1883")
	flag.StringVar(&publicPath, "publicpath", "", "path where site is rendered eg /www/site")
	flag.Parse()
	var exit bool
	if mqttHost == "" {
		os.Stderr.WriteString("--mqtthost missing eg --mqtthost=tcp://example.com:1883\n")
		exit = true
	}

	if publicPath == "" {
		os.Stderr.WriteString("--publicpath missing eg --publicpath=/www/site\n")
		exit = true
	}
	if exit {
		os.Exit(1)
	}

	funcMap := template.FuncMap{
		"ToLower":     strings.ToLower,
		"libraryName": libraryName,
	}
	templates["index.html"] = template.Must(template.ParseFiles("templates/index.html", "templates/layout.html"))
	templates["library.html"] = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/library.html", "templates/layout.html"))
	templates["bus.html"] = template.Must(template.New("").Funcs(funcMap).ParseFiles("templates/bus.html", "templates/layout.html"))

	err := copyStaticFiles()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Unable to copy static files to %s\nError: %s", publicPath, err.Error()))
		os.Exit(1)
	}
}

func main() {

	render("index.html")
	prometheusMux := http.NewServeMux()
	prometheusMux.Handle("/metrics", prometheus.Handler())
	go util.Webserver("prometheus", ":9100", prometheusMux)
	host, err := os.Hostname()
	if err != nil {
		host = "sune"
	}
	agent := ag.NewAgent(mqttHost, "sitebuilder-"+host)
	err = agent.Connect()
	if err != nil {
		log.WithField("error", err).Error("Can't connect to mqtt server")
		os.Exit(1)
	}
	agent.Subscribe("temperature/outdoor/state", updateTemperature)
	agent.Subscribe("homeassistant/sensor/motion_loft_temperature/state", updateTemperature)
	agent.Subscribe("homeassistant/sensor/motion_hall_uppe_temperature/state", updateTemperature)
	agent.Subscribe("homeassistant/sensor/motion_hall_nere_temperature/state", updateTemperature)
	agent.Subscribe("homeassistant/sensor/motion_sovrum_temperature/state", updateTemperature)
	agent.Subscribe("homeassistant/sensor/motion_badrum_uppe_temperature/state", updateTemperature)
	agent.Subscribe("homeassistant/sensor/motion_tvattstuga_temperature/state", updateTemperature)
	agent.Subscribe("ica/availableamount", updateIca)
	agent.Subscribe("opac/#", updateOpac)
	agent.Subscribe("otraf/#", updateOtraf)

	for !agent.IsTerminated() {
		time.Sleep(time.Second * 2)
	}
}
