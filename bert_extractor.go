package main

import (
	"fmt"
	//"log"
	//"time"
	"runtime"
	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

// BERTFeatureExtractor BERT特征提取器
type BERTFeatureExtractor struct {
	tok     *tokenizer.Tokenizer
	session *ort.DynamicAdvancedSession
}

// NewBERTFeatureExtractor 创建新的BERT特征提取器
func NewBERTFeatureExtractor(modelPath string, tokenizerPath string) (*BERTFeatureExtractor, error) {
	// 初始化ONNX Runtime环境
	
	// if !ort.IsInitialized() {
	// 	if err := ort.InitializeEnvironment(); err != nil {
	// 		return nil, fmt.Errorf("初始化ONNX Runtime环境失败: %w", err)
	// 	}
	// }

	tok, err := pretrained.FromFile(tokenizerPath)
	if err != nil {
		panic(err)
	}



	// 定义BERT模型的输入输出名称
	inputNames := []string{"input_ids", "token_type_ids", "attention_mask"}
	outputNames := []string{"last_hidden_state"}

	cpuCores := runtime.NumCPU()

	// 创建会话
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
    if cpuCores >= 8 {
        // 高性能CPU
        options.SetIntraOpNumThreads(8)   // 算子内并行：8线程
        options.SetInterOpNumThreads(4)   // 算子间并行：4线程
    } else if cpuCores >= 4 {
        // 中等性能CPU
        options.SetIntraOpNumThreads(4)   // 算子内并行：4线程
        options.SetInterOpNumThreads(2)   // 算子间并行：2线程
    } else {
        // 低性能CPU
        options.SetIntraOpNumThreads(2)   // 算子内并行：2线程
        options.SetInterOpNumThreads(1)   // 算子间并行：1线程
    }
	session, err := ort.NewDynamicAdvancedSession(modelPath, inputNames, outputNames, options)
	if err != nil {
		return nil, fmt.Errorf("创建BERT模型会话失败: %w", err)
	}

	return &BERTFeatureExtractor{
		tok:     tok,
		session: session,
	}, nil
}

// Destroy 销毁BERT特征提取器
func (b *BERTFeatureExtractor) Destroy() {
	fmt.Println("销毁BERT特征提取器")
	if b.session != nil {
		b.session.Destroy()
		b.session = nil
	}
}

func (b *BERTFeatureExtractor) Tokenize(text string) []string {
	tokens, _ := b.tok.Tokenize(text)
	return tokens
}

// ExtractFeatures 提取文本的BERT特征
func (b *BERTFeatureExtractor) ExtractFeatures(text string) (ort.Value, error) {

	//input := tokenizer.NewInput(text)

	// 2. 编码
	enc, err := b.tok.EncodeSingle(text, true)
	if err != nil {
		panic(err)
	}

	// 3. 直接取 3 个 []int64
	_inputIDs      := enc.Ids
	_attentionMask := enc.AttentionMask
	_tokenTypeIDs  := enc.TypeIds

	// 4. 打印对齐验证
	//fmt.Println("input_ids     :", _inputIDs)
	//fmt.Println("attention_mask:", _attentionMask)
	//fmt.Println("token_type_ids:", _tokenTypeIDs)

	inputIDs := make([]int64, len(_inputIDs))
	for i, v := range _inputIDs { // 单次遍历，CPU 会内联
		inputIDs[i] = int64(v)
	}

	attentionMask := make([]int64, len(_attentionMask))
	for i, v := range _attentionMask { // 单次遍历，CPU 会内联
		attentionMask[i] = int64(v)
	}

	tokenTypeIDs := make([]int64, len(_tokenTypeIDs))
	for i, v := range _tokenTypeIDs { // 单次遍历，CPU 会内联
		tokenTypeIDs[i] = int64(v)
	}

	// 创建输入张量
	batchSize := int64(1)
	seqLength := int64(len(inputIDs))
	
	inputIDsTensor, err := ort.NewTensor(ort.NewShape(batchSize, seqLength), inputIDs)
	if err != nil {
		return nil, fmt.Errorf("创建input_ids张量失败: %w", err)
	}
	defer inputIDsTensor.Destroy()

	seqLength = int64(len(attentionMask))
	attentionMaskTensor, err := ort.NewTensor(ort.NewShape(batchSize, seqLength), attentionMask)
	if err != nil {
		return nil, fmt.Errorf("创建attention_mask张量失败: %w", err)
	}
	defer attentionMaskTensor.Destroy()

	seqLength = int64(len(tokenTypeIDs))
	tokenTypeIDsTensor, err := ort.NewTensor(ort.NewShape(batchSize, seqLength), tokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("创建token_type_ids张量失败: %w", err)
	}
	defer tokenTypeIDsTensor.Destroy()

	// 准备输入
	inputs := []ort.Value{inputIDsTensor, tokenTypeIDsTensor, attentionMaskTensor}
	outputs := []ort.Value{nil} // 会自动分配输出张量

	// 运行模型
	err = b.session.Run(inputs, outputs)
	if err != nil {
		return nil, fmt.Errorf("运行BERT模型失败: %w", err)
	}



	// 返回特征（注意：调用者需要负责销毁返回的张量）
	return outputs[0], nil
}

// ExtractFeaturesForTTS 为TTS任务提取BERT特征
func (b *BERTFeatureExtractor) ExtractFeaturesForTTS(text string, word2ph []int64) (*ort.Tensor[float32], error) {
	// 提取特征
	tensor, err := b.ExtractFeatures(text)
	if err != nil {
		return nil, err
	}
	defer tensor.Destroy()

	// 检查输出类型
	if tensor.GetONNXType() != ort.ONNXTypeTensor {
		return nil, fmt.Errorf("BERT输出不是张量类型")
	}

	featureTensor, ok := tensor.(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("BERT特征数据类型不是float32")
	}

	//fmt.Println("BERT模型返回特征张量形状:", featureTensor.GetShape())
	//fmt.Println("BERT特征张量:", featureTensor)

	// // 获取特征数据
	flatData := featureTensor.GetData()
	//dataLen := len(flatData)

	//rowfirst := flatData[0:10]
	//fmt.Println("原始数据rowfirst:", rowfirst)

	//row2 := flatData[768:768+10]
	//fmt.Println("原始数据row2:", row2)

	//fmt.Println("featureData:", len(featureData))
	// reshape 成 [10][768]
	hidden := int64(768)
	//seqLen := int64(dataLen) / hidden
	
	//fmt.Println("dataLen:", dataLen)
	//fmt.Println("hidden:", hidden)




	// repeat
	sumWord2ph := int64(0)
	for _, r := range word2ph {
		sumWord2ph += r
	}
	//fmt.Println("sumWord2ph:", sumWord2ph)

	phoneLevelFlatData := make([]float32, sumWord2ph*hidden)
	phoneLevelFlatDataIndex := int64(0)
	for i, repeat := range word2ph {
		vec := make([]float32, hidden)
		copy(vec, flatData[i*int(hidden):(i+1)*int(hidden)])

		//fmt.Printf("flatData test: %v\n", vec[0:0+20])
		
		for j := int64(0); j < repeat; j++ {
			rows := phoneLevelFlatDataIndex + j
			copy(phoneLevelFlatData[rows*hidden:(rows+1)*hidden], vec)
		}
		phoneLevelFlatDataIndex += repeat
	}

	//fmt.Printf("phoneLevelFlatData test: %v\n", phoneLevelFlatData[8*768:8*768+20])

	newShape := ort.NewShape(int64(sumWord2ph), hidden)
		
	// 3. 如果需要将其作为另一个模型的输入，可以用新 Shape 重新封装
	phoneLevelTensor, err := ort.NewTensor(newShape, phoneLevelFlatData)
	if err != nil {
		return nil, fmt.Errorf("创建新张量失败: %w", err)
	}
	
	
	defer phoneLevelTensor.Destroy()
	//startTime := time.Now()

	retTensor, err2 := transpose2DAndAddBatchDim(phoneLevelTensor)

	//extractDuration := time.Since(startTime)
	//fmt.Printf("BERT2D转置耗时: %v\n", extractDuration)
	
	//外部管理张量生命周期
	return retTensor, err2



}
// transpose2D 将二维 float32 矩阵转置，输入输出均为 ONNX Tensor
func transpose2DAndAddBatchDim(srcTensor *ort.Tensor[float32]) (*ort.Tensor[float32], error) {
	shape := srcTensor.GetShape()
	if len(shape) != 2 {
		return nil, fmt.Errorf("transpose2D: 输入张量必须是 2 维，当前维度 %v", shape)
	}
	rows, cols := int(shape[0]), int(shape[1])
	src := srcTensor.GetData()

	// 创建转置后的数据
	dst := make([]float32, rows*cols)
	for j := 0; j < cols; j++ {
		for i := 0; i < rows; i++ {
			//dst = append(dst, src[i*cols+j])
			dst[j*rows+i] = src[i*cols+j]
		}
	}

	// 构造输出张量
	dstTensor, err := ort.NewTensor(ort.NewShape(1,int64(cols), int64(rows)), dst)
	if err != nil {
		return nil, fmt.Errorf("transpose2D: 创建输出张量失败: %w", err)
	}
	return dstTensor, nil
}


// GetFeatureShape 获取BERT特征的形状
func (b *BERTFeatureExtractor) GetFeatureShape(text string) (ort.Shape, error) {
	// 提取特征
	featureTensor, err := b.ExtractFeatures(text)
	if err != nil {
		return nil, err
	}
	defer featureTensor.Destroy()

	return featureTensor.GetShape(), nil
}
