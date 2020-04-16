package main

import (
	ping "Ping-Cli/pinger"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

var usage = `
Please Enter Host Name or Ip Address
Usage:
	ping [-i interval] [-T TTL Limit][--p] host
	
	#example pinging to cloudflare.com:
	./ping www.cloudflare.com

	# For privelage use "--p" flag:
	sudo ./ping --p www.cloudflare.com

	# For Interval use "-p" flag:
	sudo ./ping -i 500ms www.cloudflare.com

	## For TTL use "-T" flag:
	sudo ./ping -T 10 www.cloudflare.com

	Note: Default TTL is set to 50, if IMCP Time Limit Exceeded Message is Received, it will be printed and TTL will be increased by 1 untill first Echo Request is received.
	
	# By default interval is set = 1ms

`

func main() {

	interval := flag.Duration("i", time.Second, "Using -i for intervals")
	privelage := flag.Bool("p", false, "Flag for privelage access")
	ttl := flag.Int("T",60,"Flag for TTL limit")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Printf(usage)
		return
	}

	host := flag.Arg(0)

	// Create A new pinger object
	pinger, err := ping.CreatePinger(host)

	// Check if host name or ip address is valid
	if err != nil {
		fmt.Print("Invalid Host name or IP Address")
		return
	}

	// For exit keep looking for Ctrl - C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			pinger.Stop()
		}
	}()

	// Set privelage if passed with command line arguements
	//pinger.SetPrivlage(*privelaged)
	pinger.SetFlags(*interval, *privelage,*ttl)
	// Fun Begins
	pinger.Start()

}
