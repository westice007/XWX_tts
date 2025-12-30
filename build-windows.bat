set GOOS=windows
set GOARCH=amd64
go build -o tts-win.exe main.go tts-http-service.go bert_extractor.go cantonese_g2p.go english_g2p.go mandaren_g2p.go mandaren_pinyin.go text_parse.go melo-onnx-tts.go

