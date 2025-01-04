package openfigi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"sync"

	"golang.org/x/exp/constraints"
)

// ========================= PACKAGE CONFIG =========================
type mutexStruct[T any] struct {
	sync.RWMutex
	value T
}

// 🔗 BaseURL
var apiUrl mutexStruct[string]

func SetAPIBaseUrl(url string) {
	apiUrl.Lock()
	defer apiUrl.Unlock()
	apiUrl.value = url
}

func APIBaseUrl() string {
	apiUrl.RLock()
	defer apiUrl.RUnlock()
	return apiUrl.value
}

// 🔒 AUTH
// Could have used singleton, but maybe cycling API keys.
var apiKey mutexStruct[string]

func SetAPIKey(url string) {
	apiKey.Lock()
	defer apiKey.Unlock()
	apiKey.value = url
}

func APIKey() string {
	apiKey.RLock()
	defer apiKey.RUnlock()
	return apiKey.value
}

// ========================= TYPEs =========================

type interval[T constraints.Ordered] [2]T

type validator interface {
	validate() error
}

// ========================= REQUESTS =========================

type BaseItem struct {
	// Exchange code (cannot use in conjunction with `micCode`).
	// See https://api.openfigi.com/v3/mapping/values/exchCode
	ExchCode string `json:"exchCode,omitempty"`
	// ISO market identification code(MIC) (cannot use in conjunction with `exchCode`).
	// See https://api.openfigi.com/v3/mapping/values/micCode
	MicCode string `json:"micCode,omitempty"`
	// Currency.
	// See https://api.openfigi.com/v3/mapping/values/currency
	Currency string `json:"currency,omitempty"`
	// Market sector description.
	// See https://api.openfigi.com/v3/mapping/values/marketSecDes
	MarketSecDes string `json:"marketSecDes,omitempty"`
	// Security type.
	// See https://api.openfigi.com/v3/mapping/values/securityType
	SecurityType string `json:"securityType,omitempty"`
	// An alternative security type. `securityType2` is typically less specific than `securityType`.
	// Use `marketSecDes` if `securityType2` is not available.
	// See https://api.openfigi.com/v3/mapping/values/securityType2
	SecurityType2 string `json:"securityType2,omitempty"`
	// `true` to include equity instruments that are not listed on an exchange.
	IncludeUnlistedEquities bool `json:"includeUnlistedEquities,omitempty"`
	// Option type. Values: "Call" | "Put"
	OptionType string `json:"optionType,omitempty"`
	// Strike price interval, [a, b], where a, b are Numbers or null.
	// At least one entry must be a Number. When both are Numbers, a <= b.
	// [a, null]: [a, ∞); [null, b]: (-∞, b].
	Strike *interval[float64] `json:"strike,omitempty"`
	// Contract size interval, [a, b], where a, b are Numbers or null.
	// At least one entry must be a Number. When both are Numbers, a <= b.
	// [a, null]: [a, ∞); [null, b]: (-∞, b].
	ContractSize *interval[float64] `json:"contractSize,omitempty"`
	// Coupon interval, [a, b], where a, b are Numbers or null.
	// At least one entry must be a Number. When both are Numbers, a <= b.
	// [a, null]: [a, ∞); [null, b]: (-∞, b].
	Coupon *interval[float64] `json:"coupon,omitempty"`
	// Expiration date interval, [a, b], where a, b are date Strings [YYYY-MM-DD] or null.
	// At least one entry must be a date String.
	// When both are date String, a and b are no more than 1 year apart.
	// [a, null]: [a, a + (1Y)]; [null, b]: [b - (1Y), b].
	// **Requirement**: `securityType2` is `Option`.
	Expiration *interval[string] `json:"expiration,omitempty"`
	// Maturity interval, [a, b], where a, b are date Strings [YYYY-MM-DD] or null.
	// At least one entry must be a date String.
	// When both are date String, a and b are no more than 1 year apart.
	// [a, null]: [a, a + (1Y)]; [null, b]: [b - (1Y), b].
	// **Requirement**: `securityType2` is `Pool`.
	Maturity *interval[string] `json:"maturity,omitempty"`
	// State code.
	// See https://api.openfigi.com/v3/mapping/values/stateCode
	StateCode string `json:"stateCode,omitempty"`
}

// Usage:
//
//	builder := BaseItem{}.GetBuilder()
func (BaseItem) GetBuilder() BaseItemBuilder {
	return BaseItemBuilder{}
}

func (item *BaseItem) validate() error {
	switch {
	case item.ExchCode != "" && !exchCodeSet.Has(item.ExchCode):
		return fmt.Errorf("bad `exchCode`. See: %s", valuesUrl("exchCode"))
	case item.MicCode != "" && !micCodeSet.Has(item.MicCode):
		return fmt.Errorf("bad `micCode`. See: %s", valuesUrl("micCode"))
	case item.Currency != "" && !currencySet.Has(item.Currency):
		return fmt.Errorf("bad `currency`. See: %s", valuesUrl("currency"))
	case item.MarketSecDes != "" && !marketSecDesSet.Has(item.MarketSecDes):
		return fmt.Errorf("bad `marketSecDes`. See: %s", valuesUrl("marketSecDes"))
	case item.SecurityType != "" && !securityTypeSet.Has(item.SecurityType):
		return fmt.Errorf("bad `securityType`. See: %s", valuesUrl("securityType"))
	case item.SecurityType2 != "" && !securityType2Set.Has(item.SecurityType2):
		return fmt.Errorf("bad `securityType2`. See: %s", valuesUrl("securityType2"))
	case item.StateCode != "" && !stateCodeSet.Has(item.StateCode):
		return fmt.Errorf("bad `stateCode`. See: %s", valuesUrl("stateCode"))
	}

	// exchCode and micCode cannot coexist
	if item.ExchCode != "" && item.MicCode != "" {
		return fmt.Errorf("cannot use `exchCode` and `micCode` together")
	}

	// Validate intervals
	for _, interval := range []validator{item.Strike, item.ContractSize, item.Coupon, item.Expiration, item.Maturity} {
		// This is weird, somehow checking nil of interface have some quirks
		if reflect.ValueOf(interval).Kind() == reflect.Ptr && !reflect.ValueOf(interval).IsNil() {
			if err := interval.validate(); err != nil {
				return err
			}
		}
	}

	// Only option has expiration
	if !(item.SecurityType2 == "Option") && item.Expiration != nil {
		return fmt.Errorf("`expiration` is only valid for `Option`")
	}

	// Only pool has maturity
	if !(item.SecurityType2 == "Pool") && item.Maturity != nil {
		return fmt.Errorf("`maturity` is only valid for `Pool`")
	}

	return nil
}

func (b_item *BaseItem) AsMappingItem(idType string, value any) (item MappingItem, err error) {
	item = MappingItem{
		BaseItem: *b_item,
		Type:     idType,
		Value:    value,
	}
	err = item.validate()
	return
}

// MappingItem

type MappingItem struct {
	// BaseRequest fields
	BaseItem
	// Type of third party identifier. See https://www.openfigi.com/api#v3-idType-values
	// **Requirement**: For `BASE_TICKER` and `ID_EXCH_SYMBOL`, `securityType2` must be provided.
	Type string `json:"idType"`
	// The value for the represented third party identifier
	Value any `json:"idValue"`
}

// Usage:
//
//	builder := MappingItem{}.GetBuilder()
func (MappingItem) GetBuilder(idType string, value any) MappingItemBuilder {
	return MappingItemBuilder{
		BaseItemBuilder: BaseItem{}.GetBuilder(),
		item: MappingItem{
			Type:  idType,
			Value: value,
		},
	}
}

func (item *MappingItem) validate() error {
	if err := item.BaseItem.validate(); err != nil {
		return err
	}

	if !idTypeSet.Has(item.Type) {
		return fmt.Errorf("bad `idType`. See: %s", valuesUrl(item.Type))
	}

	if (item.Type == "BASE_TICKER" || item.Type == "ID_EXCH_SYMBOL") &&
		item.SecurityType2 == "" {
		return fmt.Errorf("`securityType2` must be provided for `BASE_TICKER` and `ID_EXCH_SYMBOL`")
	}

	return nil
}

func (m_item *MappingItem) AsBaseItem() (item BaseItem, err error) {
	item = m_item.BaseItem
	err = item.validate()
	return
}

type MappingRequest []MappingItem

func (req *MappingRequest) FromMappingItemBuilders(builders ...MappingItemBuilder) error {
	for _, builder := range builders {
		item, err := builder.Build()
		if err != nil {
			return err
		}
		*req = append(*req, item)
	}
	return nil
}

// ========================= RESPONSES =========================

type FIGIObject struct {
	FIGI                string `json:"figi,omitempty"`
	SecurityType        string `json:"securityType,omitempty"`
	MarketSector        string `json:"marketSector,omitempty"`
	Ticker              string `json:"ticker,omitempty"`
	Name                string `json:"name,omitempty"`
	UniqueID            string `json:"uniqueID,omitempty"`
	ExchangeCode        string `json:"exchCode,omitempty"`
	ShareClassFIGI      string `json:"shareClassFIGI,omitempty"`
	CompositeFIGI       string `json:"compositeFIGI,omitempty"`
	SecurityType2       string `json:"securityType2,omitempty"`
	SecurityDescription string `json:"securityDescription,omitempty"`
	Metadata            string `json:"metadata,omitempty"` // Exists when API is unable to show non-FIGI fields
}

type SingleMappingResponse struct {
	Data    []FIGIObject `json:"data"`
	Error   string       `json:"error,omitempty"`
	Warning []string     `json:"warning,omitempty"`
}

type SearchResponse struct {
	Data     []FIGIObject `json:"data"`
	Error    string       `json:"error,omitempty"`
	NextHash string       `json:"next,omitempty"`
	baseitem BaseItem
	query    string
}

type FilterResponse struct {
	SearchResponse
	Total int `json:"total"`
}

// ========================= API =========================

type searchOrFilterRequest struct {
	BaseItem
	Query string `json:"query,omitempty"`
	Start string `json:"start,omitempty"`
}

// Calls
func (m_req MappingRequest) Fetch() (res []SingleMappingResponse, err error) {
	jsonData, err := json.Marshal(m_req)
	if err != nil {
		return
	}
	req, _ := http.NewRequest("POST", APIBaseUrl()+"/mapping", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if key := APIKey(); key != "" {
		req.Header.Set("X-OPENFIGI-APIKEY", key)
	}
	slog.Debug(fmt.Sprintf("POST %s", APIBaseUrl()+"/mapping"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	} else if details, ok := httpStatusMap[resp.StatusCode]; ok {
		slog.Error(fmt.Sprintf("%d — %s", resp.StatusCode, details))
		err = fmt.Errorf("%d", resp.StatusCode)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &res)
	return
}

func postBaseItem[T any](endpoint string, item BaseItem, query string, start string) (res T, err error) {
	jsonData, err := json.Marshal(searchOrFilterRequest{
		BaseItem: item,
		Query:    query,
		Start:    start,
	})
	if err != nil {
		return
	}
	req, _ := http.NewRequest("POST", APIBaseUrl()+endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if key := APIKey(); key != "" {
		req.Header.Set("X-OPENFIGI-APIKEY", key)
	}
	slog.Debug(fmt.Sprintf("POST %s", APIBaseUrl()+endpoint))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	} else if details, ok := httpStatusMap[resp.StatusCode]; ok {
		slog.Error(fmt.Sprintf("%d — %s", resp.StatusCode, details))
		err = fmt.Errorf("%d", resp.StatusCode)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &res)
	return
}

func (item BaseItem) Search(query string, start string) (res SearchResponse, err error) {
	res, err = postBaseItem[SearchResponse]("/search", item, query, start)
	res.baseitem = item
	res.query = query

	return
}

func (searchRes SearchResponse) Next() (SearchResponse, error) {
	if searchRes.NextHash == "" {
		return SearchResponse{}, fmt.Errorf("no more results")
	}
	return searchRes.baseitem.Search(searchRes.query, searchRes.NextHash)
}

func (item BaseItem) Filter(query string, start string) (res FilterResponse, err error) {
	res, err = postBaseItem[FilterResponse]("/filter", item, query, start)
	res.baseitem = item
	res.query = query

	return
}

func (filterRes FilterResponse) Next() (FilterResponse, error) {
	if filterRes.NextHash == "" {
		return FilterResponse{}, fmt.Errorf("no more results")
	}
	return filterRes.baseitem.Filter(filterRes.query, filterRes.NextHash)
}

// ========================= AUXILIARY FUNC =========================

func valuesUrl(property string) string {
	return APIBaseUrl() + "/mapping/values/" + property
}

func init() {
	SetAPIBaseUrl("https://api.openfigi.com/v3")
}

// ========================= CODEGEN =========================
//go:generate go run gen/gen.go
