//package main
//
//import (
//	"fmt"
//	"github.com/coreos/go-iptables/iptables"
//	"log"
//	"strings"
//)
//
//func main() {
//	port := "20297"
//	ipt, err := iptables.New()
//	if err != nil {
//		log.Print("error")
//		return
//	}
//	ruleList, err := ipt.StructuredStats("filter", "FOWARD")
//	if err != nil {
//		fmt.Printf("ListChains of Initial failed: %v", err)
//	}
//
//	var bytesSum uint64
//	for _, rule := range ruleList {
//		filter := "dpt:" + port
//		if strings.Contains(rule.Options, filter) {
//			log.Println(rule.Options)
//			bytesSum += rule.Bytes
//		}
//	}
//	log.Println(bytesSum/1024/1024, "MB")
//}
