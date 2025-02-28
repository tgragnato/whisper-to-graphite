package main

import (
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-graphite/go-whisper"
	"github.com/marpaia/graphite-golang"
)

type rateLimiter struct {
	pointsPerSecond int64
	currentPoints   int64
	full            chan bool
	lock            *sync.Mutex
	enabled         bool
}

func newRateLimiter(pointsPerSecond int64) *rateLimiter {
	rl := new(rateLimiter)
	rl.pointsPerSecond = pointsPerSecond
	rl.currentPoints = 0
	rl.full = make(chan bool)
	rl.lock = new(sync.Mutex)
	if pointsPerSecond == 0 {
		rl.enabled = false
	} else {
		rl.enabled = true
		go func() {
			for {
				time.Sleep(1 * time.Second)
				select {
				case <-rl.full:
				default:
				}
			}
		}()
		return rl
	}
	return rl
}

func (rl *rateLimiter) limit(n int64) {
	if !rl.enabled {
		return
	}
	rl.lock.Lock()
	defer rl.lock.Unlock()

	rl.currentPoints += n
	if rl.currentPoints >= rl.pointsPerSecond {
		rl.full <- true
		rl.currentPoints = 0
	}
}

func convertFilename(filename string, baseDirectory string) (string, error) {
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	absBaseDirectory, err := filepath.Abs(baseDirectory)
	if err != nil {
		return "", err
	}
	err = nil
	if strings.HasPrefix(absFilename, absBaseDirectory) {
		metric := strings.Replace(
			strings.TrimPrefix(
				strings.TrimSuffix(
					strings.TrimPrefix(
						absFilename,
						absBaseDirectory),
					".wsp"),
				"/"),
			"/",
			".",
			-1)
		return metric, err
	}
	err = errors.New("path for whisper file does not live in BasePath")
	return "", err
}

func sendWhisperData(
	filename string,
	baseDirectory string,
	graphiteConn *graphite.Graphite,
	fromTs int,
	toTs int,
	connectRetries int,
	rateLimiter *rateLimiter,
) error {
	metricName, err := convertFilename(filename, baseDirectory)
	if err != nil {
		return err
	}

	whisperData, err := whisper.Open(filename)
	if err != nil {
		return err
	}
	archiveDataPoints, err := whisperData.Fetch(fromTs, toTs)
	if err != nil {
		return err
	}

	metrics := make([]graphite.Metric, 0, 1000)
	for _, dataPoint := range archiveDataPoints.Points() {
		interval := dataPoint.Time
		value := dataPoint.Value
		if math.IsNaN(value) || interval < fromTs || interval > toTs {
			continue
		}

		v := strconv.FormatFloat(value, 'f', 20, 64)
		metrics = append(metrics, graphite.NewMetric(metricName, v, int64(interval)))
	}

	rateLimiter.limit(int64(len(metrics)))
	for r := 1; r <= connectRetries; r++ {
		err = graphiteConn.SendMetrics(metrics)
		if err != nil && r != connectRetries {
			// Trying to reconnect to graphite with given parameters
			sleep := time.Duration(r) * time.Second
			log.Printf("Failed to send metric %v to graphite: %v", filename, err.Error())
			log.Printf("Trying to reconnect and send metric again %v times", connectRetries-r)
			log.Printf("Sleeping for %v", sleep)
			time.Sleep(sleep)
			if err := graphiteConn.Connect(); err != nil {
				log.Printf("Failed to reconnect to graphite: %v", err.Error())
				time.Sleep(time.Second)
				break
			}
		} else {
			break
		}
	}
	if err != nil {
		log.Printf("Failed to send metric %v after %v retries", filename, connectRetries)
		return err
	}
	return err
}

func findWhisperFiles(ch chan string, quit chan int, directory string) {
	visit := func(path string, info os.FileInfo, err error) error {
		if (info != nil) && !info.IsDir() {
			if strings.HasSuffix(path, ".wsp") {
				ch <- path
			}
		}
		return nil
	}
	err := filepath.Walk(directory, visit)
	if err != nil {
		log.Fatal(err)
	}
	close(quit)
}

func worker(ch chan string,
	quit chan int,
	wg *sync.WaitGroup,
	baseDirectory string,
	graphiteHost string,
	graphitePort int,
	graphiteProtocol string,
	fromTs int,
	toTs int,
	connectRetries int,
	rateLimiter *rateLimiter) {
	defer wg.Done()

	graphiteConn, err := graphite.GraphiteFactory(graphiteProtocol, graphiteHost, graphitePort, "")
	if err != nil {
		log.Printf("Failed to connect to graphite host with error: %v", err.Error())
		return
	}

	for {
		select {
		case path := <-ch:
			{

				err := sendWhisperData(path, baseDirectory, graphiteConn, fromTs, toTs, connectRetries, rateLimiter)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("OK: " + path)
				}
			}
		case <-quit:
			return
		}

	}
}
