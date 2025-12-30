package main

import (
	"fmt"
	//"log"
	//"encoding/binary"
    // "math"
    //"os"
	//"time"
	//"runtime"

	//ort "github.com/yalue/onnxruntime_go"	
)



func main() {
	fmt.Println("Starting TTS HTTP service...")
	// 启动TTS HTTP服务，指定端口为8080
	StartTTSHTTPService("8080")
}
