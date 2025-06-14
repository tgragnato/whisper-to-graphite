package main

import (
	"flag"
	"log"
	"sync"
)

func main() {
	baseDirectory := flag.String(
		"basedirectory",
		"/var/lib/graphite/whisper",
		"Base directory where whisper files are located. Used to retrieve the metric name from the filename.")
	directory := flag.String(
		"directory",
		"/var/lib/graphite/whisper/collectd",
		"Directory containing the whisper files you want to send to graphite again")
	graphiteHost := flag.String(
		"host",
		"127.0.0.1",
		"Hostname/IP of the graphite server")
	graphitePort := flag.Int(
		"port",
		2003,
		"graphite Port")
	graphiteProtocol := flag.String(
		"protocol",
		"tcp",
		"Protocol to use to transfer graphite data (tcp/udp/nop)")
	workers := flag.Int(
		"workers",
		5,
		"Workers to run in parallel")
	fromTs := flag.Int(
		"from",
		0,
		"Starting timestamp to dump data from")
	toTs := flag.Int(
		"to",
		2147483647,
		"Ending timestamp to dump data up to")
	pointsPerSecond := flag.Int64(
		"pps",
		0,
		"Number of maximum points per second to send (0 means rate limiter is disabled)")
	connectRetries := flag.Int(
		"retries",
		3,
		"How many connection retries worker will make before failure. It is progressive and each next pause will be equal to 'retry * 1s'")
	flag.Parse()

	if *graphiteProtocol != "tcp" &&
		*graphiteProtocol != "udp" &&
		*graphiteProtocol != "nop" {
		log.Fatalln("Graphite protocol " + *graphiteProtocol + " not supported, use tcp/udp/nop.")
	}
	ch := make(chan string)
	quit := make(chan int)
	var wg sync.WaitGroup

	rl := newRateLimiter(*pointsPerSecond)
	wg.Add(*workers)
	for i := 0; i < *workers; i++ {
		go worker(ch, quit, &wg, *baseDirectory, *graphiteHost, *graphitePort, *graphiteProtocol, *fromTs, *toTs, *connectRetries, rl)
	}
	go findWhisperFiles(ch, quit, *directory)
	wg.Wait()
}
