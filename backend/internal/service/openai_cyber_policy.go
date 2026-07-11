package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const opsCyberPolicyKey = "ops_cyber_policy"

type CyberPolicyMark struct {
	Code           string
	Message        string
	Body           string
	UpstreamStatus int
	UpstreamInTok  int
	UpstreamOutTok int
}

func MarkOpsCyberPolicy(c *gin.Context, mark CyberPolicyMark) {
	if c == nil || GetOpsCyberPolicy(c) != nil {
		return
	}
	mark.Code = "cyber_policy"
	mark.Message = strings.TrimSpace(mark.Message)
	mark.Body = strings.TrimSpace(mark.Body)
	c.Set(opsCyberPolicyKey, &mark)
}

func GetOpsCyberPolicy(c *gin.Context) *CyberPolicyMark {
	if c == nil {
		return nil
	}
	v, ok := c.Get(opsCyberPolicyKey)
	mark, valid := v.(*CyberPolicyMark)
	if !ok || !valid || mark == nil {
		return nil
	}
	return mark
}

func ClearOpsCyberPolicy(c *gin.Context) {
	if c != nil {
		c.Set(opsCyberPolicyKey, (*CyberPolicyMark)(nil))
	}
}

func detectOpenAICyberPolicy(payload []byte) (bool, string, string) {
	code := gjson.GetBytes(payload, "error.code").String()
	if code == "" {
		code = gjson.GetBytes(payload, "response.error.code").String()
	}
	if !strings.EqualFold(strings.TrimSpace(code), "cyber_policy") {
		return false, "", ""
	}
	message := gjson.GetBytes(payload, "error.message").String()
	if message == "" {
		message = gjson.GetBytes(payload, "response.error.message").String()
	}
	return true, "cyber_policy", strings.TrimSpace(message)
}
