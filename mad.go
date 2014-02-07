// Package to interface with MPEG Audio Decoder
package mad


//#cgo pkg-config: mad
//#include <stdlib.h>
//#include "gomad.h"
import "C"

import(
	"io"
	"unsafe"
	"reflect"
	"fmt"
	"math"
	"bytes"
	"errors"
)

type Decoder struct {
	input io.Reader
	output *bytes.Buffer
	
	buf []byte
	done bool
	
	inputCount int
	outputCount int
	dither *audio_dither
	
	//Audio Information
	SampleRate uint
	Channels uint8
	Bitrate uint64
	
	// C internals
	decoder *C.struct_mad_decoder
}

var ErrNoInput = errors.New("No Input")
var ErrDecoding = errors.New("Decoding")

const(
	BUF_SIZE = 1048576
	) 

// Create new decoder with reader
func New(i io.Reader)(*Decoder,error){
	d := new(Decoder)

	d.input = i
	
	size := int(unsafe.Sizeof(C.struct_mad_decoder{}))
	d.decoder = (*C.struct_mad_decoder)(C.malloc(C.size_t(size)))
	d.output = new(bytes.Buffer)
	d.buf = make([]byte,BUF_SIZE)
	d.done = false
	d.dither = &audio_dither{}
		
	C.setupDecoder(d.decoder,unsafe.Pointer(d))
	
	//Start decoding
	go func(){
		result := C.mad_decoder_run(d.decoder, C.MAD_DECODER_MODE_SYNC);
		if result != 0{
			fmt.Println("Decoder Error")
		}
		C.mad_decoder_finish(d.decoder)	
	}()

	return d,nil
}

// Read decoded bytes from input (buffered)
func (d *Decoder) Read(p []byte) (n int, err error){
	
	if d.inputCount == 0{
		return 0,ErrNoInput
	}else if d.outputCount == 0{
		return 0,ErrDecoding
	}else{
		return d.output.Read(p)
	}
	
	/*
	//Make buffers
	buff := new(bytes.Buffer)
	
	count := 0
	
	//Loop through bytes needed
	for count < (len(p)/2){
		sample,more := <-d.output
		
		if !more{
			fmt.Println("Channel Closed")
			return 0,io.EOF
		}else{			
			//Write to buffer
			binary.Write(buff, binary.LittleEndian, sample)
			count++
		}
	}
		
	copy(p,buff.Bytes())
	
	return len(p),nil
	*/
}
//
// Callbacks
//

//export inputFunc
func inputFunc(data unsafe.Pointer, stream *C.struct_mad_stream)(C.enum_mad_flow){	
	d := (*Decoder)(data)
	
	if d.done{
		return C.MAD_FLOW_STOP
	}
		
	//Calculate bytes to preserve
	bytes_to_preserve := uintptr(unsafe.Pointer(stream.bufend)) - uintptr(unsafe.Pointer(stream.next_frame));
		
	//Copy over remaining bytes of this frame
	var bytesPreserved []byte
	if bytes_to_preserve != 0{
		bytesPreserved = C.GoBytes(unsafe.Pointer(stream.next_frame), C.int(bytes_to_preserve))
		copy(d.buf,bytesPreserved)
	}
				
	if readCount,err := d.input.Read(d.buf[len(bytesPreserved):]); err != nil{
		if err == io.EOF{
			fmt.Println("Ended reading")
			d.done = true
		}else{	
			fmt.Println(err)
			return C.MAD_FLOW_BREAK
		}
	}else{
		fmt.Printf("Streamed %d of %d\n",readCount,len(d.buf))
		
		//Read valid information
		C.mad_stream_buffer(stream,(*C.uchar)(unsafe.Pointer(&d.buf[0])), C.ulong(readCount+len(bytesPreserved)))
		
		d.inputCount += readCount+len(bytesPreserved)
	}
	
	return C.MAD_FLOW_CONTINUE	
}

//export headerFunc
func headerFunc(data unsafe.Pointer, header *C.struct_mad_header)(C.enum_mad_flow){		
	d := (*Decoder)(data)
	
	d.Channels = 2 //Needs to determine this
	d.Bitrate = uint64(header.bitrate)
	d.SampleRate = uint(header.samplerate)
	
	return C.MAD_FLOW_CONTINUE
}

//export outputFunc
func outputFunc(data unsafe.Pointer, header *C.struct_mad_header,pcm *C.struct_mad_pcm)(C.enum_mad_flow){	
	d := (*Decoder)(data)
	        
	length := int(pcm.length)
	
	
	//lSamples := sliceFromArray(unsafe.Pointer(&pcm.samples[0][0]),length*4)
	//rSamples := sliceFromArray(unsafe.Pointer(&pcm.samples[1][0]),length*4)
	
	lSamples :=  pcm.samples[0]
	rSamples :=  pcm.samples[1]
	
	for i:=0;i<length;i++{		
        sampleL := scale(int32(lSamples[i]))*4//audioLinearDither(16,lSamples[i], d.dither);
		d.output.WriteByte(byte(sampleL>>8))
		d.output.WriteByte(byte(sampleL>>0))
		
        sampleR := scale(int32(rSamples[i]))*4//audioLinearDither(16, rSamples[i], d.dither);
		d.output.WriteByte(byte(sampleR>>8))
		d.output.WriteByte(byte(sampleR>>0))
	}
	
	d.outputCount += length

	return C.MAD_FLOW_CONTINUE
} 

//export errorFunc
func errorFunc(data unsafe.Pointer, stream *C.struct_mad_stream, frame *C.struct_mad_frame)(C.enum_mad_flow){	
	fmt.Printf("decoding error 0x%04x (%s)\n",stream.error, C.GoString(C.mad_stream_errorstr(stream)))
	
	if stream.error & 0xff00 != 0{
		return C.MAD_FLOW_CONTINUE;
	}else{
		return C.MAD_FLOW_BREAK;
	}
}

type audio_dither struct{
	err [3]C.mad_fixed_t
	rand C.mad_fixed_t
}

func prng(state uint32)(uint32){
	return (state * 0x0019660d + 0x3c6ef35f) & 0xffffffff
}

func audioLinearDither(bits uint, sample C.mad_fixed_t,dither *audio_dither)(int32){
	var scalebits uint
	var output C.mad_fixed_t
	var mask C.mad_fixed_t
	var random C.mad_fixed_t

	/* noise shape */
	sample += dither.err[0] - dither.err[1] + dither.err[2];

	dither.err[2] = dither.err[1];
	dither.err[1] = dither.err[0] / 2;

	/* bias */
	output = sample + (1 << (C.MAD_F_FRACBITS + 1 - bits - 1));

	scalebits = C.MAD_F_FRACBITS + 1 - bits;
	mask = (1 << scalebits) - 1;

	/* dither */
	random  = C.mad_fixed_t(prng(uint32(dither.rand)))
	output += (random & mask) - (dither.rand & mask);

	dither.rand = random;

	/* clip */
	if (output > math.MaxInt16) {
		output = math.MaxInt16;

		if (sample > math.MaxInt16){
			sample = math.MaxInt16;
		}
	}else if (output < math.MinInt16) {
		output = math.MinInt16;

		if (sample < math.MinInt16){
			sample = math.MinInt16;
		}
	}

	/* quantize */
	output &= ^mask;

	/* error feedback */
	dither.err[0] = sample - output;

	/* scale */
	return int32(output >> scalebits);	
}

func scale(i int32)(int16){
	if i >= math.MaxInt16{
		return math.MaxInt16
	}else if i <= math.MinInt16{
		return math.MinInt16
	}else{
		return int16(i >> (uint(C.MAD_F_FRACBITS) - 15))
	}
}

func sliceFromArray(p unsafe.Pointer,length int)([]byte){
        var theGoSlice []byte
        sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&theGoSlice)))
        sliceHeader.Cap = length
        sliceHeader.Len = length
        sliceHeader.Data = uintptr(p)
	
	return theGoSlice
}