package hamilton3c

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/immesys/hamilton-decoder/common"
	"github.com/immesys/spawnpoint/spawnable"
	"gopkg.in/immesys/bw2bind.v5"
)

/* type 5 is emitted by hamilton-3c motes for orientation
typedef struct __attribute__((packed)) {
  uint16_t type;
  int16_t flags; //which of the fields below exist, bit 0 is acc_x
  int16_t acc_x;
  int16_t acc_y;
  int16_t acc_z;
  int16_t mag_x;
  int16_t mag_y;
  int16_t mag_z;
  uint64_t uptime;
} mag_acc_measurement_t;
*/
//	Register(5, &hamilton3c.MagAccHandler{})

/* type 6 is emitted by hamilton-3c motes for temperature
typedef struct __attribute__((packed)) {
  uint16_t type;
  int16_t flags; //which of the fields below exist
  uint16_t tmp_die;
  uint16_t tmp_val;
  uint16_t hdc_tmp;
  uint16_t hdc_hum;
  int16_t light_lux;
  int16_t buttons;
  uint64_t uptime;
} temp_measurement_t;
*/
//	Register(6, &hamilton3c.TempHandler{})

type MagAccHandler struct {
	cl      *bw2bind.BW2Client
	p       *spawnable.Params
	baseuri string
}

type TempHandler struct {
	cl      *bw2bind.BW2Client
	p       *spawnable.Params
	baseuri string
}

func (t5 *MagAccHandler) Init(cl *bw2bind.BW2Client, p *spawnable.Params) {
	t5.cl = cl
	t5.p = p
	t5.baseuri = p.MustString("type5_base")
	if !strings.HasSuffix(t5.baseuri, "/") {
		t5.baseuri += "/"
	}
}
func (t6 *TempHandler) Init(cl *bw2bind.BW2Client, p *spawnable.Params) {
	t6.cl = cl
	t6.p = p
	t6.baseuri = p.MustString("type6_base")
	if !strings.HasSuffix(t6.baseuri, "/") {
		t6.baseuri += "/"
	}
}

func (t5 *MagAccHandler) Handle(sm *bw2bind.SimpleMessage, im *common.Message) {
	//TODO when we know how the fields are encoded
	if len(im.Payload) < 24 {
		return
	}
	return
}

func (t6 *TempHandler) Handle(sm *bw2bind.SimpleMessage, im *common.Message) {
	if len(im.Payload) < 24 {
		return
	}
	flags := binary.LittleEndian.Uint16(im.Payload[2:])
	dat := make(map[string]interface{})
	if flags&0x01 != 0 { //tmp_die
		raw := binary.LittleEndian.Uint16(im.Payload[4:])
		dat["tp_die_temp"] = float64(int16(raw)>>2) * 0.03125
	}
	if flags&0x02 != 0 { //tmp_val
		raw := binary.LittleEndian.Uint16(im.Payload[6:])
		mv := float64(int16(raw)) * 0.15625 //In millivolts
		dat["tp_voltage"] = mv
		v := mv / 1000.0
		die_temp := dat["tp_die_temp"].(float64)
		tref := 298.15 //K
		S_0 := 6e-14
		a1 := 1.75e-3
		a2 := -1.678e-5
		S := S_0 * (1 + a1*(die_temp-tref) + a2*(die_temp-tref)*(die_temp-tref))
		b0 := -2.94e-5
		b1 := -5.7e-7
		b2 := 4.63e-9
		c2 := 13.4
		v_os := b0 + b1*(die_temp-tref) + b2*(die_temp-tref)*(die_temp-tref)
		f_vo := v - v_os + c2*(v-v_os)*(v-v_os)
		t_obj4 := math.Pow(die_temp, 4) + f_vo/S
		t_obj := math.Pow(t_obj4, 1.0/4)
		dat["radiant_temp"] = t_obj
	}
	if flags&0x04 != 0 { //hdc_tmp
		raw := binary.LittleEndian.Uint16(im.Payload[8:])
		dat["air_temp"] = (float64(int16(raw))/65536)*165 - 40
	}
	if flags&0x08 != 0 { //hdc_rh
		raw := binary.LittleEndian.Uint16(im.Payload[10:])
		dat["air_rh"] = (float64(int16(raw)) / 65536) * 100
	}
	if flags&0x40 != 0 { //uptime
		dat["uptime"] = binary.LittleEndian.Uint64(im.Payload[16:])
	}
	if flags&0x07 != 0 { //we can do operative temperature
		v := 0.2
		r10v := math.Pow(10*v, 1/2.0)
		dat["operative_temp"] = (dat["air_temp"].(float64) + dat["air_temp"].(float64)*r10v) / (1 + r10v)
	}
	dat["time"] = im.Brtime
	npo, _ := bw2bind.CreateMsgPackPayloadObject(bw2bind.FromDotForm("2.0.11.2"), dat)
	err := t6.cl.Publish(&bw2bind.PublishParams{
		URI:            fmt.Sprintf("%ss.hamilton/%s/i.temperature/signal/operative", t6.baseuri, im.Srcmac),
		PayloadObjects: []bw2bind.PayloadObject{npo},
		AutoChain:      true,
	})
	if err != nil {
		fmt.Printf("t6 failed to publish: %v\n", err)
	}
}
