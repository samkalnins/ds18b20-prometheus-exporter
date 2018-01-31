package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/samkalnins/ds18b20-thermometer-prometheus-exporter/temp"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

//var bus_dir = flag.String("w1_bus_dir", "/sys/bus/w1/devices", "directory of the 1-wire bus")
var bus_dir = flag.String("w1_bus_dir", "fixtures/w1_devices", "directory of the 1-wire bus")
var port = flag.Int("port", 8000, "port to run http server on")

type prometheusLabels map[string][]string

// String is the method to format the flag's value, part of the flag.Value interface.
func (p *prometheusLabels) String() string {
	return fmt.Sprint(*p)
}

// Set is the method to set the flag value, part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
// It's a comma-separated list, so we split it.
func (p *prometheusLabels) Set(value string) error {
	*p = make(map[string][]string)

	for _, ls := range strings.Split(value, ",") {
		s := strings.Split(ls, "=")
		if len(s) != 3 {
			errors.New("Bad flag value -- should be temp_id=label=value")
		}
		_, initialized := (*p)[s[0]]
		if !initialized {
			(*p)[s[0]] = make([]string, 0)
		}
		(*p)[s[0]] = append((*p)[s[0]], fmt.Sprintf("%s=\"%s\"", s[1], s[2]))
	}
	return nil
}

var prometheusLabelsFlag prometheusLabels

func init() {
	flag.Var(&prometheusLabelsFlag, "prometheus_labels", "comma-separated list of labels to apply to sensors by ID e.g. 28-0417713760f=label=value,")
}

func main() {
	flag.Parse()

	log.Println(filepath.Abs("./"))

	// Main varz handler -- read and parse the temperatures on each request
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		readings, err := temperature.FindAndReadTemperatures(*bus_dir)
		if err != nil {
			log.Printf("Error reading temperatures [%s]", err)
			// TODO 500
		}

		for _, tr := range readings {
			labels := strings.Join(append(prometheusLabelsFlag[tr.Id], fmt.Sprintf("sensor=\"%s\"", tr.Id)), ",")

			// Output varz as both C & F for maximum user happiness
			fmt.Fprintf(w, "temperature_c{%s} %f\n", labels, tr.Temp_c)
			fmt.Fprintf(w, "temperature_f{%s} %f\n", labels, temperature.CentigradeToF(tr.Temp_c))
		}
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}