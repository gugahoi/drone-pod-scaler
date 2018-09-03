package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/common/expfmt"
)

var cummulativePendingJobs int

// Drone represents a drone server configuration
type Drone struct {
	host, port, protocol, metricsEndpoint, token string
}

func (d Drone) fqdn() string {
	return fmt.Sprintf("%s://%s:%s%s", d.protocol, d.host, d.port, d.metricsEndpoint)
}

// GetMetrics gets the metrics from Drone server
func (d Drone) GetMetrics() error {
	r, _ := http.NewRequest("GET", d.fqdn(), nil)
	if d.token != "" {
		r.Header.Set("Authorization", "Bearer "+d.token)
	}
	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	parser := expfmt.TextParser{}
	s, _ := parser.TextToMetricFamilies(resp.Body)
	v := s["drone_pending_jobs"].GetMetric()[0].GetGauge().GetValue()
	if v == 0 && cummulativePendingJobs > 0 {
		cummulativePendingJobs--
	} else {
		cummulativePendingJobs += int(v)
	}
	return nil
}

func main() {
	var token, host, port, timeout string
	token = GetEnvVar("token", "")
	host = GetEnvVar("host", "localhost")
	port = GetEnvVar("port", "80")
	timeout = GetEnvVar("timeout", "1")
	timeoutInt, _ := strconv.Atoi(timeout)

	drone := Drone{
		host:            host,
		port:            port,
		protocol:        "https",
		metricsEndpoint: "/metrics",
		token:           token,
	}

	for {
		drone.GetMetrics()
		log.Printf("Cummulative pending jobs: %d\n", cummulativePendingJobs)
		if cummulativePendingJobs > 10 {
			log.Println("We should probably scale now...")
		} else if cummulativePendingJobs == 0 {
			log.Println("Might want to scale down...")
		} else {
			log.Println("Nothing to do...")
		}
		time.Sleep(time.Duration(timeoutInt) * time.Second)
	}
}

// GetEnvVar gets an env variable or returns a default when not found
func GetEnvVar(name string, def string) string {
	val, found := os.LookupEnv(name)
	if found {
		return val
	}
	return def
}
