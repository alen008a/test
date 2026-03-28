package sego

import (
	"gorm.io/gorm"
	"log"
	"math"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/internal/glog"
	"msgPushSite/mdata"
	"msgPushSite/mdata/rediskey"
	"time"
)

var Sgmt Segmenter

func (seg *Segmenter) ReLoad() {
	var sgmt Segmenter
	sgmt.LoadDictionaryFromDB(Sgmt.db, Sgmt.table)
	Sgmt = sgmt
}

func Init(db *gorm.DB) {
	Sgmt.LoadDictionaryFromDB(db, rediskey.SensitiveTable)
	go Sgmt.Watch()
}

func (seg *Segmenter) LoadDictionaryFromDB(db *gorm.DB, table string) {
	var (
		err   error
		total int64
	)
	seg.db = db
	seg.table = table
	type (
		TokenSchema struct {
			Text      string `gorm:"column:word"`
			Frequency int    `gorm:"column:frequency"`
		}
	)
	seg.dict = NewDictionary()
	err = db.Table(table).Where("is_delete = 0").Count(&total).Error
	if err != nil {
		log.Fatalf("获取%s表总数量出错:%s\n", table, err.Error())
		return
	}
	max := int(math.Ceil(float64(total) / float64(perCount)))
	start := time.Now().Nanosecond()
	for i := 0; i < max; i++ {
		var tokens []TokenSchema
		err = db.Table(table).Select("word, frequency").
			Where("is_delete = 0").Offset(i * perCount).Limit(perCount).Find(&tokens).Error
		if err != nil {
			log.Fatalf("从数据库加载数据异常，异常:%s\n", err.Error())
		}

		for _, value := range tokens {
			seg.dict.addKeyword(value.Text)
			// 过滤频率太小的词
			if value.Frequency < minTokenFrequency {
				continue
			}
			// 将分词添加到字典中
			words := splitTextToWords([]byte(value.Text))
			token := Token{text: words, frequency: value.Frequency, pos: ""}
			seg.dict.addToken(token)
		}
	}
	seg.dict.buildNewTrie()
	end := time.Now().Nanosecond()
	glog.Infof("敏感词共计%d条，构建词库耗时%d毫秒", total, (end-start)/1e6)
	// 计算每个分词的路径值，路径值含义见Token结构体的注释
	logTotalFrequency := float32(math.Log2(float64(seg.dict.totalFrequency)))
	for i := range seg.dict.tokens {
		token := &seg.dict.tokens[i]
		token.distance = logTotalFrequency - float32(math.Log2(float64(token.frequency)))
	}

	// 对每个分词进行细致划分，用于搜索引擎模式，该模式用法见Token结构体的注释。
	for i := range seg.dict.tokens {
		token := &seg.dict.tokens[i]
		segments := seg.segmentWords(token.text, true)

		// 计算需要添加的子分词数目
		numTokensToAdd := 0
		for iToken := 0; iToken < len(segments); iToken++ {
			if len(segments[iToken].token.text) > 0 {
				numTokensToAdd++
			}
		}
		token.segments = make([]*Segment, numTokensToAdd)

		// 添加子分词
		iSegmentsToAdd := 0
		for iToken := 0; iToken < len(segments); iToken++ {
			if len(segments[iToken].token.text) > 0 {
				token.segments[iSegmentsToAdd] = &segments[iToken]
				iSegmentsToAdd++
			}
		}
	}

	log.Println("sego词典载入完毕")
}

func (seg *Segmenter) Watch() {

	sub := core.Subscribe(rediskey.SensitivePublish)
	for {
		select {
		case msg := <-sub.Channel():
			glog.Infof("reload dict signal received %v", msg)
			if msg == nil {
				glog.Infof("msg is nil")
				continue
			}
			var event KeywordEvent
			err := mdata.Cjson.UnmarshalFromString(msg.Payload, &event)
			if err != nil {
				glog.Infof("incorrect keyword change event. %v = ", err)
			} else {
				if event.Type == 1 {
					for _, word := range event.Words {
						seg.dict.addKeyword(word)
					}
					seg.dict.buildNewTrie()
				} else if event.Type == 2 {
					for _, word := range event.Words {
						seg.dict.removeKeyword(word)
					}
					seg.dict.buildNewTrie()
				}
			}
		}
	}
}

type KeywordEvent struct {
	Type  int      // 1增加 2删除
	Words []string //变动的敏感词
}
