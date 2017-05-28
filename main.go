package main

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
    "os"

	"github.com/immesys/hamilton-decoder/common"

	"github.com/immesys/spawnpoint/spawnable"
	"gopkg.in/immesys/bw2bind.v5"
)

var Handlers map[uint16]common.TypeHandler
var c_forwarded uint64
var c_dropped uint64

func Register(t uint16, h common.TypeHandler) {
	if Handlers == nil {
		Handlers = make(map[uint16]common.TypeHandler)
	}
	Handlers[t] = h
}

func main() {
    if len(os.Args) != 2 {
        fmt.Printf("usage: decoder <paramsfile>\n")
        os.Exit(1)
    } 
	cl := bw2bind.ConnectOrExit("")
	cl.SetEntityFromEnvironOrExit()
	params, err := spawnable.GetParamsFile(os.Args[1])
    if err != nil {
        panic(err)
    }
	signal := params.MustString("signal")
	listenuri := params.MustString("listenuri")
	if !strings.HasSuffix(listenuri, "/") {
		listenuri += "/"
	}
	// Initialize the handlers
	for _, h := range Handlers {
		h.Init(cl, params)
	}
	realuri := listenuri + "*/s.hamilton/+/i.l7g/signal/" + signal
	fmt.Println("Beginning to listen on ", realuri)
	ch := cl.SubscribeOrExit(&bw2bind.SubscribeParams{
		AutoChain: true,
		URI:       realuri,
	})
	for i := 0; i < 10; i++ {
		go handleIncoming(ch)
	}
	PrintStats()
}

func PrintStats() {
	for {
		time.Sleep(5 * time.Second)
		fmt.Printf("forwarded=%d dropped=%d\n", c_forwarded, c_dropped)
	}
}

func handleIncoming(ch chan *bw2bind.SimpleMessage) {
	for m := range ch {
		po := m.GetOnePODF("2.0.10.1")
		if po == nil {
			fmt.Printf("po mismatch\n")
			continue
		}
		im := common.Message{}
		po.(bw2bind.MsgPackPayloadObject).ValueInto(&im)
		if len(im.Payload) < 2 {
			atomic.AddUint64(&c_dropped, 1)
			continue
		}
		mtype := binary.LittleEndian.Uint16(im.Payload)
		h, ok := Handlers[mtype]
		if ok {
			atomic.AddUint64(&c_forwarded, 1)
			h.Handle(m, &im)
		} else {
			atomic.AddUint64(&c_dropped, 1)
		}
	}
}
