package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/prometheus/common/expfmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
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
		cummulativePendingJobs++
	}
	return nil
}

func main() {
	var token, host, port, timeout, protocol string
	token = GetEnvVar("token", "")
	host = GetEnvVar("host", "localhost")
	port = GetEnvVar("port", "80")
	timeout = GetEnvVar("timeout", "1")
	protocol = GetEnvVar("protocol", "http")
	timeoutInt, _ := strconv.Atoi(timeout)

	drone := Drone{
		host:            host,
		port:            port,
		protocol:        protocol,
		token:           token,
		metricsEndpoint: "/metrics",
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

func k8s() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	result, err := deploymentsClient.Create(deployment)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
}
