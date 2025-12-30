# IFLOW.md - TTS-Golang 项目文档

## 项目概述

TTS-Golang 是一个使用 ONNX Runtime 的文本到语音 (TTS) 项目，基于 Go 语言实现。该项目支持多语言 TTS 功能，特别针对中文（普通话和粤语）+ 英语混合文本的语音合成。项目基于 melo-tts 和 sherpa-onnx-go 框架构建，使用 ONNX 格式的预训练模型进行语音合成。

### 核心特性

- **多语言支持**：支持中文（普通话、粤语）+ 英语混合文本的语音合成
- **ONNX Runtime**：使用 ONNX Runtime 进行高效的模型推理
- **GPU 加速**：支持 CUDA 加速以提升推理性能
- **BERT 集成**：集成了 BERT 特征提取器以提升语音质量
- **G2P 转换**：内置图形到音素 (Grapheme-to-Phoneme) 转换功能，支持中文和英文的音素转换

### 主要文件结构

- `main.go` - 项目主入口文件，演示 TTS 功能
- `melo-onnx-tts.go` - 核心 TTS 引擎实现
- `text_parse.go` - 文本解析和分段功能
- `mandaren_g2p.go` - 普通话 G2P 转换实现
- `cantonese_g2p.go` - 粤语 G2P 转换实现
- `bert_extractor.go` - BERT 特征提取器实现
- `english_g2p.go` - 英文 G2P 转换实现
- `onnxruntime-win-x64-gpu-1.23.2/` - Windows 平台的 ONNX Runtime 库

## 项目架构

### 核心组件

1. **XWX_TTS** - 主要的 TTS 引擎，负责模型加载、推理和音频生成
2. **BERTFeatureExtractor** - BERT 特征提取器，用于提取文本的语义特征
3. **G2P 模块** - 图形到音素转换模块，将文本转换为音素序列
4. **Text Parser** - 文本解析器，将混合文本按语言类型分段

### 语言支持

- **ZH_X** - 中文+英语混合
- **YUE_EN** - 粤语+英语混合

## 依赖项

- Go 1.21.5+
- ONNX Runtime Go 绑定: github.com/yalue/onnxruntime_go
- 多个中文拼音处理库
- BERT 词元化器和特征提取库

## 构建和运行

### 环境准备

1. 确保安装 Go 1.23.0 或更高版本
2. 确保系统具有可用的 ONNX Runtime 库
3. 准备必要的模型文件：
   - `yue_en_tts-model.onnx` - 粤语+英语模型
   - `zh_x_tts-model.onnx` - 中文+英语模型
   - `bert-base-multilingual-cased.onnx` - BERT 模型
   - `bert-base-multilingual-cased.json` - BERT 词元化器配置
   - `yue_en_symbolid.json` - 粤语+英语音素ID映射
   - `zh_x_symbolid.json` - 中文+英语音素ID映射

### 构建步骤

```bash
# 安装依赖
go mod tidy

# 运行项目
go run main.go
```

### 重要参数

- 采样率：24000 Hz
- 音频格式：WAV (16位 PCM)
- 支持速度调节：0.5~2.0 倍速

## 开发约定

### 代码结构

- 所有核心功能在 `main` 包中
- 每个语言的 G2P 功能分离到独立文件
- 使用 Go 语言标准库和经过验证的第三方库
- 模块化的架构设计

### 文本处理流程

1. 使用 `SplitText` 将输入文本按类型分段
2. 对中文文本进行 G2P 转换
3. 对英文文本使用 BERT 和 G2P 处理
4. 将音素序列映射到模型支持的 ID
5. 使用 ONNX Runtime 进行推理
6. 生成 PCM 音频数据并保存为 WAV 文件

## 模型配置

- TTS 模型路径根据语言自动选择
- BERT 模型支持多语言
- 自动检测 CPU 核心数以优化线程数配置

## 音频输出

- 默认输出 24kHz 采样率的 WAV 文件
- 支持速度控制（0.5-2.0倍速）
- 支持不同说话人 ID

## 项目限制

- 需要预训练的 ONNX 模型文件
- 粤语处理依赖外部 Python 服务 (pycantonese_service.py)
- CUDA 支持需要相应的 GPU 环境