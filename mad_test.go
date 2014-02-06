package mad

import (
	"testing"
	"os"
	//"io"
	"fmt"
	"time"
	"bufio"
)

func TestFromFile(t *testing.T){
	//Open test file
	file,_ := os.Open("test.mp3")

	buffer := bufio.NewReader(file)

	//Create decoder
	decoder,_ := New(buffer)
	
	//Read information
	fmt.Println(*decoder)
	
	go decoder.Decode()
	
	/*
	//Read all data
	var buffer []byte
	c := 0
	
	count,err := decoder.Read(buffer)
	
	for err != io.EOF{
		c += count
		count,err = decoder.Read(buffer)		
	}
	
	fmt.Println(c)
	*/		
	fmt.Println("Decoding")
	
	time.Sleep(5*time.Second)
	fmt.Println(decoder.SampleRate)
		
	fmt.Println("Finished")
	
}