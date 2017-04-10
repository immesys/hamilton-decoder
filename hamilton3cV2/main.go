package hamilton3c

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"math"
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

	f_uptime := binary.LittleEndian.Uint64(plaintext[0:8])
	f_flags := binary.LittleEndian.Uint16(plaintext[8:10])
	f_acc_x := binary.LittleEndian.Uint16(plaintext[10:12])
	f_acc_y := binary.LittleEndian.Uint16(plaintext[12:14])
	f_acc_z := binary.LittleEndian.Uint16(plaintext[14:16])
	f_mag_x := binary.LittleEndian.Uint16(plaintext[16:18])
	f_mag_y := binary.LittleEndian.Uint16(plaintext[18:20])
	f_mag_z := binary.LittleEndian.Uint16(plaintext[20:22])
	f_tmp_die := binary.LittleEndian.Uint16(plaintext[22:24])
	f_tmp_val := binary.LittleEndian.Uint16(plaintext[24:26])
	f_hdc_tmp := binary.LittleEndian.Uint16(plaintext[26:28])
	f_hdc_rh := binary.LittleEndian.Uint16(plaintext[28:30])
	f_light_lux := binary.LittleEndian.Uint16(plaintext[30:32])
	f_buttons := binary.LittleEndian.Uint16(plaintext[32:34])
	f_occup := binary.LittleEndian.Uint16(plaintext[34:36])
	dat := make(map[string]float64)
	dat["uptime"] = float64(f_uptime)
	if f_flags&(1<<0) != 0 {
		//accel
		dat["acc_x"] = float64(f_acc_x) * 0.244
		dat["acc_y"] = float64(f_acc_y) * 0.244
		dat["acc_z"] = float64(f_acc_z) * 0.244

	}
	if f_flags&(1<<1) != 0 {
		dat["mag_x"] = float64(f_mag_x) * 0.1
		dat["mag_y"] = float64(f_mag_y) * 0.1
		dat["mag_z"] = float64(f_mag_z) * 0.1
	}
	if f_flags&(1<<2) != 0 {
		//TMP
		dat["tp_die_temp"] = float64(int16(f_tmp_die)>>2) * 0.03125
		uv := float64(int16(f_tmp_val)) * 0.15625
		dat["tp_voltage"] = uv
		//		fmt.Printf("voltage was %f uv\n", uv)
		// v := uv / 1000000.0
		// die_temp := dat["tp_die_temp"].(float64) + 273.15
		// tref := 298.15 //K
		// S_0 := 6e-14
		// a1 := 1.75e-3
		// a2 := -1.678e-5
		// S := S_0 * (1 + a1*(die_temp-tref) + a2*(die_temp-tref)*(die_temp-tref))
		// b0 := -2.94e-5
		// b1 := -5.7e-7
		// b2 := 4.63e-9
		// c2 := 13.4
		// v_os := b0 + b1*(die_temp-tref) + b2*(die_temp-tref)*(die_temp-tref)
		// f_vo := v - v_os + c2*(v-v_os)*(v-v_os)
		// t_obj4 := math.Pow(die_temp, 4) + f_vo/S
		// t_obj := math.Pow(t_obj4, 1.0/4)
		// dat["radiant_temp"] = t_obj
		// fmt.Println("radiant was", t_obj)
	}

	if f_flags&(1<<3) != 0 {
		//HDC
		rh := float64(f_hdc_rh) / 100
		t := float64(f_hdc_tmp) / 100
		dat["air_temp"] = t
		dat["air_rh"] = rh
		expn := (17.67 * t) / (t + 243.5)
		dat["air_hum"] = (6.112 * math.Pow(math.E, expn) * rh * 2.1674) / (273.15 + t)
	}
	if f_flags&(1<<4) != 0 {
		//LUX
		dat["lux"] = math.Pow(10, float64(f_light_lux)/(65536.0/5.0))
	}
	if f_flags&(1<<5) != 0 {
		dat["button_events"] = float64(f_buttons)
	}
	if f_flags&(1<<6) != 0 {
		dat["presence"] = float64(f_occup) / 32768
	}

	// if flags&0x07 == 0x07 { //we can do operative temperature
	// 	v := 0.2
	// 	r10v := math.Pow(10*v, 1/2.0)
	// 	dat["operative_temp"] = (dat["air_temp"].(float64) + dat["air_temp"].(float64)*r10v) / (1 + r10v)
	// }
	dat["time"] = float64(im.Brtime)
	npo, _ := bw2bind.CreateMsgPackPayloadObject(bw2bind.FromDotForm("2.0.11.2"), dat)
	err = t.cl.Publish(&bw2bind.PublishParams{
		URI:            fmt.Sprintf("%ss.hamilton/%s/i.temperature/signal/operative", t.baseuri, im.Srcmac),
		PayloadObjects: []bw2bind.PayloadObject{npo},
		AutoChain:      true,
	})
	if err != nil {
		fmt.Printf("tcrypt failed to publish: %v\n", err)
	} else {
		fmt.Println("tcrypt pub")
	}
}
