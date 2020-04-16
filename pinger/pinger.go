package pinger

/*


**/
import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	timeSliceLength      = 8
	ICMPv4protocolNumber = 1
	ICMPv6protocolNumber = 58
)

var (
	ipv4Proto = map[string]string{"ip": "ip4:icmp", "udp": "udp4"}
	ipv6Proto = map[string]string{"ip": "ip6:ipv6-icmp", "udp": "udp6"}
)

/*
	Pinger Object
*/
type Pinger struct {
	ipaddr          *net.IPAddr
	addr            string
	hasIPv4         bool
	hasIPv6         bool
	network         string
	source          string
	done            bool
	finish          chan bool
	rtts            []time.Duration
	seq             int
	packageSent     int
	packageReceived int
	id              int
	interval        time.Duration
	ttl             int
}

type packet struct {
	bytes  []byte
	nbytes int
	ttl    int
	addr   net.Addr
}

/*
Check if IP Address is ipv4
*/
func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

/*
Check if IP Address is ipv6
*/
func isIPv6(ip net.IP) bool {
	return len(ip) == net.IPv6len
}

func (pinger *Pinger) SetFlags(interval time.Duration, privelage bool, TTL int) {
	pinger.interval = interval
	if privelage {
		pinger.network = "udp"
	}
	pinger.ttl = TTL
}

/*
Start the Ping
*/
func (pinger *Pinger) Start() {
	var conn *icmp.PacketConn
	if pinger.hasIPv4 {
		if conn = pinger.listen(ipv4Proto[pinger.network]); conn == nil {
			fmt.Println("Error Starting Connection with the host: Try with: sudo ./ping hostname")
			return
		}
		conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
		conn.IPv4PacketConn().SetTTL(pinger.ttl)
	} else {
		if conn = pinger.listen(ipv6Proto[pinger.network]); conn == nil {
			return
		}
		conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
		conn.IPv6PacketConn().SetHopLimit(pinger.ttl + 1)
	}
	defer conn.Close()

	var waitGroup sync.WaitGroup

	pktChannel := make(chan *packet, 5) // Channel for Storing packets
	defer close(pktChannel)
	waitGroup.Add(1)
	go pinger.receiveImcp(conn, pktChannel, &waitGroup) // Start Receiving Packets

	interval := time.NewTicker(pinger.interval)
	defer interval.Stop()
	fmt.Printf("Starting pinging to %s", pinger.ipaddr.IP)
	fmt.Println()
	defer pinger.printSummary()
	for {

		select {
		case <-pinger.finish:
			waitGroup.Wait()
			return
		case <-interval.C:
			// Sent ICMP request and try receiving ICMP reuqest
			err := pinger.sendICMP(conn)
			if err != nil {
				fmt.Println("FATAL: ", err.Error())
			}
		case pkt := <-pktChannel:
			err := pinger.processPacket(pkt, conn)
			if err != nil {
				fmt.Println("FATAL: ", err.Error())
			}
		}
	}
}

/*
Code for initilizing the pinger
*/
func CreatePinger(host string) (*Pinger, error) {
	ipaddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return nil, err
	}
	var ipv4, ipv6 bool = false, false

	if isIPv4(ipaddr.IP) {
		ipv4 = true
	} else if isIPv6(ipaddr.IP) {
		ipv6 = true
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Pinger{
		ipaddr:          ipaddr,
		addr:            host,
		hasIPv4:         ipv4,
		hasIPv6:         ipv6,
		network:         "ip",
		finish:          make(chan bool),
		seq:             0,
		packageSent:     0,
		packageReceived: 0,
		id:              r.Intn(math.MaxInt16),
	}, nil
}

/*
Start Listening at the Port
*/
func (pinger *Pinger) listen(netProto string) *icmp.PacketConn {
	conn, err := icmp.ListenPacket(netProto, pinger.source)
	if err != nil {
		close(pinger.finish)
		return nil
	}
	return conn
}

/*
Code for receiving packets from the destination
*/
func (pinger *Pinger) receiveImcp(
	conn *icmp.PacketConn,
	pktChannel chan<- *packet,
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()
	for {
		select {
		case <-pinger.finish:
			return
		default:
			bytes := make([]byte, 512) // Buffer for storing data
			conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
			var n, ttl int
			var err error
			var src net.Addr
			if pinger.hasIPv4 {
				var cm *ipv4.ControlMessage
				n, cm, src, err = conn.IPv4PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.TTL
				}
			} else {
				var cm *ipv6.ControlMessage
				n, cm, src, err = conn.IPv6PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.HopLimit
				}
			}

			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						continue
					} else {
						close(pinger.finish)
						return
					}
				}
			}
			pktChannel <- &packet{bytes: bytes, nbytes: n, ttl: ttl, addr: src}
		}
	}
}

/*
Function to process the read packer.
1) Parse the message from the bytes received based on the type of message
2) Checks if it is the valid message type
3) Prinf information about the message
*/
func (pinger *Pinger) processPacket(recv *packet, conn *icmp.PacketConn) error {
	receivedAt := time.Now()
	var proto int
	if pinger.hasIPv4 {
		proto = ICMPv4protocolNumber
	} else {
		proto = ICMPv6protocolNumber
	}

	var message *icmp.Message
	var err error

	message, err = icmp.ParseMessage(proto, recv.bytes)
	if err != nil {
		return err
	}

	var rtt time.Duration
	switch pkt := message.Body.(type) {
	case *icmp.Echo:
		if pkt.ID == pinger.id {
			rtt = receivedAt.Sub((bytesToTime(pkt.Data[:timeSliceLength])))
			fmt.Printf("%d bytes from %s: icmp_seq=%d ttl=%d time=%v\n",
				recv.nbytes, pinger.ipaddr.IP, pkt.Seq, recv.ttl, rtt)
			pinger.packageReceived++
		}
	case *icmp.TimeExceeded:
		if int64(binary.BigEndian.Uint16(pkt.Data[24:26])) == int64(pinger.id) {
			var ttl int
			if pinger.hasIPv4 {
				ttl, _ = conn.IPv4PacketConn().TTL()
				conn.IPv4PacketConn().SetTTL(ttl + 1)
			} else {
				ttl, _ = conn.IPv6PacketConn().HopLimit()
				conn.IPv6PacketConn().SetHopLimit(ttl + 1)

			}
			fmt.Printf("Time Limit Exceeded:  %d hop: %s \n", ttl, recv.addr)
		}
	default:
		return fmt.Errorf("invalid ICMP echo reply")
	}
	pinger.rtts = append(pinger.rtts, rtt)
	return nil
}

/*
Create Data For sending as a payload
*/
func createData() []byte {
	t := time.Now()
	nsec := t.UnixNano()
	timeBytes := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		timeBytes[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return timeBytes
}

/*
Send ICMP message to destination address
*/
func (pinger *Pinger) sendICMP(conn *icmp.PacketConn) error {
	var proto icmp.Type
	proto = ipv4.ICMPTypeEcho // Type := echo.ipv4 (default)
	if pinger.hasIPv6 {
		proto = ipv6.ICMPTypeEchoRequest
	}

	var dst net.Addr = pinger.ipaddr

	// Id network is privelaged
	if pinger.network == "udp" {
		dst = &net.UDPAddr{IP: pinger.ipaddr.IP, Zone: pinger.ipaddr.Zone}
	}

	// Construct message, with pinger id, seq
	messageBytes, err := (&icmp.Message{
		Type: proto,
		Code: 0,
		Body: &icmp.Echo{
			ID:   pinger.id,
			Seq:  pinger.seq,
			Data: createData(),
		},
	}).Marshal(nil)

	if err != nil {
		return err
	}
	pinger.seq++

	// Write the message to the destination
	for {
		if _, err := conn.WriteTo(messageBytes, dst); err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
		}
		pinger.packageSent++

		break
	}

	return nil
}

/*
Function to print after executing the ping.
*/
func (pinger *Pinger) printSummary() {
	packetsLoss := float64(pinger.packageSent-pinger.packageReceived) / float64(pinger.packageSent) * 100
	fmt.Printf("Summary for the %s ", pinger.ipaddr)
	fmt.Println()
	fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
		pinger.packageSent, pinger.packageReceived, packetsLoss)
}

/*
Function to convert bytes to Int
*/
func bytesToInt(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

/*
Function to stop ping
*/
func (pinger *Pinger) Stop() {
	close(pinger.finish)
}

/*
Extract Message from packet message body
*/
func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}
