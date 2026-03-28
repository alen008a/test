package metadata

import (
	"msgPushSite/internal/context"
	"msgPushSite/mdata"
	"strconv"
)

func GetSiteId(c *context.Context) int {
	if c.SiteId != "" {
		siteId, _ := strconv.Atoi(c.SiteId)
		return siteId
	}
	s := c.Request.Header.Get(mdata.HeaderSite)
	siteId, _ := strconv.Atoi(s)
	return siteId
}

func GetSiteIdString(c *context.Context) string {
	if c.SiteId != "" {
		return c.SiteId
	}
	return c.Request.Header.Get(mdata.HeaderSite)
}
