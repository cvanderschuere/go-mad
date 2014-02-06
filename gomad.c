#include "gomad.h"

extern enum mad_flow inputFunc(void *data, struct mad_stream *stream);
extern enum mad_flow errorFunc(void *data, struct mad_stream *stream, struct mad_frame *frame);
extern enum mad_flow outputFunc(void *data, const struct mad_header *header, struct mad_pcm *pcm);
extern enum mad_flow headerFunc(void *data, const struct mad_header *header);

//Setup MAD decoder
struct mad_decoder* setupDecoder(struct mad_decoder *decoder, void *data){
	mad_decoder_init(decoder, data,&inputFunc,&headerFunc,0,&outputFunc,&errorFunc,0); // Ignored HEADER,FILTER,MESSAGE

	return decoder;
}