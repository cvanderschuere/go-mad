// Package to interface with MPEG Audio Decoder
package mad


//#cgo pkg-config: mad
//#include <stdlib.h>
//#include "gomad.h"
import "C"

import(
	"io"
	"unsafe"
	"fmt"
	"encoding/binary"
	"math"
	"bytes"
)

type Decoder struct {
	input io.Reader
	output chan int16
	
	buf []byte
	done bool
	
	//Audio Information
	SampleRate uint
	Channels uint8
	Bitrate uint64
	
	// C internals
	decoder *C.struct_mad_decoder
}

const(
	BUF_SIZE = 1048576
	) 

// Create new decoder with reader
func New(i io.Reader)(*Decoder,error){
	d := new(Decoder)

	d.input = i
	
	size := int(unsafe.Sizeof(C.struct_mad_decoder{}))
	d.decoder = (*C.struct_mad_decoder)(C.malloc(C.size_t(size)))
	d.output = make(chan int16, 50000) //random value for buffer
	d.buf = make([]byte,BUF_SIZE)
	d.done = false
		
	C.setupDecoder(d.decoder,unsafe.Pointer(d))
	
	//Start decoding
	go func(){
		result := C.mad_decoder_run(d.decoder, C.MAD_DECODER_MODE_SYNC);
		if result != 0{
			fmt.Println("Decoder Error")
		}
		C.mad_decoder_finish(d.decoder)	
		close(d.output)
	}()

	return d,nil
}

// Read decoded bytes from input (buffered)
func (d *Decoder) Read(p []byte) (n int, err error){
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
	//uintptr(unsafe.Pointer(p))
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
		
		d.output<-l16
		d.output<-r16
	}

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


func scale(i int32)(int16){
	if i >= math.MaxInt16{
		return math.MaxInt16
	}else if i <= math.MinInt16{
		return math.MinInt16
	}else{
		return int16(i >> (uint(C.MAD_F_FRACBITS) - 15))
	}
}