package mad

import (
	"testing"
	"os"
	"io"
	"fmt"
	"bufio"
	"github.com/cvanderschuere/alsa-go"
	"time"
)

func TestFromFile(t *testing.T){
	//Open test file
	file,_ := os.Open("test.mp3")	
	buffer := bufio.NewReader(file)
		
	//Create decoder
	decoder,_ := New(buffer)
	
	fmt.Println("Decoding")
		
	//Read all data
	readBuf := make([]byte,500000)
	c := 0
	
	//Open ALSA pipe
	controlChan := make(chan bool)
	streamChan := alsa.Init(controlChan)
	
	//Create stream
	dataChan := make(chan alsa.AudioData, 500)
	current_stream := alsa.AudioStream{Channels:2, Rate:int(44100),SampleFormat:alsa.INT16_TYPE, DataStream:dataChan}

	streamChan<-current_stream
	
	//Pause initially
	go func(){
		controlChan<-false
		time.Sleep(8*time.Second)
		controlChan<-true
	}()
	
	count,err := decoder.Read(readBuf)
	
	for err != io.EOF{
		//fmt.Println(readBuf)
		
		current_stream.DataStream<-readBuf
		
		c += count
		count,err = decoder.Read(readBuf)		
	}
		
	//time.Sleep(20*time.Second)
		
		
	fmt.Println(c)				
	fmt.Println("Finished")
	
}