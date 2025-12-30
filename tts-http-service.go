package main

import (
	"bytes"
	// "encoding/json"
	"fmt"
	// "io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TTS引擎缓存，key为 language-device_type 组合
var ttsEngineCache = make(map[string]*XWX_TTS)
var ttsEngineMutex sync.RWMutex // 用于保护缓存map的并发访问

// API请求结构体
type TTSRequest struct {
	Text       string     `json:"text" binding:"required"`           // 要转换的文本
	Language   Language   `json:"language" binding:"required"`       // 语言类型
	SpeakerID  *int       `json:"speaker_id,omitempty"`              // 发音人ID，默认为0
	Speed      *float32   `json:"speed,omitempty"`                   // 速度，默认为1.0
	DeviceType *DeviceType `json:"device_type,omitempty"`            // 设备类型，默认为GPU
}

// API响应结构体
type TTSResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	AudioLen int    `json:"audio_length,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// 获取或创建TTS引擎实例
func getOrCreateTTSEngine(language Language, deviceType DeviceType) (*XWX_TTS, error) {
	key := fmt.Sprintf("%s-%s", language, deviceType)
	
	// 尝试从缓存中获取
	ttsEngineMutex.RLock()
	if engine, exists := ttsEngineCache[key]; exists {
		ttsEngineMutex.RUnlock()
		return engine, nil
	}
	ttsEngineMutex.RUnlock()
	
	// 缓存中不存在，创建新的引擎实例
	ttsEngineMutex.Lock()
	defer ttsEngineMutex.Unlock()
	
	// 双重检查，防止并发创建
	if engine, exists := ttsEngineCache[key]; exists {
		return engine, nil
	}
	
	fmt.Printf("创建新的TTS引擎实例，语言: %s, 设备: %s\n", language, deviceType)
	newEngine, err := NewXWX_TTS(language, deviceType)
	if err != nil {
		return nil, fmt.Errorf("创建TTS引擎失败: %v", err)
	}
	
	ttsEngineCache[key] = newEngine
	fmt.Printf("TTS引擎实例创建成功并已缓存，当前缓存大小: %d\n", len(ttsEngineCache))
	
	return newEngine, nil
}

// TTS API处理器
func ttsHandler(c *gin.Context) {
	startTime := time.Now()

	// 解析请求体
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 设置默认值
	speakerID := 0
	if req.SpeakerID != nil {
		speakerID = *req.SpeakerID
	}

	speed := float32(1.0)
	if req.Speed != nil {
		speed = *req.Speed
	}

	deviceType := CPU
	if req.DeviceType != nil {
		deviceType = *req.DeviceType
	}

	// 获取或创建TTS引擎实例
	ttsEngine, err := getOrCreateTTSEngine(req.Language, deviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 执行TTS转换
	audioData := ttsEngine.Tts_pcm(req.Text, speakerID, speed)

	// 计算音频时长
	sampleRate := 24000
	audioDuration := float64(len(audioData)) / float64(sampleRate)

	// 将PCM数据转换为WAV格式
	wavBuffer := &bytes.Buffer{}
	err = writeWAVToBuffer(audioData, wavBuffer, sampleRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "生成WAV音频失败: " + err.Error(),
		})
		return
	}

	// 计算处理时间
	duration := time.Since(startTime)

	// 设置响应头
	c.Header("Content-Type", "audio/wav")
	c.Header("Content-Disposition", "attachment; filename=\"tts_output.wav\"")
	c.Header("Content-Length", strconv.Itoa(wavBuffer.Len()))
	
	// 返回WAV音频流
	c.Data(http.StatusOK, "audio/wav", wavBuffer.Bytes())

	// 记录日志
	fmt.Printf("TTS API调用成功: 文本长度=%d, 语言=%s, 发音人=%d, 速度=%.2f, 设备=%s, 音频大小=%d bytes, 音频时长=%.2f秒, 耗时=%v\n", 
		len(req.Text), req.Language, speakerID, speed, deviceType, wavBuffer.Len(), audioDuration, duration)
}

// 将PCM数据写入WAV格式的buffer
func writeWAVToBuffer(pcmData []float32, buffer *bytes.Buffer, sampleRate int) error {
	// WAV文件参数
	const (
		bitsPerSample = 16
		numChannels   = 1
	)

	dataSize := uint32(len(pcmData) * bitsPerSample / 8 * numChannels)
	fileSize := dataSize + 36 // 文件大小 = 数据大小 + 头大小(44) - "RIFF"标识(4) - 文件大小字段(4)

	// 写入RIFF头
	buffer.WriteString("RIFF")
	writeUint32(buffer, fileSize)
	buffer.WriteString("WAVE")

	// 写入fmt chunk
	buffer.WriteString("fmt ")
	writeUint32(buffer, 16) // chunk size
	writeUint16(buffer, 1)  // PCM格式
	writeUint16(buffer, uint16(numChannels))
	writeUint32(buffer, uint32(sampleRate))
	writeUint32(buffer, uint32(sampleRate*bitsPerSample/8*numChannels)) // 字节率
	writeUint16(buffer, uint16(bitsPerSample/8*numChannels))            // 块对齐
	writeUint16(buffer, uint16(bitsPerSample))

	// 写入data chunk头
	buffer.WriteString("data")
	writeUint32(buffer, dataSize)

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
		err := writeUint16(buffer, uint16(pcmValue))
		if err != nil {
			return err
		}
	}

	return nil
}

// 辅助函数：写入32位整数（小端格式）
func writeUint32(buffer *bytes.Buffer, value uint32) error {
	return writeUint(buffer, value, 4)
}

// 辅助函数：写入16位整数（小端格式）
func writeUint16(buffer *bytes.Buffer, value uint16) error {
	return writeUint(buffer, uint32(value), 2)
}

// 辅助函数：写入指定字节数的整数（小端格式）
func writeUint(buffer *bytes.Buffer, value uint32, size int) error {
	for i := 0; i < size; i++ {
		buffer.WriteByte(byte(value))
		value >>= 8
	}
	return nil
}

// 健康检查API
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "TTS服务运行正常",
		"timestamp": time.Now().Unix(),
		"engine_cache_size": len(ttsEngineCache),
	})
}

// 获取支持的语言列表
func languagesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"languages": []string{
			string(ZH_X),   // 中文+英语
			string(YUE_EN), // 粤语+英语
		},
		"message": "支持的语言列表",
	})
}


// 启动HTTP服务函数
func StartTTSHTTPService(port string) {
	// 设置Gin为生产模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin路由器
	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
	})

	// API路由
	r.POST("/tts", ttsHandler)                    // TTS转换API
	r.GET("/health", healthHandler)               // 健康检查
	r.GET("/languages", languagesHandler)         // 支持的语言列表
	
	fmt.Printf("TTS HTTP服务启动中，监听端口: %s\n", port)
	
	if err := r.Run(":" + port); err != nil {
		fmt.Printf("启动HTTP服务失败: %v\n", err)
	}
}