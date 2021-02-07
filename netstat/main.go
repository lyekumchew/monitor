package main

import (
	"github.com/Scalingo/go-netstat"
	"log"
)

func (i *multiFlag) String() string {
	return "flags..."
}

var localIPList multiFlag
var etherList multiFlag

func (i *multiFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
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
	flow := getEtherTransmitBytes("ens224")
	log.Println(flow)
}
