package main

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/immesys/spawnpoint/spawnable"
	"github.com/pborman/uuid"
	"gopkg.in/immesys/bw2bind.v5"
)

func init() {
	Register(4, &Type4Handler{})
}

var Type4Namespace = uuid.Parse("821c0592-9316-4716-b4f4-d2c0dc436dab")

type Type4Handler struct {
	cl      *bw2bind.BW2Client
	p       *spawnable.Params
	baseuri string
}

type T4Message struct {
	UUID  string  `msgpack:"UUID"`
	Time  int64   `msgpack:"Time"`
	Value float64 `msgpack:"Value"`
}

func (t4 *Type4Handler) Init(cl *bw2bind.BW2Client, p *spawnable.Params) {
	t4.cl = cl
	t4.p = p
	t4.baseuri = p.MustString("type4_base")
	if !strings.HasSuffix(t4.baseuri, "/") {
		t4.baseuri += "/"
	}
}
func (t4 *Type4Handler) Handle(m *bw2bind.SimpleMessage) {
	po := m.GetOnePODF("2.0.10.1")
	if po == nil {
		fmt.Printf("po mismatch\n")
		return
	}
	im := message{}
	po.(bw2bind.MsgPackPayloadObject).ValueInto(&im)
	if len(im.Payload) < 2 {
		atomic.AddUint64(&c_dropped, 1)
		return
	}
	tempi := binary.LittleEndian.Uint32(im.Payload[9:])
	tempf := float64(tempi) / 10000.
	obj := T4Message{
		Time:  im.Brtime,
		Value: tempf,
		UUID:  uuid.NewSHA1(Type4Namespace, []byte(im.Srcmac)).String(),
	}
	npo, _ := bw2bind.CreateMsgPackPayloadObject(bw2bind.PONumTimeseriesReading, obj)
	err := t4.cl.Publish(&bw2bind.PublishParams{
		URI:            fmt.Sprintf("%ss.hamilton/%s/i.temperature/signal/temperature", t4.baseuri, im.Srcmac),
		PayloadObjects: []bw2bind.PayloadObject{npo},
		AutoChain:      true,
	})
	if err != nil {
		fmt.Printf("failed to publish: %v\n", err)
	}
}
