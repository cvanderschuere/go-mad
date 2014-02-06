// Package to interface with MPEG Audio Decoder
package mad

/*
	#cgo pkg-config: mad
	#include <stdlib.h>
	#include "gomad.h"
*/
import "C"

import(
	"io"
	"unsafe"
	"fmt"
	"bytes"
	"encoding/binary"
	"math"
)

type Decoder struct {
	input io.Reader
	b *bytes.Buffer
	
	//Audio Information
	SampleRate uint
	Channels uint8
	Bitrate uint64
	
	// C internals
	decoder *C.struct_mad_decoder
}

// Create new decoder with reader
func New(i io.Reader)(*Decoder,error){
	d := new(Decoder)
	

	d.input = i
	
	size := int(unsafe.Sizeof(C.struct_mad_decoder{}))
	d.decoder = (*C.struct_mad_decoder)(C.malloc(C.size_t(size)))
	//struct mad_decoder *decoder = malloc(sizeof(struct mad_decoder));	
	d.b = new(bytes.Buffer)
		
	C.setupDecoder(d.decoder,unsafe.Pointer(d))
	
	/*
	//Start decoding
	go func(){
		fmt.Println("Starting decoder:")
		result := C.mad_decoder_run(d.decoder, C.MAD_DECODER_MODE_SYNC);
		fmt.Println("Result: ")
		fmt.Println(result)
		
		//C.mad_decoder_finish(d.decoder);	
	}()
	*/

	return d,nil
}

func (d *Decoder) Decode(){
	result := C.mad_decoder_run(d.decoder, C.MAD_DECODER_MODE_SYNC);
	fmt.Println("Result: ")
	fmt.Println(result)
	
	C.mad_decoder_finish(d.decoder)	
}

// Read decoded bytes from input (buffered)
func (d *Decoder) Read(p []byte) (n int, err error){
	return d.b.Read(p)
}

//
// Callbacks
//

//export inputFunc
func inputFunc(data unsafe.Pointer, stream *C.struct_mad_stream)(C.enum_mad_flow){
	fmt.Println("input")
	
	d := (*Decoder)(data)
		
	var buffer []byte
	if count,err := d.input.Read(buffer); err != nil{
		if err == io.EOF{
			//Exit normally
			 C.mad_decoder_finish(d.decoder)
			
			return C.MAD_FLOW_STOP
			
		}else{	
			fmt.Println(err)
			return C.MAD_FLOW_BREAK
		}
	}else{
		//Read valid information
		C.mad_stream_buffer(stream, (*C.uchar)(unsafe.Pointer(&buffer[0])), C.ulong(count))
	
		return C.MAD_FLOW_CONTINUE	
	}

	return C.MAD_FLOW_CONTINUE
}

//export headerFunc
func headerFunc(data unsafe.Pointer, header *C.struct_mad_header)(C.enum_mad_flow){
	fmt.Println("header")
	
	d := (*Decoder)(data)
	
	
	d.Channels = 2 //Needs to determine this
	d.Bitrate = uint64(header.bitrate)
	d.SampleRate = uint(header.samplerate)
	
	return C.MAD_FLOW_CONTINUE
}

//export outputFunc
func outputFunc(data unsafe.Pointer, header *C.struct_mad_header,pcm *C.struct_mad_pcm)(C.enum_mad_flow){
	fmt.Println("output")
	
	d := (*Decoder)(data)
	        
	length := int(pcm.length)
	
	lSamples := C.GoBytes(unsafe.Pointer(&pcm.samples[0][0]), C.int(length*4)) //32 bits each
	rSamples := C.GoBytes(unsafe.Pointer(&pcm.samples[1][0]), C.int(length*4)) //32 bits each
	lBuf := bytes.NewReader(lSamples)
	rBuf := bytes.NewReader(rSamples)	
	
	for i:=0;i<length;i++{
		//Read in a signed 32 bit sample
		var l int32
		var r int32
		
		binary.Read(lBuf,binary.LittleEndian,&l)
		binary.Read(rBuf,binary.LittleEndian,&r)
		
		//Convert to 16 bit sample
		l16 := scale(l)
		r16 := scale(r)
		
		//Write to buffer
		binary.Write(d.b, binary.LittleEndian, l16)
		binary.Write(d.b, binary.LittleEndian, r16)
	}
	return C.MAD_FLOW_CONTINUE
} 

//export errorFunc
func errorFunc(data unsafe.Pointer, stream *C.struct_mad_stream, frame *C.struct_mad_frame)(C.enum_mad_flow){
	fmt.Println("Error Func")
	
	return C.MAD_FLOW_CONTINUE
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