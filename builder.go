package openfigi

import (
	"fmt"
	"math"
	"time"

	"golang.org/x/exp/constraints"
)

// ========================= BASE ITEM =========================
type BaseItemBuilder struct {
	item BaseItem
}

func (b *BaseItemBuilder) SetExchCode(exchCode string) *BaseItemBuilder {
	b.item.ExchCode = exchCode
	return b
}

func (b *BaseItemBuilder) SetMicCode(micCode string) *BaseItemBuilder {
	b.item.MicCode = micCode
	return b
}

func (b *BaseItemBuilder) SetCurrency(currency string) *BaseItemBuilder {
	b.item.Currency = currency
	return b
}

func (b *BaseItemBuilder) SetMarketSecDes(marketSecDes string) *BaseItemBuilder {
	b.item.MarketSecDes = marketSecDes
	return b
}

func (b *BaseItemBuilder) SetSecurityType(securityType string) *BaseItemBuilder {
	b.item.SecurityType = securityType
	return b
}

func (b *BaseItemBuilder) SetSecurityType2(securityType2 string) *BaseItemBuilder {
	b.item.SecurityType2 = securityType2
	return b
}

func (b *BaseItemBuilder) SetIncludeUnlistedEquities(include bool) *BaseItemBuilder {
	b.item.IncludeUnlistedEquities = include
	return b
}

func (b *BaseItemBuilder) SetOptionType(optionType string) *BaseItemBuilder {
	b.item.OptionType = optionType
	return b
}

// Usage:
//
//	builder.SetStrike([2]any{nil, 2})
func (b *BaseItemBuilder) SetStrike(strike [2]any) *BaseItemBuilder {
	strikeRange := intepretRange[float64](strike)
	b.item.Strike = &strikeRange
	return b
}

// Usage:
//
//	builder.SetContractSize([2]any{2, nil})
func (b *BaseItemBuilder) SetContractSize(contractSize [2]any) *BaseItemBuilder {
	contractSizeRange := intepretRange[float64](contractSize)
	b.item.ContractSize = &contractSizeRange
	return b
}

// Usage:
//
//	builder.SetCoupon([2]any{nil, 2})
func (b *BaseItemBuilder) SetCoupon(coupon [2]any) *BaseItemBuilder {
	couponRange := intepretRange[float64](coupon)
	b.item.Coupon = &couponRange
	return b
}

// Usage:
//
//	builder.SetExpiration([2]any{"2021-01-01", "2022-01-01"})
func (b *BaseItemBuilder) SetExpiration(expiration [2]any) *BaseItemBuilder {
	expirationRange := intepretRange[string](expiration)
	b.item.Expiration = &expirationRange
	return b
}

// Usage:
//
//	builder.SetMaturity([2]any{nil, "2022-01-01"})
func (b *BaseItemBuilder) SetMaturity(maturity [2]any) *BaseItemBuilder {
	maturityRange := intepretRange[string](maturity)
	b.item.Maturity = &maturityRange
	return b
}

func (b *BaseItemBuilder) SetStateCode(stateCode string) *BaseItemBuilder {
	b.item.StateCode = stateCode
	return b
}

func (b *BaseItemBuilder) Build() (item BaseItem, err error) {
	item = b.item
	err = item.validate()
	return
}

// ========================= MAPPING ITEM =========================

type MappingItemBuilder struct {
	BaseItemBuilder
	item MappingItem
}

func (m *MappingItemBuilder) Build() (item MappingItem, err error) {
	m.item.BaseItem = m.BaseItemBuilder.item

	item = m.item
	err = m.item.validate()
	return
}

// ========================= AUXILIARY FUNC =========================

func intepretRange[T constraints.Ordered](interval [2]interface{}) interval[T] {
	var zero T
	switch any(zero).(type) {
	case float64:
		if interval[0] == nil {
			interval[0] = math.Inf(-1)
		}
		if interval[1] == nil {
			interval[1] = math.Inf(1)
		}
	case string:
		if interval[0] == nil {
			interval[0] = ""
		}
		if interval[1] == nil {
			interval[1] = ""
		}
	}
	return [2]T{interval[0].(T), interval[1].(T)}
}

func (interval interval[T]) validate() error {
	var zero T
	switch any(zero).(type) {
	case float64:
		start, _ := any(interval[0]).(float64)
		end, _ := any(interval[1]).(float64)
		if math.IsInf(start, -1) && math.IsInf(end, 1) {
			return fmt.Errorf("interval cannot be [null, null]")
		} else {
			if start > end {
				return fmt.Errorf("bad interval: %v > %v", start, end)
			}
		}
	case string:
		start, _ := any(interval[0]).(string)
		end, _ := any(interval[1]).(string)
		if start == "" && end == "" {
			return fmt.Errorf("interval cannot be [null, null]")
		} else {
			if s, err := time.Parse(time.DateOnly, start); start != "" && err != nil {
				return fmt.Errorf("bad date format: %v", err)
			} else if e, err := time.Parse(time.DateOnly, end); end != "" && err != nil {
				return fmt.Errorf("bad date format: %v", err)
			} else if start != "" && end != "" && s.After(e) {
				return fmt.Errorf("bad interval: %v > %v", s, e)
			}
		}
	}

	return nil
}
