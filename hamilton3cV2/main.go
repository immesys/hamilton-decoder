package hamilton3c

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/immesys/hamilton-decoder/common"
	"github.com/immesys/hcr"
	"github.com/immesys/spawnpoint/spawnable"
	"gopkg.in/immesys/bw2bind.v5"
)

type Handler struct {
	cl        *bw2bind.BW2Client
	p         *spawnable.Params
	baseuri   string
	depsecret string
}

func (t *Handler) Init(cl *bw2bind.BW2Client, p *spawnable.Params) {
	t.cl = cl
	t.p = p
	t.baseuri = p.MustString("type8_base")
	if !strings.HasSuffix(t.baseuri, "/") {
		t.baseuri += "/"
	}
	t.depsecret = p.MustString("type8_deployment_rsecret")
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
#define AES_SKIP_START_BYTES 4

typedef struct __attribute__((packed,aligned(4))) {
  0   uint16_t type;
  2   uint16_t serial;
  //From below is encrypted
  //We use a zero IV, so it is important that the first AES block
  //is completely unique, which is why we include uptime.
  //It is expected that hamilton nodes never reboot and that
  //uptime is strictly monotonic
  4   uint64_t uptime;
  12  uint16_t flags; //which of the fields below exist
  14  int16_t acc_x;
  16  int16_t acc_y;
  18  int16_t acc_z;
  20  int16_t mag_x;
  22  int16_t mag_y;
  24  int16_t mag_z;
  26  uint16_t tmp_die;
  28  uint16_t tmp_val;
  30  int16_t hdc_tmp;
  32  int16_t hdc_hum;
  34  uint16_t light_lux;
  36  uint16_t buttons;
  38  uint16_t occup;
  40  uint32_t reserved1;
  44  uint32_t reserved2;
  48  uint32_t reserved3;
  52
} ham7c_t;
*/
func (t *Handler) Handle(sm *bw2bind.SimpleMessage, im *common.Message) {
	fmt.Printf("payload length is %d\n", len(im.Payload))
	if len(im.Payload) != 52 {
		fmt.Printf("dropping hamilton-3c-v2 packet due to length mismatch: expected 52 got %d\n", len(im.Payload))
	}
	serial := binary.LittleEndian.Uint16(im.Payload[2:])
	moteinfo, err := hcr.GetMoteInfo(context.Background(), int(serial), t.depsecret)
	if err != nil {
		fmt.Printf("[%04x/%s] dropping hamilton-3c-v2 packet due to HCR error: %v\n", serial, t.depsecret, err)
		return
	}
	block, err := aes.NewCipher(moteinfo.AESK[:16])
	if err != nil {
		fmt.Printf("[%04x/%s] dropping hamilton-3c-v2 packet due to error: %v\n", serial, t.depsecret, err)
		return
	}
	iv := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F}
	dce := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(im.Payload)-4)
	dce.CryptBlocks(plaintext, im.Payload[4:])
	for i := 40; i < 48; i++ {
		if plaintext[i] != 0 {
			fmt.Printf("[%04x/%s] dropping hamilton-3c-v2 packet because it looks like AES key is wrong\n", serial, t.depsecret)
			return
		}
	}

	flags := binary.LittleEndian.Uint16
	fmt.Printf("Payload %x\n", plaintext)
	_ = dce
	return
	/*
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
		}*/
}
