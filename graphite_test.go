package main

import (
	"net"
	"testing"
)

func TestIsNop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		graphite *Graphite
		want     bool
	}{
		{
			name:     "nop is true",
			graphite: &Graphite{nop: true},
			want:     true,
		},
		{
			name:     "nop is false",
			graphite: &Graphite{nop: false},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.graphite.IsNop(); got != tt.want {
				t.Errorf("Graphite.IsNop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGraphiteNop(t *testing.T) {
	t.Parallel()

	g := NewGraphiteNop("localhost", 2003)
	if !g.IsNop() {
		t.Errorf("NewGraphiteNop() should create a nop Graphite, got nop=%v", g.IsNop())
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()

	g := &Graphite{
		Host:     "localhost",
		Port:     2003,
		Protocol: "tcp",
		nop:      false,
	}

	if err := g.Connect(); err == nil {
		t.Errorf("Connect() on non-nop Graphite should return an error, got nil")
	}

	g.nop = true
	if err := g.Connect(); err != nil {
		t.Errorf("Connect() on nop Graphite returned error: %v", err)
	}
}

func TestDisconnect(t *testing.T) {
	t.Parallel()

	conn, _ := net.Pipe()
	g := &Graphite{
		conn: conn,
	}

	err := g.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}
	if g.conn != nil {
		t.Errorf("Disconnect() did not set conn to nil")
	}
}

func TestSendMetric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		graphite *Graphite
		metric   Metric
		wantErr  bool
	}{
		{
			name:     "nop graphite",
			graphite: &Graphite{nop: true},
			metric:   Metric{Name: "test.metric", Value: "10", Timestamp: 1234567890},
			wantErr:  false,
		},
		{
			name:     "nop graphite with logging disabled",
			graphite: &Graphite{nop: true, DisableLog: true},
			metric:   Metric{Name: "test.metric", Value: "10", Timestamp: 1234567890},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.graphite.SendMetric(tt.metric)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	g := &Graphite{
		conn:     conn1,
		Protocol: "tcp",
	}

	metric := NewMetric("test.metric", "10", 1234567890)

	go func() {
		buf := make([]byte, 1024)
		if _, err := conn2.Read(buf); err != nil {
			t.Errorf("Error reading from mock connection: %v", err)
		}
	}()

	if err := g.SendMetric(metric); err != nil {
		t.Errorf("SendMetric() with mock connection error = %v", err)
	}
}

func TestSendMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		graphite *Graphite
		metrics  []Metric
		wantErr  bool
	}{
		{
			name:     "nop graphite",
			graphite: &Graphite{nop: true},
			metrics: []Metric{
				{Name: "test.metric1", Value: "10", Timestamp: 1234567890},
				{Name: "test.metric2", Value: "20", Timestamp: 1234567890},
			},
			wantErr: false,
		},
		{
			name:     "nop graphite with logging disabled",
			graphite: &Graphite{nop: true, DisableLog: true},
			metrics: []Metric{
				{Name: "test.metric1", Value: "10", Timestamp: 1234567890},
				{Name: "test.metric2", Value: "20", Timestamp: 1234567890},
			},
			wantErr: false,
		},
		{
			name:     "with prefix",
			graphite: &Graphite{nop: true, Prefix: "prefix"},
			metrics: []Metric{
				{Name: "test.metric", Value: "10", Timestamp: 1234567890},
			},
			wantErr: false,
		},
		{
			name:     "with zero value metric",
			graphite: &Graphite{nop: true},
			metrics: []Metric{
				{}, // zero value metric should be ignored
				{Name: "test.metric", Value: "10", Timestamp: 1234567890},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.graphite.SendMetrics(tt.metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	g := &Graphite{
		conn:     conn1,
		Protocol: "tcp",
	}

	go func() {
		buf := make([]byte, 1024)
		if _, err := conn2.Read(buf); err != nil {
			t.Errorf("Error reading from mock connection: %v", err)
			return
		}
	}()

	metrics := []Metric{
		NewMetric("test.metric1", "10", 1234567890),
		NewMetric("test.metric2", "20", 1234567890),
	}

	if err := g.SendMetrics(metrics); err != nil {
		t.Errorf("SendMetrics() with mock TCP connection error = %v", err)
	}

	udpConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	defer udpConn.Close()

	g = &Graphite{
		conn:     udpConn,
		Protocol: "udp",
	}

	if err := g.SendMetrics(metrics); err != nil {
		t.Errorf("SendMetrics() with mock UDP connection error = %v", err)
	}
}

func TestSimpleSend(t *testing.T) {
	t.Parallel()

	t.Run("nop graphite", func(t *testing.T) {
		g := &Graphite{nop: true}
		err := g.SimpleSend("test.metric", "10")
		if err != nil {
			t.Errorf("SimpleSend() with nop Graphite returned error: %v", err)
		}
	})

	t.Run("nop graphite with logging disabled", func(t *testing.T) {
		g := &Graphite{nop: true, DisableLog: true}
		err := g.SimpleSend("test.metric", "10")
		if err != nil {
			t.Errorf("SimpleSend() with nop Graphite and logging disabled returned error: %v", err)
		}
	})

	t.Run("tcp connection", func(t *testing.T) {
		conn1, conn2 := net.Pipe()
		defer conn1.Close()
		defer conn2.Close()

		g := &Graphite{
			conn:     conn1,
			Protocol: "tcp",
		}

		go func() {
			buf := make([]byte, 1024)
			if _, err := conn2.Read(buf); err != nil {
				t.Errorf("Error reading from mock connection: %v", err)
			}
		}()

		err := g.SimpleSend("test.metric", "10")
		if err != nil {
			t.Errorf("SimpleSend() with mock TCP connection error = %v", err)
		}
	})

	t.Run("udp connection", func(t *testing.T) {
		udpConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		defer udpConn.Close()

		g := &Graphite{
			conn:     udpConn,
			Protocol: "udp",
		}

		err := g.SimpleSend("test.metric", "10")
		if err != nil {
			t.Errorf("SimpleSend() with mock UDP connection error = %v", err)
		}
	})

	t.Run("with prefix", func(t *testing.T) {
		g := &Graphite{nop: true, Prefix: "prefix"}
		err := g.SimpleSend("test.metric", "10")
		if err != nil {
			t.Errorf("SimpleSend() with prefix returned error: %v", err)
		}
	})
}
