package dht

import (
	"fmt"
	"net"
	"time"
)

// DHT server
type DHT struct {
	cfg     *Config
	buf     []byte
	exit    chan bool
	conn    *net.UDPConn
	route   *Table
	handler Handler
}

// NewDHT returns DHT
func NewDHT(cfg *Config) *DHT {
	d := &DHT{
		cfg:  cfg,
		buf:  make([]byte, 4096),
		exit: make(chan bool),
	}
	if cfg == nil {
		d.cfg = NewConfig()
	}
	d.route = NewTable(d.ID())
	return d
}

// Run dht server
func (d *DHT) Run(handler Handler) (err error) {
	conn, err := net.ListenPacket("udp", d.cfg.Address)
	if err != nil {
		return
	}
	defer conn.Close()
	d.conn = conn.(*net.UDPConn)
	d.handler = handler
	d.loop()
	return
}

// Exit dht server
func (d *DHT) Exit() {
	close(d.exit)
}

// ID returns dht id
func (d *DHT) ID() *ID {
	if d.cfg.ID == nil {
		d.cfg.ID = NewRandomID()
	}
	return d.cfg.ID
}

// Addr returns dht address
func (d *DHT) Addr() *net.UDPAddr {
	if d.conn != nil {
		return d.conn.LocalAddr().(*net.UDPAddr)
	}
	return nil
}

// Route returns route table
func (d *DHT) Route() *Table {
	return d.route
}

func (d *DHT) loop() {
	msg := make(chan []byte, 1024)
	go d.recvMessage(msg)

	d.initialize()

	cleanup := time.Tick(30 * time.Second)
	for {
		select {
		case <-d.exit:
			goto EXIT
		case <-cleanup:
			d.cleanup()
		case m := <-msg:
			d.handleMessage(m)
		}
	}

EXIT:
	d.unInitialize()
}

func (d *DHT) initialize() {
	data := map[string]interface{}{
		"id":     d.ID().Bytes(),
		"target": d.ID().Bytes(),
	}
	msg := newQueryMessage("fn", "find_node", data)
	for _, route := range d.cfg.Routes {
		addr, err := net.ResolveUDPAddr("udp", route)
		if err == nil {
			d.conn.WriteToUDP(msg, addr)
		}
	}

	if d.handler != nil {
		d.handler.Initialize()
	}
}

func (d *DHT) unInitialize() {
	if d.handler != nil {
		d.handler.UnInitialize()
	}
}

func (d *DHT) cleanup() {
	d.route.Map(func(b *Bucket) bool {
		b.Map(func(n *Node) bool {
			fmt.Println(n)
			return true
		})
		return true
	})
}

func (d *DHT) handleMessage(b []byte) {
	p := decodeMessage(b)
	fmt.Printf("%#v\n", *p)

	switch p.T {
	case "fn":
	}
}

func (d *DHT) ping(n *Node) {
	data := map[string]interface{}{
		"id": d.ID().Bytes(),
	}
	msg := newQueryMessage("pn", "ping", data)
	d.sendMessage([]*Node{n}, msg)
}

func (d *DHT) findNode(id *ID) {
	data := map[string]interface{}{
		"id":     d.ID().Bytes(),
		"target": id.Bytes(),
	}
	msg := newQueryMessage("fn", "find_node", data)
	d.sendMessage(d.route.Lookup(id), msg)
}

func (d *DHT) getPeers(id *ID) {
	data := map[string]interface{}{
		"id":        d.ID().Bytes(),
		"info_hash": id.Bytes(),
	}
	msg := newQueryMessage("gp", "get_peers", data)
	d.sendMessage(d.route.Lookup(id), msg)
}

func (d *DHT) sendMessage(nodes []*Node, msg []byte) {
	for _, node := range nodes {
		d.conn.WriteToUDP(msg, node.Addr())
	}
}

func (d *DHT) recvMessage(msg chan []byte) {
	buf := make([]byte, d.cfg.PacketSize)
	for {
		n, addr, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		_ = n
		_ = addr

		msg <- buf

		select {
		case <-d.exit:
			return
		}
	}
}

func (d *DHT) announcePeer() {
}

func (d *DHT) replyPing(n *Node) {
}

func (d *DHT) replyFindNode(n *Node) {
}

func (d *DHT) replyGetPeers(n *Node) {
}

func (d *DHT) replyAnnouncePeer(n *Node) {
}
