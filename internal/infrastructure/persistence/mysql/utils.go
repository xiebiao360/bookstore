package mysql

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// isDuplicateError 判断是否为MySQL唯一索引冲突错误
// MySQL错误码:
// - 1062: Duplicate entry 'xxx' for key 'yyy'
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// GORM v2的错误判断
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// 兼容检查:错误信息包含"Duplicate entry"
	return strings.Contains(err.Error(), "Duplicate entry")
}
