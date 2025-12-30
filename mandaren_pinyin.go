package main

import (
	"strings"
	"regexp"
	"strconv"
	pinyin_sentence "github.com/Lofanmi/pinyin-golang/pinyin"
)


var pinyinSentenceDict *pinyin_sentence.Dict



func Mandaren_pinyinresourcePreload() {
	pinyinSentenceDict = pinyin_sentence.NewDict()
}



// 声母表
var initialArray = strings.Split(
	"b,p,m,f,d,t,n,l,g,k,h,j,q,x,r,zh,ch,sh,z,c,s",
	",",
)
var reFinalExceptions = regexp.MustCompile("^(j|q|x)(ū|ú|ǔ|ù)$")
var reFinal2Exceptions = regexp.MustCompile("^(j|q|x)u(\\d?)$")
var finalExceptionsMap = map[string]string{
	"ū": "ǖ",
	"ú": "ǘ",
	"ǔ": "ǚ",
	"ù": "ǜ",
}
// 获取单个拼音中的声母
func Get_initial(p string) string {
	s := ""
	for _, v := range initialArray {
		if strings.HasPrefix(p, v) {
			s = v
			break
		}
	}
	return s
}

// 获取单个拼音中的韵母
func Get_final(p string) (string) {
	n := Get_initial(p)
	if n == "" {
		return handleYW(p)
	}

	// 特例 j/q/x
	matches := reFinalExceptions.FindStringSubmatch(p)
	// jū -> jǖ
	if len(matches) == 3 && matches[1] != "" && matches[2] != "" {
		v, _ := finalExceptionsMap[matches[2]]
		return v
	}
	// ju -> jv, ju1 -> jv1
	p = reFinal2Exceptions.ReplaceAllString(p, "${1}v$2")
	final := strings.Join(strings.SplitN(p, n, 2), "")


	return final
}

func Get_final_tone(p string) (string, int) {
	final := Get_final(p)

	if len(final) <= 1 {
		return final, 0
	}
	last := final[len(final)-1]
	if last < '0' || last > '9' {
		return final, 0 // 无数字
	}else{
		final = final[:len(final)-1]
	}
	tone := int(last - '0')

	return final, tone
}

// 处理 y, w
func handleYW(p string) string {
	// 特例 y/w
	if strings.HasPrefix(p, "yu") {
		p = "v" + p[2:] // yu -> v
	} else if strings.HasPrefix(p, "yi") {
		p = p[1:] // yi -> i
	} else if strings.HasPrefix(p, "y") {
		p = "i" + p[1:] // y -> i
	} else if strings.HasPrefix(p, "wu") {
		p = p[1:] // wu -> u
	} else if strings.HasPrefix(p, "w") {
		p = "u" + p[1:] // w -> u
	}
	return p
}


func Mandaren_pinyin(zh_text string) []map[string]string {
	retPinyins := []map[string]string{}	

	pys := pinyinSentenceDict.Convert(zh_text, " ").ASCII()
	for _, py := range strings.Split(pys, " "){
		initial := Get_initial(py)
		final ,tone := Get_final_tone(py)
		//fmt.Println("声母韵母提取：", initial, final, tone)
		//fmt.Println("py声母韵母长度：", len(initial), len(final))
		itemMap := map[string]string{
			"initial": initial,
			"final": final,
			"tone": strconv.Itoa(tone),
		}
		retPinyins = append(retPinyins, itemMap)
		
	}
	return retPinyins
}