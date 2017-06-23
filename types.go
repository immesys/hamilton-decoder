package main

import (
	 "github.com/immesys/hamilton-decoder/hamilton3c"
	hamilton3cV2 "github.com/immesys/hamilton-decoder/hamilton3cV2"
	_ "github.com/immesys/hamilton-decoder/hamilton7"
)

func init() {
	/* type 4 is emitted by hamilton-7 motes
	     typedef struct __attribute__((packed)) {
	     uint16_t type;
	     int8_t flags; //which of the fields below exist, bit 0 is acc_x
	     int16_t acc_x;
	     int16_t acc_y;
	     int16_t acc_z;
	     int32_t temperature; // in C*10000
	     int32_t lux;
	     uint64_t uptime;
	   } measurement_t;
	*/
//	Register(4, &hamilton7.Hamilton7Handler{})

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
	  uint16_t light_lux;
	  uint16_t buttons;
	  uint64_t uptime;
	  uint16_t occup;
	} temp_measurement_t;
	*/
	Register(6, &hamilton3c.TempHandler{})

	//7 is for anemometer

	//8 is for encrypted hamilton
	Register(8, &hamilton3cV2.Handler{})
}
