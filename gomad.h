#include <stdio.h>
#include <stdint.h>
#include <limits.h>
#include <string.h>
#include <mad.h>

enum mad_flow input(void *data,struct mad_stream *stream);
struct mad_decoder* setupDecoder(struct mad_decoder *decoder, void *data);