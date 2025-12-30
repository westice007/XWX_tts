package main

import (
    "time"
	"os"
	"io"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"encoding/binary"
	ort "github.com/yalue/onnxruntime_go"	
)

// 定义语言枚举类型
type Language string

const (
	YUE_EN Language = "yue_en" // 粤语+英语
	ZH_X  Language = "zh_x"  // 中文+英语
)

// 定义设备类型枚举
type DeviceType string

const (
	CPU  DeviceType = "cpu"  // CPU 设备
	GPU  DeviceType = "gpu"  // GPU 设备
)


// 支持粤语英语的高性能TTS引擎,基于 melotts和sherpa-onnx-go
type XWX_TTS struct {
	//tok     *tokenizer.Tokenizer
	cpuCores int
	language Language
	deviceType DeviceType
	ttsModelPath string
	bertModelPath string
	bertTokenizerPath string

	symbolIDMap map[string]int
	session *ort.DynamicAdvancedSession
	bertExtractor *BERTFeatureExtractor
}



func init_onnx_environment() {
	if ort.IsInitialized() {
		return
	}
	// 设置ONNX Runtime环境
	// 判断当前系统
	if runtime.GOOS == "windows" {
		ort.SetSharedLibraryPath("./onnxruntime-win-x64-gpu-1.23.2/lib/onnxruntime.dll")
	} else if runtime.GOOS == "linux" {
		ort.SetSharedLibraryPath("./onnxruntime-linux-x64-gpu-1.23.2/lib/libonnxruntime.so")
	} else {
		panic("不支持的操作系统")
	}


	fmt.Println("TTS (Text-to-Speech) Project with ONNX Runtime")

	cpuCores := runtime.NumCPU()
    fmt.Printf("CPU核心数: %d\n", cpuCores)

	
    err := ort.InitializeEnvironment()
    if err != nil {
        panic(err)
    }
	//对象销毁时释放
    //defer ort.DestroyEnvironment()

	fmt.Println("ONNX Runtime initialized successfully")

}



func NewXWX_TTS(language Language, deviceType DeviceType) (*XWX_TTS, error) {
	start := time.Now()
	cpuCores := runtime.NumCPU()
    fmt.Printf("CPU核心数: %d\n", cpuCores)
	
	m := &XWX_TTS{
		cpuCores: cpuCores,
		language: language,
		deviceType: deviceType,
	}
	init_onnx_environment()
	m.prepareModelPath()

	m.load_symbolid()
	m.init_tts_onnx_model()
	m.init_bert_model()

	//g2p相关资源预加载
	CantoneseResourcePreload()
	// 预加载英文g2p字典
	EnglishResourcePreload()
	// 预加载普通话g2p字典
	MandarenResourcePreload()

	fmt.Printf("初始化tts引擎耗时: %v\n", time.Since(start))

	return m, nil
}

func (m *XWX_TTS) Destroy() {
	if m.session != nil {
		m.session.Destroy()
		m.session = nil
	}
	if m.bertExtractor != nil {
		m.bertExtractor.Destroy()
		m.bertExtractor = nil
	}
	//ort.DestroyEnvironment()
}

// func (m *XWX_TTS) init_onnx_environment() {
// 	// 设置ONNX Runtime环境
// 	// 判断当前系统
// 	if runtime.GOOS == "windows" {
// 		ort.SetSharedLibraryPath("./onnxruntime-win-x64-gpu-1.23.2/lib/onnxruntime.dll")
// 	} else if runtime.GOOS == "linux" {
// 		ort.SetSharedLibraryPath("./onnxruntime-linux-x64-gpu-1.23.2/lib/libonnxruntime.so")
// 	} else {
// 		panic("不支持的操作系统")
// 	}



// 	fmt.Println("TTS (Text-to-Speech) Project with ONNX Runtime")

// 	cpuCores := runtime.NumCPU()
//     fmt.Printf("CPU核心数: %d\n", cpuCores)

	
//     err := ort.InitializeEnvironment()
//     if err != nil {
//         panic(err)
//     }
// 	//对象销毁时释放
//     //defer ort.DestroyEnvironment()

// 	fmt.Println("ONNX Runtime initialized successfully")

// }

func (m *XWX_TTS) init_tts_onnx_model() {
	options, err := ort.NewSessionOptions()
	if err != nil {
		panic(err)
	}
	defer options.Destroy()

	// 设置图优化级别
	options.SetGraphOptimizationLevel(ort.GraphOptimizationLevelEnableAll)

	// 设置日志级别
	options.SetLogSeverityLevel(ort.LoggingLevelWarning)

	// 设置线程数
	// 根据CPU核心数设置线程数
    if m.cpuCores >= 8 {
        // 高性能CPU
        options.SetIntraOpNumThreads(8)   // 算子内并行：8线程
        options.SetInterOpNumThreads(4)   // 算子间并行：4线程
    } else if m.cpuCores >= 4 {
        // 中等性能CPU
        options.SetIntraOpNumThreads(4)   // 算子内并行：4线程
        options.SetInterOpNumThreads(2)   // 算子间并行：2线程
    } else {
        // 低性能CPU
        options.SetIntraOpNumThreads(2)   // 算子内并行：2线程
        options.SetInterOpNumThreads(1)   // 算子间并行：1线程
    }

	inputNames := []string{"x", "x_lengths", "tones", "sid", "bert", "ja_bert", "sdp_ratio", "noise_scale", "noise_scale_w", "length_scale"}
	
	outputNames := []string{"y"}

	// 根据设备类型配置执行提供程序
	if m.deviceType == GPU {
		// 1. 创建 CUDA 提供程序选项
		cudaOptions, err := ort.NewCUDAProviderOptions()
		if err != nil {
			log.Fatal("无法创建 CUDA 选项:", err)
		}
		defer cudaOptions.Destroy() // 使用完记得销毁，释放 C 内存

		// 2. 如果需要修改默认配置（例如设置显卡 ID 为 0）
		// 注意：Update 接收的是 map[string]string
		err = cudaOptions.Update(map[string]string{
			"device_id": "0",
		})
		if err != nil {
			log.Fatal("设置 CUDA 参数失败:", err)
		}

		// 3. 将选项添加进会话配置中
		// 此时传入的是 cudaOptions 指针，不再是简单的整数 0
		err = options.AppendExecutionProviderCUDA(cudaOptions)
		if err != nil {
			fmt.Println("CUDA 加速不可用，将回退到 CPU:", err)
		}else{
			fmt.Println("CUDA 加速可用，将使用GPU推理")
		}
	} else {
		// 使用 CPU 设备
		fmt.Println("使用 CPU 设备进行推理")
	}

    
    dynamicSession, err := ort.NewDynamicAdvancedSession(m.ttsModelPath, inputNames, outputNames, options)
	if err != nil {
		panic(err)
	}
	m.session = dynamicSession
}

func (m *XWX_TTS) init_bert_model() {

	bertExtractor, err := NewBERTFeatureExtractor(m.bertModelPath, m.bertTokenizerPath)
	if err != nil {
		log.Fatalf("创建BERT特征提取器失败: %v", err)
	}else{
		//log.Infof("创建BERT特征提取器成功: %v", bertExtractor)
	}
	//defer bertExtractor.Destroy()
	//销毁权归于 XWX_TTS
	m.bertExtractor = bertExtractor
}

func (m *XWX_TTS)prepareModelPath() {
	// 根据语言设置模型路径
	if m.language == YUE_EN {
		m.ttsModelPath = 	  "./yue_en_tts-model.onnx"
		m.bertModelPath = 	  "./bert-base-multilingual-cased.onnx"
		m.bertTokenizerPath = "./bert-base-multilingual-cased.json"
	}
	if m.language == ZH_X {
		m.ttsModelPath = 	  "./zh_x_tts-model.onnx"
		m.bertModelPath = 	  "./bert-base-multilingual-cased.onnx"
		m.bertTokenizerPath = "./bert-base-multilingual-cased.json"
	}

	// 检查文件是否存在
	if _, err := os.Stat(m.ttsModelPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("模型文件不存在: %s", m.ttsModelPath))
	}
	if _, err := os.Stat(m.bertModelPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("模型文件不存在: %s", m.bertModelPath))
	}
	if _, err := os.Stat(m.bertTokenizerPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("模型文件不存在: %s", m.bertTokenizerPath))
	}
}

func (m *XWX_TTS)load_symbolid() {
	// 加载音素符号ID映射
	symbolidFileName := ""
	if m.language == YUE_EN {
		symbolidFileName = "yue_en_symbolid.json"
	} else if m.language == ZH_X {
		symbolidFileName = "zh_x_symbolid.json"
	}

	symbolIDMap := make(map[string]int)
	jsonFile, err := os.Open(symbolidFileName)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	var jsonData map[string][]string
	json.Unmarshal(byteValue, &jsonData)
	symbols := jsonData["symbols"]
	for i, symbol := range symbols {
		symbolIDMap[symbol] = i
	}
	m.symbolIDMap = symbolIDMap
}

func (m *XWX_TTS)mapping_phones(phones []string) []int64 {
	// 映射到模型支持的 phones 列表，并添加隔位0
	mappedPhones := []int64{}
	for _, phone := range phones {
		if id, ok := m.symbolIDMap[phone]; ok {
			mappedPhones = append(mappedPhones, 0)
			mappedPhones = append(mappedPhones, int64(id))
		}
	}
	mappedPhones = append(mappedPhones, 0) //尾部再加一个
	return mappedPhones
}

func (m *XWX_TTS)mapping_tones(tones []int) []int64 {
	// 映射到模型支持的 tones 列表，并添加隔位0，
	mappedTones := []int64{}
	for _, tone := range tones {
		mappedTones = append(mappedTones, 0) //有个偏移20，因为还有melotts的默认其它语言
		mappedTones = append(mappedTones, int64(tone + 20))		
	}
	mappedTones = append(mappedTones, 0) //尾部再加一个
	return mappedTones
}

func (m *XWX_TTS)mapping_word2ph(word2ph []int) []int64 {
	// 映射到模型支持的 word2ph 列表，每个位置x2。  参考melotts的逻辑
	mappedWord2ph := []int64{}
	for _, ph := range word2ph {
		mappedWord2ph = append(mappedWord2ph, int64(ph*2))
	}
	mappedWord2ph[0] += 1 
	return mappedWord2ph
}

// 推理得到pcm音频数据 speakerid一般为0， speed为 0.5~2.0
// 返回数据为float32类型的pcm音频数据, 采样率24000
func (m *XWX_TTS)Tts_pcm(text string, speakerid int, speed float32) []float32 {
	startTime000 := time.Now()
	mix_phones := []string{"_"}
	mix_tones := []int{0}
	mix_word2ph := []int{1}
	filteredText := ""

	if m.language == YUE_EN {
		mix_phones ,mix_tones, mix_word2ph, filteredText = CantoneseMix_g2p(text, m.bertExtractor)
	} else if m.language == ZH_X {
		mix_phones ,mix_tones, mix_word2ph, filteredText = MandarenMix_g2p(text, m.bertExtractor)
	}
	mappedPhones := m.mapping_phones(mix_phones)
	mappedTones := m.mapping_tones(mix_tones)
	mappedWord2ph := m.mapping_word2ph(mix_word2ph)
		
	mappedPhonesLen := int64(len(mappedPhones))
	mappedTonesLen := int64(len(mappedTones))
	//mappedWord2phLen := int64(len(mappedWord2ph))

	
	xShape := ort.NewShape(1, mappedPhonesLen)
	xTensor, _ := ort.NewTensor(xShape, mappedPhones)  
	defer xTensor.Destroy()

	xLengthsData := []int64{mappedPhonesLen}
	xLengthsShape := ort.NewShape(1)
	xLengthsTensor, _ := ort.NewTensor(xLengthsShape, xLengthsData) // x_lengths 数据
	defer xLengthsTensor.Destroy()

	tonesShape := ort.NewShape(1, mappedTonesLen)
	tonesTensor, _ := ort.NewTensor(tonesShape, mappedTones) // tones 数据	
	defer tonesTensor.Destroy()
	
	sidData := []int64{int64(speakerid)} //speakerid ,外部指定
	sidShape := ort.NewShape(1)
	sidTensor, _ := ort.NewTensor(sidShape, sidData) // sid 数据
	defer sidTensor.Destroy()

	//根据melotts逻辑，bertshape全0向量
	bertShape := ort.NewShape(1, 1024, mappedPhonesLen)
    bertTensor, _ := ort.NewEmptyTensor[float32](bertShape)
    defer bertTensor.Destroy()

	jaBertTensor, err := m.bertExtractor.ExtractFeaturesForTTS(filteredText, mappedWord2ph)
	//fmt.Println("jaBertTensor形状:", jaBertTensor.GetShape())
	if err != nil {
		fmt.Printf("提取JA-BERT特征失败: %v", err)
	}
	defer jaBertTensor.Destroy()

	sdpRatioData := []float32{0.2}
	sdpRatioShape := ort.NewShape(1)
	sdpRatioTensor, _ := ort.NewTensor(sdpRatioShape, sdpRatioData) // sdp_ratio 数据
	defer sdpRatioTensor.Destroy()

	noiseScaleData := []float32{0.6}
	noiseScaleShape := ort.NewShape(1)
	noiseScaleTensor, _ := ort.NewTensor(noiseScaleShape, noiseScaleData) // noise_scale 数据
	defer noiseScaleTensor.Destroy()

	noiseScaleWTensor, _ := ort.NewEmptyTensor[float32](noiseScaleShape)
    defer noiseScaleWTensor.Destroy()


	lengthScaleData := []float32{float32(1.0 / speed)}
	lengthScaleShape := ort.NewShape(1)
	lengthScaleTensor, _ := ort.NewTensor(lengthScaleShape, lengthScaleData) // length_scale 数据
	defer lengthScaleTensor.Destroy()

	inputs := []ort.Value{xTensor, xLengthsTensor, tonesTensor, sidTensor, bertTensor, jaBertTensor, sdpRatioTensor, noiseScaleTensor, noiseScaleWTensor, lengthScaleTensor}	

	outputs := []ort.Value{nil} // 会自动分配输出张量

	// 计算推理耗时
	startTime := time.Now()
	err = m.session.Run(inputs, outputs)
	duration := time.Since(startTime)
	
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("\n=== 推理性能 ===")
	fmt.Printf("推理耗时: %v\n", duration)
	fmt.Printf("推理速度: %.2f seconds\n", duration.Seconds())

	// 清理自动分配的输出张量
	defer outputs[0].Destroy()

    // 打印输出张量的形状信息
    //fmt.Println("\n=== 输出张量形状信息 ===")
    //outputShape := outputs[0].GetShape()
    //fmt.Printf("输出张量维度: %d\n", len(outputShape))
    //fmt.Printf("输出形状: %v\n", outputShape)
    //fmt.Printf("输出扁平化大小: %d\n", outputShape.FlattenedSize())

	duration = time.Since(startTime000)
	fmt.Printf("TTS 处理耗时: %v\n", duration)

	floatTensor, ok := outputs[0].(*ort.Tensor[float32])
    if !ok {
        fmt.Errorf("无法转换为float32张量")
    }
    
    data := floatTensor.GetData()
    //fmt.Printf("音频数据长度: %d 个采样点\n", len(data))
	//fmt.Printf("音频数据示例: %v\n", data[:20])

	return data
}

func (m *XWX_TTS)TtsTest(text string, wavOutPath string) {

	speaker_id := 0
	speed := float32(1.2)
	pcmData := m.Tts_pcm(text, speaker_id, speed)

	sampleRate := 24000
	err := saveAsWAV(pcmData, wavOutPath , sampleRate)
	
	if err != nil {
		fmt.Printf("写入WAV文件失败: %v", err)
	}
}


// saveAsWAV 将张量数据保存为WAV文件
func saveAsWAV(pcmData []float32, filename string, sampleRate int) error {
    
    fmt.Printf("音频时长: %.2f 秒\n", float64(len(pcmData))/float64(sampleRate))
    
    // 创建WAV文件
    file, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("创建文件失败: %w", err)
    }
    defer file.Close()
    
    // 写入WAV文件头
    err = writeWAVHeader(file, len(pcmData), sampleRate)
    if err != nil {
        return fmt.Errorf("写入WAV头失败: %w", err)
    }
    
    // 将float32数据转换为16位PCM并写入
    for _, sample := range pcmData {
        // 确保数据范围在 [-1, 1] 之间
        if sample > 1.0 {
            sample = 1.0
        } else if sample < -1.0 {
            sample = -1.0
        }
        
        // 转换为16位PCM
        pcmValue := int16(sample * 32767)
        
        // 写入小端格式
        err = binary.Write(file, binary.LittleEndian, pcmValue)
        if err != nil {
            return fmt.Errorf("写入音频数据失败: %w", err)
        }
    }
    
    fmt.Printf("✓ WAV文件已保存: %s\n", filename)
    return nil
}

// writeWAVHeader 写入WAV文件头
func writeWAVHeader(file *os.File, numSamples int, sampleRate int) error {
    // WAV文件参数
    const (
        bitsPerSample = 16
        numChannels   = 1
    )
    
    dataSize := uint32(numSamples * bitsPerSample / 8 * numChannels)
    fileSize := dataSize + 36 // 文件大小 = 数据大小 + 头大小(44) - "RIFF"标识(4) - 文件大小字段(4)
    
    // 写入RIFF头
    file.WriteString("RIFF")
    binary.Write(file, binary.LittleEndian, uint32(fileSize))
    file.WriteString("WAVE")
    
    // 写入fmt chunk
    file.WriteString("fmt ")
    binary.Write(file, binary.LittleEndian, uint32(16)) // chunk size
    binary.Write(file, binary.LittleEndian, uint16(1))  // PCM格式
    binary.Write(file, binary.LittleEndian, uint16(numChannels))
    binary.Write(file, binary.LittleEndian, uint32(sampleRate))
    binary.Write(file, binary.LittleEndian, uint32(sampleRate*bitsPerSample/8*numChannels)) // 字节率
    binary.Write(file, binary.LittleEndian, uint16(bitsPerSample/8*numChannels)) // 块对齐
    binary.Write(file, binary.LittleEndian, uint16(bitsPerSample))
    
    // 写入data chunk头
    file.WriteString("data")
    binary.Write(file, binary.LittleEndian, dataSize)
    
    return nil
}
