package main

import (
	"fmt"
	"github.com/ti-mo/conntrack"
)

func main() {
	c, _ := conntrack.Dial(nil)
	df, _ := c.DumpFilter(conntrack.Filter{})
	for _, v := range df {
		fmt.Println(v)
	}
}
