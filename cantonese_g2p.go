package main

import (
	"bytes"
    "io"
    "time"
	"encoding/json"
	"fmt"
	// "io/ioutil"
	"net/http"
	//"net/url"
	"strconv"
	"strings"
	"github.com/liuzl/gocc"
	"github.com/ZingYao/chinese_number"
)

var s2hk *gocc.OpenCC

func CantoneseResourcePreload()(){
	if s2hk == nil {
		s2t, _ := gocc.New("s2hk")
		s2hk = s2t
	}

}

func CantoneseMix_g2p(text string, bertExtractor *BERTFeatureExtractor) ([]string, []int, []int, string) {


	mix_phones := []string{"_"}
	mix_tones := []int{0}
	mix_word2ph := []int{1}

	// 处理中英文特殊符号混合语句

	// 将中文、 英文、数字、符号 分离成单独顺序片段
	segments := SplitText(text)

	chineseSentences := map[string]string{}

	filteredText := ""

	// 将中文单独拿出来，统一调用粤语拼音分词接口
	for index, segment := range segments {
		indexStr := strconv.Itoa(index)
		if segment.Type == TypeChinese{
			// 统一转换为香港繁体
	        sentence, _ := s2hk.Convert(segment.Content)
			chineseSentences[indexStr] = sentence
			filteredText += sentence
		}
		if segment.Type == TypeNumber{
			// 数字简单转换为中文模式，具体取决于业务模式，如钱币 日期 时间，可在前端进行处理
			num, _ := strconv.ParseInt(segment.Content, 10, 64)
			zhstr := chinese_number.Number2Simplified(num)
			fmt.Printf("number:%d to simplified chinese:%q\n",num,zhstr)
			chineseSentences[indexStr] = zhstr
			filteredText += zhstr
		} 
		if segment.Type == TypeEnglish{
			filteredText += segment.Content
		}
		if segment.Type == TypePunctuation{
			filteredText += segment.Content
		}
	}
	
	var jyupinyinMap map[string]interface{}
	if len(chineseSentences) > 0 {
		start := time.Now()
		jyupinyinMap = request_jyuping(chineseSentences)
		elapsed := time.Since(start)
		fmt.Printf("请求粤语拼音接口 (耗时: %v)\n", elapsed)
	}

	//fmt.Println("jyupinyinList:",jyupinyinList)
	//真正处理文本
	for index, segment := range segments {
		indexStr := strconv.Itoa(index)
		if segment.Type == TypeChinese{
			
			jyupinyinList := jyupinyinMap[indexStr].([]interface{})
			phones, tones, word2ph := Cantonese_g2p(jyupinyinList)
			mix_phones = append(mix_phones, phones...)
			mix_tones = append(mix_tones, tones...)
			mix_word2ph = append(mix_word2ph, word2ph...)
		}
		if segment.Type == TypeNumber{
			// 数字简单转换为中文模式，具体取决于业务模式，如钱币 日期 时间，可在前端进行处理
			jyupinyinList := jyupinyinMap[indexStr].([]interface{})
			phones, tones, word2ph := Cantonese_g2p(jyupinyinList)
			mix_phones = append(mix_phones, phones...)
			mix_tones = append(mix_tones, tones...)
			mix_word2ph = append(mix_word2ph, word2ph...)

		}
		if segment.Type == TypeEnglish{
			en_phones ,en_tones, en_word2ph := English_g2p(segment.Content, bertExtractor)
			mix_phones = append(mix_phones, en_phones...)
			mix_tones = append(mix_tones, en_tones...)
			mix_word2ph = append(mix_word2ph, en_word2ph...)
		}
		if segment.Type == TypePunctuation{
			// 符号直接添加到phones，tones 设为0，word2ph 设为1
			for _, r := range segment.Content {
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



func Cantonese_g2p(jyupinyinList []interface{}) ([]string, []int, []int) {
	//此函数

	phones := []string{}
	tones := []int{}
	word2ph := []int{}

	for _, item := range jyupinyinList {
		jyupingMap := item.(map[string]interface{})
		//zhchar := jyupingMap["char"].(string)
		initial_list := jyupingMap["initial_list"].([]interface{})
		//fmt.Println("zhchar:",zhchar)
		for _, initialItem := range initial_list {
			//fmt.Println("initial:", initial["initial"])
			initialItemDict := initialItem.(map[string]interface{})
			initial := initialItemDict["initial"].(string)
			nucleus := initialItemDict["nucleus"].(string)
			coda := initialItemDict["coda"].(string)
			tone := initialItemDict["tone"].(string)

			toneInt, _ := strconv.Atoi(tone)

			//fmt.Printf("initial: %v, nucleus: %v, coda: %v, tone: %v\n", initial, nucleus, coda, tone)
			phoneCountPerword := 0
			if !IsEmptyOrWhitespace(initial) {
				phones = append(phones, initial)
				tones = append(tones, toneInt)
				phoneCountPerword++
			}
			if !IsEmptyOrWhitespace(nucleus) {
				phones = append(phones, nucleus)
				tones = append(tones, toneInt)
				phoneCountPerword++
			}
			if !IsEmptyOrWhitespace(coda) {
				phones = append(phones, coda)
				tones = append(tones, toneInt)
				phoneCountPerword++
			}
			if phoneCountPerword > 0 {
                word2ph = append(word2ph, phoneCountPerword)
			}

		}
	}


	return phones, tones, word2ph
}

func IsEmptyOrWhitespace(s string) bool {
    // 方法1: 使用strings.TrimSpace
    trimmed := strings.TrimSpace(s)
    return len(trimmed) == 0
    
    // 方法2: 手动遍历（更高效）
    // for _, r := range s {
    //     if !unicode.IsSpace(r) {
    //         return false
    //     }
    // }
    // return true
}

func request_jyuping(sentences map[string]string) map[string]interface{} {


	
	// 3. 创建HTTP客户端
    client := &http.Client{
        Timeout: 10 * time.Second, // 设置超时时间
    }

	// 将 sentences 转为 JSON 字符串
	jsonBytes, err := json.Marshal(sentences)
	if err != nil {
		fmt.Printf("sentences 转 JSON 失败: %v\n", err)
		return nil
	}
	// hkout := string(jsonBytes)
    
	// reqbodystr := fmt.Sprintf(`{"content": "%s"}`, hkout)

    // 4. 创建请求，使用 jsonBytes 作为请求体
    req, err := http.NewRequest(
        "POST",
        "http://127.0.0.1:48000/cantonese_split",
        bytes.NewReader(jsonBytes),
    )
    // 4. 创建请求
    if err != nil {
        fmt.Printf("创建请求失败: %v\n", err)
        
    }
    
    // 5. 设置请求头
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "MyGoClient/1.0")
    req.Header.Set("Authorization", "Bearer your-token-here") // 如果需要认证
    
    // 6. 发送请求
    resp, err := client.Do(req)
    if err != nil {
        fmt.Printf("请求失败: %v\n", err)
        
    }
    defer resp.Body.Close() // 确保关闭响应体
    
    // 7. 读取响应体
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Printf("读取响应失败: %v\n", err)
        
    }
    
    // 8. 输出响应信息
    //fmt.Printf("状态码: %d\n", resp.StatusCode)
    //fmt.Printf("响应头: %v\n", resp.Header)
   // fmt.Printf("响应体: %s\n", string(body))

    // // 9. 解析JSON响应（可选）
	var response map[string]interface{}
    if resp.StatusCode == http.StatusOK {
        
        if err := json.Unmarshal(body, &response); err == nil {
            //fmt.Printf("解析后的响应: %+v\n", response)
        }else{
			fmt.Printf("解析响应失败: %v\n", err)
		}
		//fmt.Printf("response : %v\n", response)

    }

	return response
}