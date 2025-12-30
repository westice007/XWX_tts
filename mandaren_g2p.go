package main

import (
	// "bytes"
    // "io"
    // "time"
	// "encoding/json"
	// "fmt"
	// "io/ioutil"
	// "net/http"
	//"net/url"
	"strconv"
	// "strings"
	// "regexp"
	"strings"
	// "unicode"
	//"github.com/mozillazg/go-pinyin"
	//"github.com/go-ego/gpy"
	//"github.com/go-ego/gpy/phrase"
	

	"github.com/ZingYao/chinese_number"

)

func MandarenResourcePreload() {
	Mandaren_pinyinresourcePreload()

}

func MandarenMix_g2p(text string, bertExtractor *BERTFeatureExtractor) ([]string, []int, []int, string) {
	
	mix_phones := []string{"_"}
	mix_tones := []int{0}
	mix_word2ph := []int{1}

	// 处理中英文特殊符号混合语句

	// 将中文、 英文、数字、符号 分离成单独顺序片段
	segments := SplitText(text)

	//chineseSentences := map[string]string{}

	filteredText := ""

	// 将中文单独拿出来，统一调用粤语拼音分词接口
	for _, segment := range segments {
		//indexStr := strconv.Itoa(index)
		if segment.Type == TypeChinese{
			sentence := segment.Content
			filteredText += sentence
			phones, tones, word2ph := Mandaren_g2p(sentence)
			mix_phones = append(mix_phones, phones...)
			mix_tones = append(mix_tones, tones...)
			mix_word2ph = append(mix_word2ph, word2ph...)
		}
		if segment.Type == TypeNumber{
			// 数字简单转换为中文模式，具体取决于业务模式，如钱币 日期 时间，可在前端进行处理
			num, _ := strconv.ParseInt(segment.Content, 10, 64)
			zhstr := chinese_number.Number2Simplified(num)
			filteredText += zhstr
			phones, tones, word2ph := Mandaren_g2p(zhstr)
			mix_phones = append(mix_phones, phones...)
			mix_tones = append(mix_tones, tones...)
			mix_word2ph = append(mix_word2ph, word2ph...)
		} 
		if segment.Type == TypeEnglish{
			sentence := segment.Content
			filteredText += sentence
			en_phones ,en_tones, en_word2ph := English_g2p(sentence, bertExtractor)
			mix_phones = append(mix_phones, en_phones...)
			mix_tones = append(mix_tones, en_tones...)
			mix_word2ph = append(mix_word2ph, en_word2ph...)
		}
		if segment.Type == TypePunctuation{
			sentence := segment.Content
			filteredText += sentence
			for _, r := range sentence {
				//fmt.Printf("添加标点r: %v\n", string(r))
				mix_phones = append(mix_phones, string(r))
				mix_tones = append(mix_tones, 0)
				mix_word2ph = append(mix_word2ph, 1)
			}
		}
	}

	//首尾添加下划线
	mix_phones = append(mix_phones, "_")
	mix_tones = append(mix_tones, 0)
	mix_word2ph = append(mix_word2ph, 1)

	return 	mix_phones ,mix_tones, mix_word2ph, filteredText
}


func Mandaren_g2p(zh_text string) ([]string, []int, []int) {
	//// 加载分词词典

	// fmt.Println("测试完毕")

	phones := []string{}
	tones := []int{}
	word2ph := []int{}

	pys := pinyinSentenceDict.Convert(zh_text, " ").ASCII()
	for _, py := range strings.Split(pys, " "){
		initial := Get_initial(py)
		final ,tone := Get_final_tone(py)
		//fmt.Println("声母韵母提取：", initial, final, tone)
		//fmt.Println("py声母韵母长度：", len(initial), len(final))

		phoneCountPerword := 0
		if len(initial) > 0 {
			phones = append(phones, initial)
			tones = append(tones, tone)
			phoneCountPerword++
		}  
		if len(final) > 0 {
			phones = append(phones, final)
			tones = append(tones, tone)
			phoneCountPerword++
		}
		word2ph = append(word2ph, phoneCountPerword)

		if phoneCountPerword == 0 {
			//fmt.Printf("汉字: %c 没有拼音\n", char)
		}
	}

	return phones, tones, word2ph
}

func MandarenPinyinTest() {
	// text := "我是中国人，最近有点焦虑"
	// args := pinyin.NewArgs()
	// result := []string{}

	// for _, r := range text {
	// 	hz := string(r)
	// 	// 1. 提取声母
	// 	args.Style = pinyin.Initials
	// 	initials := pinyin.Pinyin(hz, args)
	// 	if len(initials) > 0 && initials[0][0] != "" {
	// 		result = append(result, initials[0][0])
	// 	}

	// 	// 2. 提取韵母 (不带声调)
	// 	args.Style = pinyin.Finals
	// 	finals := pinyin.Pinyin(hz, args)
	// 	if len(finals) > 0 && finals[0][0] != "" {
	// 		result = append(result, finals[0][0])
	// 	}
        
    //     // 如果是非汉字字符（如逗号），手动处理
    //     if len(initials) == 0 {
    //         result = append(result, hz)
    //     }
	// }
	// fmt.Println(result)

}