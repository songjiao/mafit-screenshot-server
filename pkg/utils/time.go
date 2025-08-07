package utils

import (
	"fmt"
	"time"
)

// GenerateTaskKey 基于当前时间生成任务键
func GenerateTaskKey(symbol, market string) string {
	now := time.Now()
	dateHour := now.Format("2006010215") // yyyyMMddHH
	return fmt.Sprintf("%s_%s_%s", symbol, market, dateHour)
}

// ParseSymbolMarket 解析股票代码和市场代码
func ParseSymbolMarket(symbolMarket string) (symbol, market string, err error) {
	// 格式: symbol.market
	// 例如: NVDA.us, AAPL.us
	if len(symbolMarket) == 0 {
		return "", "", fmt.Errorf("empty symbol.market")
	}

	// 查找最后一个点号
	lastDot := -1
	for i := len(symbolMarket) - 1; i >= 0; i-- {
		if symbolMarket[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot == -1 || lastDot == 0 || lastDot == len(symbolMarket)-1 {
		return "", "", fmt.Errorf("invalid symbol.market format")
	}

	symbol = symbolMarket[:lastDot]
	market = symbolMarket[lastDot+1:]

	return symbol, market, nil
}

// FormatDateTime 格式化日期时间
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05Z")
}

// EstimateCompletionTime 估算完成时间
func EstimateCompletionTime() time.Time {
	return time.Now().Add(5 * time.Minute)
}
