package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"time"
)

// Graphite is a struct that defines the relevant properties of a graphite
// connection
type Graphite struct {
	Host       string
	Port       int
	Protocol   string
	Timeout    time.Duration
	Prefix     string
	conn       net.Conn
	nop        bool
	DisableLog bool
}

// defaultTimeout is the default number of seconds that we're willing to wait
// before forcing the connection establishment to fail
const defaultTimeout = 5

// IsNop is a getter for *graphite.Graphite.nop
func (graphite *Graphite) IsNop() bool {
	return graphite.nop
}

// Given a Graphite struct, Connect populates the Graphite.conn field with an
// appropriate TCP connection
func (graphite *Graphite) Connect() error {
	if !graphite.IsNop() {
		if graphite.conn != nil {
			graphite.conn.Close()
		}

		address := net.JoinHostPort(graphite.Host, fmt.Sprintf("%d", graphite.Port))

		if graphite.Timeout == 0 {
			graphite.Timeout = defaultTimeout * time.Second
		}

		var err error
		var conn net.Conn

		if graphite.Protocol == "udp" {
			var udpAddr *net.UDPAddr
			udpAddr, err = net.ResolveUDPAddr("udp", address)
			if err != nil {
				return err
			}
			conn, err = net.DialUDP(graphite.Protocol, nil, udpAddr)
		} else {
			conn, err = net.DialTimeout(graphite.Protocol, address, graphite.Timeout)
		}

		if err != nil {
			return err
		}

		graphite.conn = conn
	}

	return nil
}

// Given a Graphite struct, Disconnect closes the Graphite.conn field
func (graphite *Graphite) Disconnect() error {
	err := graphite.conn.Close()
	graphite.conn = nil
	return err
}

// Given a Metric struct, the SendMetric method sends the supplied metric to the
// Graphite connection that the method is called upon
func (graphite *Graphite) SendMetric(metric Metric) error {
	metrics := make([]Metric, 1)
	metrics[0] = metric

	return graphite.sendMetrics(metrics)
}

// Given a slice of Metrics, the SendMetrics method sends the metrics, as a
// batch, to the Graphite connection that the method is called upon
func (graphite *Graphite) SendMetrics(metrics []Metric) error {
	return graphite.sendMetrics(metrics)
}

// formatMetric formats a single metric for sending to Graphite
func (graphite *Graphite) formatMetric(metric Metric) (string, bool) {
	zeroed_metric := Metric{} // ignore unintialized metrics
	if metric == zeroed_metric {
		return "", false // ignore unintialized metrics
	}

	timestamp := metric.Timestamp
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}

	metric_name := metric.Name
	if graphite.Prefix != "" {
		metric_name = fmt.Sprintf("%s.%s", graphite.Prefix, metric.Name)
	}

	return fmt.Sprintf("%s %s %d\n", metric_name, metric.Value, timestamp), true
}

// sendMetrics is an internal function that is used to write to the TCP
// connection in order to communicate metrics to the remote Graphite host
func (graphite *Graphite) sendMetrics(metrics []Metric) error {
	if graphite.IsNop() {
		if !graphite.DisableLog {
			for _, metric := range metrics {
				log.Printf("Graphite: %s\n", metric)
			}
		}
		return nil
	}

	buf := bytes.NewBufferString("")
	for _, metric := range metrics {
		formatted, valid := graphite.formatMetric(metric)
		if !valid {
			continue
		}

		if graphite.Protocol == "udp" {
			fmt.Fprint(graphite.conn, formatted)
			continue
		}
		buf.WriteString(formatted)
	}

	if graphite.Protocol == "tcp" {
		_, err := graphite.conn.Write(buf.Bytes())
		//fmt.Print("Sent msg:", buf.String(), "'")
		if err != nil {
			return err
		}
	}
	return nil
}

// The SimpleSend method can be used to just pass a metric name and value and
// have it be sent to the Graphite host with the current timestamp
func (graphite *Graphite) SimpleSend(stat string, value string) error {
	metrics := make([]Metric, 1)
	metrics[0] = NewMetric(stat, value, time.Now().Unix())
	err := graphite.sendMetrics(metrics)
	if err != nil {
		return err
	}
	return nil
}

// NewGraphite is a factory method that's used to create a new Graphite
func NewGraphite(host string, port int) (*Graphite, error) {
	return GraphiteFactory("tcp", host, port, "")
}

// NewGraphiteWithMetricPrefix is a factory method that's used to create a new Graphite with a metric prefix
func NewGraphiteWithMetricPrefix(host string, port int, prefix string) (*Graphite, error) {
	return GraphiteFactory("tcp", host, port, prefix)
}

// When a UDP connection to Graphite is required
func NewGraphiteUDP(host string, port int) (*Graphite, error) {
	return GraphiteFactory("udp", host, port, "")
}

// NewGraphiteNop is a factory method that returns a Graphite struct but will
// not actually try to send any packets to a remote host and, instead, will just
// log. This is useful if you want to use Graphite in a project but don't want
// to make Graphite a requirement for the project.
func NewGraphiteNop(host string, port int) *Graphite {
	graphiteNop, _ := GraphiteFactory("nop", host, port, "")
	return graphiteNop
}

func GraphiteFactory(protocol string, host string, port int, prefix string) (*Graphite, error) {
	var graphite *Graphite

	switch protocol {
	case "tcp":
		graphite = &Graphite{Host: host, Port: port, Protocol: "tcp", Prefix: prefix}
	case "udp":
		graphite = &Graphite{Host: host, Port: port, Protocol: "udp", Prefix: prefix}
	case "nop":
		graphite = &Graphite{Host: host, Port: port, nop: true}
	}

	err := graphite.Connect()
	if err != nil {
		return nil, err
	}

	return graphite, nil
}
