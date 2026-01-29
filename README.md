# XWX_TTS 高效开源文本转语音

# 一个使用 ONNX Runtime 的文本到语音 (TTS) 项目，基于 melotts自训练模型实现。

# 目前已完成中文普通话英语g2p函数及混合训练，粤语英语g2p函数及混合训练

# 目前只开放tts推理端代码，golang实现，支持cpu及gpu推理，cpu推理的效率较高，更高的可能kokoro可以一战。理论上可以在任何cpu平台运行，性价比极高。

# melotts官方还支持韩语、日语、法语、德语、西班牙语。只要能找到对应语言的拼音分离方法及单人语音数据，可以支持任何语言训练及推理，推理速度也很快(1天准备数据、1天训练模型)

# 对于中文/方言、英语 只需给出5秒以上音色数据即可训练tts模型(训练部分基于melotts,暂不开放)

# 运行方法：
## windows平台：
### 当前目录解压放置 onnxruntime-win-x64-gpu-1.23.2/lib 目录及文件
#### ./build-win.sh
### 运行
#### ./tts-win.exe


## linux平台：
### 当前目录解压放置 onnxruntime-linux-x64-gpu-1.23.2/lib 目录及文件
#### ./build-linux.sh
### 运行 chmod +x tts-linux
#### ./tts-linux

## GPU运行模式需安装cuda驱动及cudnn


## 下载onnx模型 symbolid文件  bert模型文件 6个文件到当前目录
## https://huggingface.co/westice/xwx_tts

## 访问接口：POST http://127.0.0.1:8080/tts
```json
{
    "text": "我在迪士尼，每个项目最终归属于某个团队。您可以加入多个团队,清理工作：这些“火花”的大小直接衡量了它发育成胚胎的能力。，how are you?",
    "language": "zh_x",
    "device_type": "gpu",
    "speaker_id": 0,
    "speed": 1.1
}
```


## 测试运行源码

### language := "zh_x"
### deviceType := GPU //CPU
### newEngine, err := NewXWX_TTS(language, deviceType)
### newEngine.TtsTest("测试中文字符， are you ok?", "output.wav")
