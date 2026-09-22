package service

import (
	"math/big"
	"regexp"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

var versionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,49}$`)

func GetDefaultVersion() string { return "v1" }

// GenerateNextVersion 递增末尾数字；没有数字后缀则追加 .1。
// 示例：v1 -> v2，v1.0.9 -> v1.0.10，release -> release.1。
// 日期样式只是版本标签，不按日历进位；超长结果由校验或数据库拒绝。
func GenerateNextVersion(current string) string {
	if current == "" {
		return GetDefaultVersion()
	}
	start := len(current)
	for start > 0 && current[start-1] >= '0' && current[start-1] <= '9' {
		start--
	}
	if start == len(current) {
		return current + ".1"
	}
	number, _ := new(big.Int).SetString(current[start:], 10)
	return current[:start] + number.Add(number, big.NewInt(1)).String()
}

func ValidateVersion(version string) error {
	if !versionPattern.MatchString(version) {
		return errors.New(errors.ErrCodeInvalidParam, "版本号须为1至50个字符，以字母或数字开头，仅包含字母、数字、点、下划线和连字符")
	}
	return nil
}
