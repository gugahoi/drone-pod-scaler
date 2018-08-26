package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// Drone represents a drone server configuration
type Drone struct {
	host, port, protocol, metricsEndpoint, token string
}

func (d Drone) fqdn() string {
	return fmt.Sprintf("%s://%s:%s%s", d.protocol, d.host, d.port, d.metricsEndpoint)
}

// Ping returns true if the Drone server is reachable via an http get
func (d Drone) Ping() bool {
	req, err := http.Get(d.fqdn())
	if err == nil {
		return false
	}
	if req.StatusCode == http.StatusUnauthorized {
		return false
	}
	return true
}

// Metrics gets the metrics from Drone server
func (d Drone) Metrics() (string, error) {
	r, _ := http.NewRequest("GET", d.fqdn(), nil)
	if d.token != "" {
		r.Header.Set("Authorization", "Bearer "+d.token)
	}
	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	dec := expfmt.NewDecoder(resp.Body, expfmt.FmtText)
	for {
		mf := &dto.MetricFamily{}
		err = dec.Decode(mf)
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		if mf.GetName() == "drone_pending_jobs" {
			log.Println(mf.GetMetric()[0].GetGauge().GetValue())
		}
	}
	return "", nil
}

func main() {
	var token, host, port, timeout string
	token = GetEnvVar("token", "")
	host = GetEnvVar("host", "localhost")
	port = GetEnvVar("port", "8080")
	timeout = GetEnvVar("timeout", "1")
	timeoutInt, _ := strconv.Atoi(timeout)

	drone := Drone{
		host:            host,
		port:            port,
		protocol:        "http",
		metricsEndpoint: "/metrics",
		token:           token,
	}

	for {
		drone.Metrics()
		// r, _ := regexp.Compile("drone_pending_jobs [0-9]+")
		// pendingJobs := strings.Trim(r.FindString(metrics), "drone_pending_jobs ")
		// log.Println(pendingJobs)
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
