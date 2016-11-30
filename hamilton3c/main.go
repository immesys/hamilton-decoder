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

/*
die_temp = 23.59 +273.15
tp_voltage = -34.84e-6
td = die_temp-298.15
s_0 = 5e-14
a1 = 1.75e-3
a2 = -1.678e-5
b0=-2.94e-5
b1 = -5.7e-7
b2 = 4.63e-9
c2 = 13.4
v_os = b0+b1*td + b2*td*td
print ("vos ",v_os)
vobj = tp_voltage
vd = vobj-v_os
fvo = vd + c2*vd*vd
print ("fvo", fvo)
s = s_0*(1+a1*td + a2*td*td)
print ("s", s)
tobj4 = die_temp**4 + fvo/s
print ("tobj4", tobj4)
tobj = tobj4**(1/4.)
tobjc = tobj-273.15
print ("tobjc", tobjc)

*/

/* type 6 is emitted by hamilton-3c motes for temperature
0600 fc00 0000 0000   ec61 fc58   0000 0000 f65954a11e000000 0000
0600 fc00 0000 0000   f065 ac6f   335c 0000 fddb0f881e000000 0000
typedef struct __attribute__((packed)) {
  uint16_t type; 0:2
  uint16_t flags; 2:4 00fc
  uint16_t tmp_die; 4:6
  uint16_t tmp_val; 6:8
  uint16_t hdc_tmp; 8:10
  uint16_t hdc_hum; 10:12
  uint16_t light_lux; 12:14
  uint16_t buttons; 14:16
  uint64_t uptime; 16:24
  uint16_t occup; 24:26
} temp_measurement_t;

0600fc0000000000ec61fc5800000000f65954a11e0000000000

typedef struct __attribute__((packed)) {
  uint16_t type;
  uint16_t flags; //which of the fields below exist
  uint16_t tmp_die;
  uint16_t tmp_val;
  uint16_t hdc_tmp;
  uint16_t hdc_hum;
  uint16_t light_lux;
  uint16_t buttons;
  uint64_t uptime;
  uint16_t occup;
} temp_measurement_t;

#define FLAG_TEMP_HAS_TMP_DIE  0x01
#define FLAG_TEMP_HAS_TMP_VAL  0x02
#define FLAG_TEMP_HAS_HDC_TMP  0x04
#define FLAG_TEMP_HAS_HDC_HUM  0x08
#define FLAG_TEMP_HAS_LUX      0x10
#define FLAG_TEMP_HAS_BUTTONS  0x20
#define FLAG_TEMP_HAS_UPTIME   0x40
#define FLAG_TEMP_HAS_OCCUP    0x80
*/
func (t6 *TempHandler) Handle(sm *bw2bind.SimpleMessage, im *common.Message) {
	if len(im.Payload) < 26 {
		return
	}
	fmt.Printf("payload: %x\n", im.Payload)
	flags := binary.LittleEndian.Uint16(im.Payload[2:])
	dat := make(map[string]interface{})
	if flags&0x01 != 0 { //tmp_die
		raw := binary.LittleEndian.Uint16(im.Payload[4:])
		dat["tp_die_temp"] = float64(int16(raw)>>2) * 0.03125
	}
	if flags&0x02 != 0 { //tmp_val
		raw := binary.LittleEndian.Uint16(im.Payload[6:])
		uv := float64(int16(raw)) * 0.15625
		dat["tp_voltage"] = uv
		fmt.Printf("voltage was %f uv\n", uv)
		v := uv / 1000000.0
		die_temp := dat["tp_die_temp"].(float64) + 273.15
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
		fmt.Println("radiant was", t_obj)
	}
	if flags&0x04 != 0 { //hdc_tmp
		raw := binary.LittleEndian.Uint16(im.Payload[8:])
		fmt.Printf("hdc_tmp raw: %d\n", raw)
		dat["air_temp"] = (float64(raw)/65536)*165 - 40
	}
	if flags&0x08 != 0 { //hdc_rh
		raw := binary.LittleEndian.Uint16(im.Payload[10:])
		fmt.Printf("hdc_rh raw: %d\n", raw)
		dat["air_rh"] = (float64(raw) / 65536) * 100
	}
	if flags&0x10 != 0 { //lux
		raw := binary.LittleEndian.Uint16(im.Payload[12:])
		fmt.Printf("lux raw: %d\n", raw)
		dat["lux"] = math.Pow(10, float64(raw)/(65536.0/5.0))
	}
	if flags&0x20 != 0 { //buttons
		raw := binary.LittleEndian.Uint16(im.Payload[14:])
		fmt.Printf("buttons raw: %d\n", raw)
		dat["button_events"] = raw
	}
	if flags&0x40 != 0 { //uptime
		dat["uptime"] = binary.LittleEndian.Uint64(im.Payload[16:])
		fmt.Printf("uptime raw: %d\n", binary.LittleEndian.Uint64(im.Payload[16:]))
	}
	if flags&0x80 != 0 { //occupancy
		dat["presence"] = float64(binary.LittleEndian.Uint16(im.Payload[24:26])) / 32768
		fmt.Printf("presence raw: %d\n", binary.LittleEndian.Uint16(im.Payload[24:26]))
	}
	if flags&0x07 == 0x07 { //we can do operative temperature
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
	} else {
		fmt.Println("t6 pub")
	}
}
