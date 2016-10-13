package main

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/immesys/spawnpoint/spawnable"
	"gopkg.in/immesys/bw2bind.v5"
)

type message struct {
	Srcmac  string `msgpack:"srcmac"`
	Srcip   string `msgpack:"srcip"`
	Popid   string `msgpack:"popid"`
	Poptime int64  `msgpack:"poptime"`
	Brtime  int64  `msgpack:"brtime"`
	Rssi    int    `msgpack:"rssi"`
	Lqi     int    `msgpack:"lqi"`
	Payload []byte `msgpack:"payload"`
}

type TypeHandler interface {
	Init(cl *bw2bind.BW2Client, p *spawnable.Params)
	Handle(m *bw2bind.SimpleMessage)
}

var Handlers map[uint16]TypeHandler
var c_forwarded uint64
var c_dropped uint64

func Register(t uint16, h TypeHandler) {
	if Handlers == nil {
		Handlers = make(map[uint16]TypeHandler)
	}
	Handlers[t] = h
}
func main() {
	cl := bw2bind.ConnectOrExit("")
	cl.SetEntityFromEnvironOrExit()
	params := spawnable.GetParamsOrExit()
	signal := params.MustString("signal")
	listenuri := params.MustString("listenuri")
	if !strings.HasSuffix(listenuri, "/") {
		listenuri += "/"
	}
	// Initialize the handlers
	for _, h := range Handlers {
		h.Init(cl, params)
	}
	ch := cl.SubscribeOrExit(&bw2bind.SubscribeParams{
		AutoChain: true,
		URI:       listenuri + "*/s.hamilton/+/i.l7g/signal/" + signal,
	})
	for i := 0; i < 10; i++ {
		go handleIncoming(ch)
	}
	PrintStats()
}

func PrintStats() {
	for {
		time.Sleep(5 * time.Second)
		fmt.Printf("forwarded=%d\n", c_forwarded)
	}
}

func handleIncoming(ch chan *bw2bind.SimpleMessage) {
	for m := range ch {
		po := m.GetOnePODF("2.0.10.1")
		if po == nil {
			fmt.Printf("po mismatch\n")
			continue
		}
		im := message{}
		po.(bw2bind.MsgPackPayloadObject).ValueInto(&im)
		if len(im.Payload) < 2 {
			atomic.AddUint64(&c_dropped, 1)
			continue
		}
		mtype := binary.LittleEndian.Uint16(im.Payload)
		h, ok := Handlers[mtype]
		if ok {
			atomic.AddUint64(&c_forwarded, 1)
			h.Handle(m)
		} else {
			fmt.Printf("no handler found\n")
			atomic.AddUint64(&c_dropped, 1)
		}
	}
}
