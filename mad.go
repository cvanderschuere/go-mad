// Package to interface with MPEG Audio Decoder
package mad

/*
	#cgo pkg-config: mad
	#include <stdlib.h>
	#include <mad.h>

	static inline struct mad_decoder* setupDecoder(void *data, void *i, void *o,void *e){
		struct mad_decoder *decoder = malloc(sizeof(struct mad_decoder));
		mad_decoder_init(decoder, data,i,0,0,o,e,0);
	
		return decoder;
	}	
*/
import "C"

import(
	"io"
	"unsafe"
	"fmt"
)

type Decoder struct {
	input *io.Reader
	
	// C internals
	decoder *C.struct_mad_decoder
}

// Create new decoder with reader
func New(i *io.Reader)(*Decoder,error){
	d := new(Decoder)
	

	d.input = i
	d.decoder = C.setupDecoder(unsafe.Pointer(d.input),unsafe.Pointer(&inputFuncVar),unsafe.Pointer(&outputFuncVar),unsafe.Pointer(&errorFuncVar))// Ignored HEADER,FILTER,MESSAGE

	return d,nil
}

func (d *Decoder) Read(p []byte) (n int, err error){
	
	
	return 0,nil
}


//
// Callbacks
//

var inputFuncVar = inputFunc
var outputFuncVar = outputFunc
var errorFuncVar = errorFunc


func inputFunc(data unsafe.Pointer, stream *C.struct_mad_stream)(C.enum_mad_flow){
	
	return C.MAD_FLOW_CONTINUE
}

func outputFunc(data unsafe.Pointer, header *C.struct_mad_header,pcm *C.struct_mad_pcm)(C.enum_mad_flow){
	return C.MAD_FLOW_CONTINUE
	
} 

func errorFunc(data unsafe.Pointer, stream *C.struct_mad_stream, frame *C.struct_mad_frame)(C.enum_mad_flow){
	return C.MAD_FLOW_CONTINUE
	
}