package mdata

import "math"

// 通用的分页返回
type PageResp struct {
	PageNum      int         `json:"pageNum"`
	PageSize     int         `json:"pageSize"`
	Total        int64       `json:"total"`
	List         interface{} `json:"list"`
	TotalPage    int         `json:"totalPage"`
	PrePage      int         `json:"prePage"`
	NextPage     int         `json:"nextPage"`
	ChatCategory int         `json:"category"`     //0表所有 1表聊天 2表晒单
	CategoryType int         `json:"categoryType"` //0 所有 如果 category晒单 ，type -1为普通单 2 为大单
}

type ResultPack struct {
	List     *PageResp
	ErrCode  int
	ErrorMsg string
}

func (p *PageResp) Paginator(list interface{}, page, size int, total int64) *PageResp {

	var (
		prePage  int
		nextPage int
	)

	if size <= 0 {
		size = 1
	}

	if total <= 0 {
		total = 1
	}

	if page <= 0 {
		page = 1
	}

	totalPages := int(math.Ceil(float64(total) / float64(size))) //page总数
	nextPage = page + 1

	if page >= totalPages {
		page = totalPages
		nextPage = page
	}

	prePage = int(math.Max(float64(1), float64(page-1)))
	p.PageNum = page
	p.PageSize = size
	p.List = list
	p.TotalPage = totalPages
	p.PrePage = prePage
	p.NextPage = nextPage
	p.Total = total
	return p
}
