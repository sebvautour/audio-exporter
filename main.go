package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var promHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "audio_pitch",
	Buckets: []float64{100, 200, 300, 500, 1000, 2000, 30000},
})

var promMaxFreq = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "audio_max_freq",
})

func main() {
	go func() {
		reg := prometheus.NewRegistry()
		reg.MustRegister(promHistogram, promMaxFreq)

		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		log.Fatal(http.ListenAndServe(":8888", nil))
	}()

	cmd := exec.Command("python3", "audio.py")
	cmd.Stderr = os.Stderr
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		renew := time.Now().Add(time.Second)
		max := float64(0)
		for scanner.Scan() {
			n, err := strconv.ParseFloat(scanner.Text(), 64)
			if err != nil {
				continue
			}
			promHistogram.Observe(n)
			if n > max {
				max = n
				promMaxFreq.Set(n)
			}

			if renew.Before(time.Now()) {
				fmt.Println("renew time")
				renew = time.Now().Add(time.Second * 5)
				max = 0
			}

			fmt.Printf("cur: %v, max: %v\n", n, max)
		}
		fmt.Println("done")
	}()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
