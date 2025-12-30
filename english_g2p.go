package main



import (
	// "bytes"
    // "io"
    "time"
	// "encoding/json"
	"fmt"
	// "io/ioutil"
	// "net/http"
	// //"net/url"
	"strconv"
	"strings"
	"slices"
	"github.com/nlpodyssey/gopickle/types"
	"github.com/nlpodyssey/gopickle/pickle"
	"github.com/agnivade/levenshtein"
)

var cmudictCache map[string]*types.List
var cmudictCacheKeys []string

func EnglishResourcePreload() {
	if cmudictCache == nil {
		loadEnglishG2PDict()
	}
}

func loadEnglishG2PDict() {
	fmt.Println("开始加载英语cmudict...")
	start := time.Now()

	foo, err := pickle.Load("cmudict_cache.pickle") 
	if err != nil {
		fmt.Printf("load pickle file failed: %v\n", err)
	}
	cmudictFoo := foo.(*types.Dict)

	//fmt.Printf("load pickle file success: %T \n", cmudictCache)

	//keys := cmudictFoo.Keys()
	dictLen := len(*cmudictFoo)
	cmudictCacheKeys = make([]string, 0, dictLen)
	cmudictCache = make(map[string]*types.List, dictLen)

	for _, entry := range *cmudictFoo {
		// entry 结构通常是 {Key: interface{}, Value: interface{}}
		if s, ok := entry.Key.(string); ok {
			cmudictCacheKeys = append(cmudictCacheKeys, s)
			
			// 直接从 entry.Value 取值，不要再调用 .Get(v) !
			if val, ok := entry.Value.(*types.List); ok {
				cmudictCache[s] = val
			}
		}
	}
	slices.Sort(cmudictCacheKeys)
	//fmt.Printf("cmudictCacheKeys: %v\n", cmudictCacheKeys)

	elapsed := time.Since(start)
	fmt.Printf("加载cmudict完成条数: %d (耗时: %v)\n", dictLen, elapsed)
	//test, _ := cmudictCache.Get("WHITEOOK")
	//fmt.Printf("test: %v\n", test)	

}

func FindClosestEnglishWord(target string) string {
	closest := cmudictCacheKeys[0]
	// 计算初始最小距离
	minDist := levenshtein.ComputeDistance(target, cmudictCacheKeys[0])

	for _, word := range cmudictCacheKeys[1:] {
		dist := levenshtein.ComputeDistance(target, word)
		// 如果距离更小，则更新最匹配项
		if dist < minDist {
			minDist = dist
			closest = word
		}
		// 如果距离为0，说明完全匹配，直接返回
		if minDist == 0 {
			break
		}
	}

	return closest
}

func split_phone_tone(phonetone string) ( string,  int) {
	phonePart := phonetone
	tone := 0
	if strings.HasSuffix(phonetone, "0") ||strings.HasSuffix(phonetone, "1") || strings.HasSuffix(phonetone, "2") || strings.HasSuffix(phonetone, "3") || strings.HasSuffix(phonetone, "4") {
		tonePart := phonetone[len(phonetone)-1:]
		phonePart = phonetone[:len(phonetone)-1]
		tone, _ = strconv.Atoi(tonePart) 
		tone++
	}else{
		tone = 0
	}
	return strings.ToLower(phonePart), tone
}

func English_g2p(text string, bertExtractor *BERTFeatureExtractor) ([]string, []int, []int) {
	if cmudictCache == nil {
		loadEnglishG2PDict()
	}

	en_phones := []string{}
	en_tones := []int{}
	en_word2ph := []int{} //每个token 对应的音素数量


	englis_tokens := bertExtractor.Tokenize(text)
	//fmt.Println("english tokens:", englis_tokens)

	//对english token进行分组
	groups := []interface{}{}
	for _, token := range englis_tokens {
		if !strings.HasPrefix(token, "##") {
			wordparts := []string{token}
			groups = append(groups, wordparts)
		}else{
			lastGroup := groups[len(groups)-1].([]string)
				
			curWordPart := strings.TrimPrefix(token, "##")
			groups[len(groups)-1] = append(lastGroup, curWordPart)
		}
		
	}
	//fmt.Println("english groups:", groups)

	//对每个group进行g2p
	for _, wordparts := range groups {
		word := strings.Join(wordparts.([]string), "")
		wordUp := strings.ToUpper(word)

		var cmuPhones *types.List
		if val, ok := cmudictCache[wordUp]; ok {
			//fmt.Printf("%v g2p: %v\n", word,val)
			cmuPhones = val
		}else{
			fmt.Printf("g2p单词表没有: %v\n", wordUp)	
			start := time.Now()
			closest := FindClosestEnglishWord(wordUp)
			elapsed := time.Since(start)
			fmt.Printf("找最相近的: %v (耗时: %v)\n", closest, elapsed)
			//fmt.Printf("closest: %v\n", closest)
			if closestVal, ok := cmudictCache[closest]; ok {
				fmt.Printf("%v g2p: %v\n", closest, closestVal)
				cmuPhones = closestVal
			}else{
				fmt.Printf("g2p单词表没有: %v\n", closest)
			}
		}

		oneword_phone_count := 0
		for i := 0; i < cmuPhones.Len(); i++ {
			cmuPhone := cmuPhones.Get(i).(*types.List)
			oneword_phone_count += cmuPhone.Len()
			for j := 0; j < cmuPhone.Len(); j++ {
				_cmuPhone := cmuPhone.Get(j).(string)
				phonePart, tone := split_phone_tone(_cmuPhone)
				//fmt.Printf("%v %v\n", phonePart, tone)
				en_phones = append(en_phones, phonePart)
				en_tones = append(en_tones, tone)
			}
		}

		oneword_token_count := len(wordparts.([]string))

		// 一个单词有多少个token, 音素，将音素数量均分到每个token上
		// 最终效果是每个token对应多少音素
		oneword_word2ph :=  make([]int, oneword_token_count)
		// 2. 迭代分配每一个音素
		for i := 0; i < oneword_phone_count; i++ {
			// 3. 寻找当前分配值最小的索引
			minIndex := 0
			minTasks := oneword_word2ph[0]

			for j := 1; j < oneword_token_count; j++ {
				if oneword_word2ph[j] < minTasks {
					minTasks = oneword_word2ph[j]
					minIndex = j
				}
			}

			// 4. 给分配最少的 Token 增加一个音素
			oneword_word2ph[minIndex]++
		}

		// oneword_word2ph 拼接到 en_word2ph
		en_word2ph = append(en_word2ph, oneword_word2ph...)

	}
	
	return 	en_phones ,en_tones, en_word2ph
}
