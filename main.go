package main

import (
	"flag"
	"github.com/Scalingo/go-netstat"
	"github.com/go-co-op/gocron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sinlov/qqwry-golang/qqwry"
	"github.com/ti-mo/conntrack"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type multiFlag []string

func (i *multiFlag) String() string {
	return "flags..."
}

var ipPortList multiFlag
var ipPortListFormat [][]string
var etherList multiFlag

func (i *multiFlag) Set(value string) error {
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

	// traffic
	traffic = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "monitor_traffic_total",
		Help: "Monitor",
	}, []string{"ether"})
)

//var needRecordPort []uint16
//var localIP string
//var etherName string
var trafficBytes map[string]uint64

func init() {
	trafficBytes = make(map[string]uint64)

	// qqwry
	var datPath = "qqwry.dat"
	qqwry.DatData.FilePath = datPath
	init := qqwry.DatData.InitDatFile()
	if v, ok := init.(error); ok {
		if v != nil {
			log.Printf("init InitDatFile error %s", v)
		}
	}

	// format: 127.0.0.1:4021|127.0.0.2:4000
	flag.Var(&ipPortList, "ipPort", "the monitoring port.")
	flag.Var(&etherList, "ether", "Ether Name.")

	ipPortListFormat = [][]string{}
}

func job() {
	c, err := conntrack.Dial(nil)

	if err != nil {
		log.Println("conntrack error!")
		return
	}

	df, err := c.DumpFilter(conntrack.Filter{})
	if err != nil {
		log.Fatal(err)
	}

	for _, ipPortList := range ipPortListFormat {
		freq := make(map[string]int)
		port := 0

		// 过滤包, 只要国内到机器的包
		for _, f := range df {
			for _, ipPort := range ipPortList {
				ipPortSplit := strings.Split(ipPort, ":")
				if len(ipPortSplit) != 2 {
					log.Println("ip port split error!")
					return
				}
				ip := ipPortSplit[0]
				port, _ = strconv.Atoi(ipPortSplit[1])
				if port == 0 {
					log.Println("port strcov.Atoi error!")
					return
				}
				if f.TupleOrig.Proto.DestinationPort == uint16(port) && f.TupleOrig.IP.DestinationAddress.String() == ip {
					freq[f.TupleOrig.IP.SourceAddress.String()]++
				}
			}
		}

		serverForwardIP.WithLabelValues(strconv.Itoa(port)).Add(float64(len(freq)))

		for ip, connSum := range freq {
			res := qqwry.NewQQwry().SearchByIPv4(ip)
			var country string
			if res.Err == "" {
				country = res.Country
			} else {
				country = "无"
			}
			serverForwardConnections.WithLabelValues(strconv.Itoa(port), ip, country).Add(float64(connSum))
		}
	}

	// stats traffic
	for _, etherName := range etherList {
		last := trafficBytes[etherName]
		now := getEtherTransmitBytes(etherName)
		if now < last {
			log.Println("ether traffic stat error!")
		} else {
			increase := now - last
			trafficBytes[etherName] = now
			traffic.WithLabelValues(etherName).Add(float64(increase))
		}
	}
}

func getEtherTransmitBytes(etherName string) uint64 {
	var flow uint64
	stat, _ := netstat.Stats()
	for _, ether := range stat {
		if ether.Interface == etherName {
			flow = ether.Transmit.Bytes
		}
	}
	return flow
}

func main() {
	flag.Parse()

	if len(ipPortList) == 0 {
		log.Fatal("Local IP Address is empty, error!")
	}

	if len(etherList) == 0 {
		log.Fatal("Ether Num is 0, error!")
	}

	log.Println(etherList)

	// init ether traffic
	for _, etherName := range etherList {
		flow := getEtherTransmitBytes(etherName)
		if flow == uint64(0) {
			log.Fatal("ether error!", etherName)
		} else {
			trafficBytes[etherName] = flow
		}
	}

	log.Println(ipPortList)

	for _, v := range ipPortList {
		ipPortListFormat = append(ipPortListFormat, strings.Split(v, "|"))
	}

	for k, v := range ipPortListFormat {
		log.Println(k, "ipPortList", v)
	}

	// setup cronjob
	cron := gocron.NewScheduler(time.UTC)
	// do job
	_, _ = cron.Every(15).Second().Do(job)
	// start cron job
	cron.StartAsync()

	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe("127.0.0.1:2112", nil)
}
