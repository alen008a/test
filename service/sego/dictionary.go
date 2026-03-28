package sego

import (
	"fmt"
	"github.com/adamzy/cedar-go"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/internal/glog"
	"msgPushSite/lib/cache"
	"msgPushSite/mdata/rediskey"
	"regexp"
	"time"
)

// Dictionary结构体实现了一个字串前缀树，一个分词可能出现在叶子节点也有可能出现在非叶节点
type Dictionary struct {
	newTrie        Trie                //新Trie树
	trie           *cedar.Cedar        // Cedar 前缀树
	maxTokenLength int                 // 词典中最长的分词
	tokens         []Token             // 词典中所有的分词，方便遍历
	totalFrequency int64               // 词典中所有分词的频率之和
	words          map[string]struct{} //词典中所有的敏感词，方便动态调整
}

func NewDictionary() *Dictionary {
	return &Dictionary{trie: cedar.New(), words: make(map[string]struct{}, 1024)}
}

// 词典中最长的分词
func (dict *Dictionary) MaxTokenLength() int {
	return dict.maxTokenLength
}

// 词典中分词数目
func (dict *Dictionary) NumTokens() int {
	return len(dict.tokens)
}

// 词典中所有分词的频率之和
func (dict *Dictionary) TotalFrequency() int64 {
	return dict.totalFrequency
}

// 新trie树
func (dict *Dictionary) GetNewTrie() Trie {
	return dict.newTrie
}

// 释放资源
func (dict *Dictionary) Close() {
	dict.newTrie = nil
	dict.trie = nil
	dict.maxTokenLength = 0
	dict.tokens = nil
	dict.totalFrequency = int64(0)
	dict.words = nil
}

// 向词典中加入一个分词
func (dict *Dictionary) addToken(token Token) {
	bytes := textSliceToBytes(token.text)
	_, err := dict.trie.Get(bytes)
	if err == nil {
		return
	}

	dict.trie.Insert(bytes, dict.NumTokens())
	dict.tokens = append(dict.tokens, token)
	dict.totalFrequency += int64(token.frequency)
	if len(token.text) > dict.maxTokenLength {
		dict.maxTokenLength = len(token.text)
	}
}

func (dict *Dictionary) addKeyword(keyword string) {
	if _, ok := dict.words[keyword]; !ok {
		dict.words[keyword] = struct{}{}
	}
}

func (dict *Dictionary) removeKeyword(keyword string) {
	if _, ok := dict.words[keyword]; ok {
		delete(dict.words, keyword)
	}
}

func (dict *Dictionary) buildNewTrie() {
	trie := NewTrie([]string{})
	for word := range dict.words {
		trie.Add([]string{word})
	}
	trie.Build()
	dict.newTrie = trie
}

// 在词典中查找和字元组words可以前缀匹配的所有分词
// 返回值为找到的分词数
func (dict *Dictionary) lookupTokens(words []Text, tokens []*Token) (numOfTokens int) {
	var id, value int
	var err error
	for _, word := range words {
		id, err = dict.trie.Jump(word, id)
		if err != nil {
			break
		}
		value, err = dict.trie.Value(id)
		if err == nil {
			tokens[numOfTokens] = &dict.tokens[value]
			numOfTokens++
		}
	}
	return
}

// IsExistContinuousWord 是否存在连续的字母和数字
func (dict *Dictionary) IsExistContinuousWord(msg string) (bool, string) {

	number, err := cache.GetOrSet(rediskey.ShieldNumberSet, 5*time.Minute, func() (i interface{}, e error) {
		return core.GetKeyInt(false, rediskey.ShieldNumberSet)
	})
	if err != nil {
		glog.Errorf("查询消息位数屏蔽设置失败:%+v", err)
		number = 0
	}
	if number != 0 {
		//正则：是否存在连续的7个以上数字或字母或标点
		str := fmt.Sprintf("(\\pP|\\w){%d,}", number)
		exp := regexp.MustCompile(str)
		matchStrArr := exp.FindAllStringSubmatch(msg, -1)
		if len(matchStrArr) > 0 && len(matchStrArr[0]) > 0 {
			glog.Warnf("聊天信息：'%s'含有连续字符'%s'，过滤掉", msg, matchStrArr[0][0])
			return true, matchStrArr[0][0]
		}
	}
	return false, ""
}
