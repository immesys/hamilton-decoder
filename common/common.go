package common

import (
	"github.com/immesys/spawnpoint/spawnable"
	"gopkg.in/immesys/bw2bind.v5"
)

type TypeHandler interface {
	Init(cl *bw2bind.BW2Client, p *spawnable.Params)
	Handle(sm *bw2bind.SimpleMessage, m *Message)
}

type Message struct {
	Srcmac  string `msgpack:"srcmac"`
	Srcip   string `msgpack:"srcip"`
	Popid   string `msgpack:"popid"`
	Poptime int64  `msgpack:"poptime"`
	Brtime  int64  `msgpack:"brtime"`
	Rssi    int    `msgpack:"rssi"`
	Lqi     int    `msgpack:"lqi"`
	Payload []byte `msgpack:"payload"`
}
