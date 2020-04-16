Ping-Cli

Submission: Cloudflare Internship Application: Systems

A simple ICMP Echo implementation, based on golang.org/x/net/icmp.
Feature: 
1) IPv4 and IPv6 support
2) Arguement to set TTL and see Time Limit Excedded Messages

It sends ICMP packet(s) and waits for a response. When it's finished, it prints the summary based
of packet sent, received and lost.

Installation:

For installing you only need three external libraries:
1) "golang.org/x/net/icmp"
2) "golang.org/x/net/ipv4"
3) "golang.org/x/net/ipv6"

You can download these using "go get $name_of_library$"

Then build this package using:
`go build ping.go`

Run using:
`./ping hostname` or `sudo ./ping hostname`

This command will create a binary executable file: Using this cli is very easy:

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

	Note: Default TTL is set to 50, if IMCP Time Limit Exceeded Message is Received, it will be printed and TTL will be increased by 1 until first Echo Request is received.
	
	# By default interval is set = 1ms

`

Structure of Code:
Root Directory: "Ping-Cli"
	Main File: `ping.go:` Consits of `main` package and cli code

	pinger: custome build ping package
		/pinger.go: This is where all the code for ping exists.

Troubleshoot:

As the net packages is sending an "unprivileged" ping via UDP, it is recommended to have root privelages
"sudo ./ping cloudflare.com"


Thanks for this wonderful opportunity, I have never coded in Go before and have neither work 
on this type of project. This is just my passion for programming and learning new things which 
made me to attempt this task. This won't be the best submission, but this is my best attempt to learn 
2 things in 1 day and successfully implement basic ping package. Even if I don't get selected for interview I would 
request you to at-least give me feedback on this 
piece of code, so I can improve myself in future.

I have also hosted the code on my github repo: https://github.com/mohit8935/Simple-Go-Ping
email: mohitnihalani.in@gmail.com