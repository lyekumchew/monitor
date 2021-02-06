package main

import (
	"flag"
	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sinlov/qqwry-golang/qqwry"
	"github.com/ti-mo/conntrack"
	"log"
	"net/http"
	"strconv"
	"time"
)

type portsFlag []string

func (i *portsFlag) String() string {
	return "ports..."
}

var ports portsFlag

func (i *portsFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	// 总的 connections, 不区分 ip
	serverForwardConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "monitor_forward_connections_total",
		Help: "Monitor",
	}, []string{"port", "ip", "country"})

	// 当前 IP 数量
	serverForwardIP = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "monitor_forward_ip_total",
		Help: "Monitor",
	}, []string{"port"})
)

var needRecordPort []uint16
var localIP string

func init() {
	// qqwry
	var datPath = "qqwry.dat"
	qqwry.DatData.FilePath = datPath
	init := qqwry.DatData.InitDatFile()
	if v, ok := init.(error); ok {
		if v != nil {
			log.Printf("init InitDatFile error %s", v)
		}
	}

	flag.Var(&ports, "port", "the monitoring port.")
	flag.StringVar(&localIP, "ip", "", "local IP Address.")
}

func job() {
	c, err := conntrack.Dial(nil)

	df, err := c.DumpFilter(conntrack.Filter{})
	if err != nil {
		log.Fatal(err)
	}

	for _, port := range needRecordPort {
		freq := make(map[string]int)

		// 过滤包, 只要国内到机器的包
		for _, f := range df {
			if f.TupleOrig.Proto.DestinationPort == port && f.TupleOrig.IP.DestinationAddress.String() == localIP {
				freq[f.TupleOrig.IP.SourceAddress.String()]++
			}
		}

		serverForwardIP.WithLabelValues(strconv.Itoa(int(port))).Add(float64(len(freq)))

		for ip, connSum := range freq {
			res := qqwry.NewQQwry().SearchByIPv4(ip)
			var country string
			if res.Err == "" {
				country = res.Country
			} else {
				country = "无"
			}
			serverForwardConnections.WithLabelValues(strconv.Itoa(int(port)), ip, country).Add(float64(connSum))
		}
	}
}

func main() {
	flag.Parse()

	for _, port := range ports {
		v, err := strconv.Atoi(port)
		if err != nil {
			panic("port error!")
		}
		if uint16(v) < 0 || uint16(v) > 65535 {
			panic("port error!")
		}
		needRecordPort = append(needRecordPort, uint16(v))
	}

	if len(needRecordPort) == 0 {
		log.Fatal("Port num == 0, error!")
	}

	if localIP == "" {
		log.Fatal("Local IP Address is empty, error!")
	}

	log.Print("The monitoring port is: ", needRecordPort)
	log.Print("Local IP Address is: ", localIP)

	// setup cronjob
	cron := gocron.NewScheduler(time.UTC)
	// do job
	_, _ = cron.Every(15).Second().Do(job)
	// start cron job
	cron.StartAsync()

	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":2112", nil)
}
