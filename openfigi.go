package openfigi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"golang.org/x/exp/constraints"
)

const API_BASE_URL = "https://api.openfigi.com/v3"

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
	case !exchCodeSet.Has(item.ExchCode):
		return fmt.Errorf("bad `exchCode`. See: %s", valuesUrl(item.ExchCode))
	case !micCodeSet.Has(item.MicCode):
		return fmt.Errorf("bad `micCode`. See: %s", valuesUrl(item.MicCode))
	case !currencySet.Has(item.Currency):
		return fmt.Errorf("bad `currency`. See: %s", valuesUrl(item.Currency))
	case !marketSecDesSet.Has(item.MarketSecDes):
		return fmt.Errorf("bad `marketSecDes`. See: %s", valuesUrl(item.MarketSecDes))
	case !securityTypeSet.Has(item.SecurityType):
		return fmt.Errorf("bad `securityType`. See: %s", valuesUrl(item.SecurityType))
	case !securityType2Set.Has(item.SecurityType2):
		return fmt.Errorf("bad `securityType2`. See: %s", valuesUrl(item.SecurityType2))
	case !stateCodeSet.Has(item.StateCode):
		return fmt.Errorf("bad `stateCode`. See: %s", valuesUrl(item.StateCode))
	}

	// exchCode and micCode cannot coexist
	if item.ExchCode != "" && item.MicCode != "" {
		return fmt.Errorf("cannot use `exchCode` and `micCode` together")
	}

	// Validate intervals
	for _, interval := range []validator{item.Strike, item.ContractSize, item.Coupon, item.Expiration, item.Maturity} {
		if err := interval.validate(); err != nil {
			return err
		}
	}

	// Only option has expiration
	if item.SecurityType2 == "Option" && *item.Expiration != [2]string{"", ""} {
		return fmt.Errorf("`expiration` is only valid for `Option`")
	}

	// Only pool has maturity
	if item.SecurityType2 == "Pool" && *item.Maturity != [2]string{"", ""} {
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
	FIGI                string `json:"figi"`
	SecurityType        string `json:"securityType"`
	MarketSector        string `json:"marketSector"`
	Ticker              string `json:"ticker"`
	Name                string `json:"name"`
	UniqueID            string `json:"uniqueID"`
	ExchangeCode        string `json:"exchCode"`
	ShareClassFIGI      string `json:"shareClassFIGI"`
	CompositeFIGI       string `json:"compositeFIGI"`
	SecurityType2       string `json:"securityType2"`
	SecurityDescription string `json:"securityDescription"`
	Metadata            string `json:"metadata"` // Exists when API is unable to show non-FIGI fields
}

type SingleMappingResponse struct {
	Data    []FIGIObject `json:"data"`
	Error   string       `json:"error"`
	Warning []string     `json:"warning"`
}

type SearchResponse struct {
	Data     []FIGIObject `json:"data"`
	Error    string       `json:"error"`
	NextHash string       `json:"next"`
	baseitem BaseItem
	query    string
}

type FilterResponse struct {
	Data     []FIGIObject `json:"data"`
	Error    string       `json:"error"`
	NextHash string       `json:"next"`
	Total    int          `json:"total"`
	baseitem BaseItem
	query    string
}

// ========================= API =========================

// 🔒 AUTH
// Could have used singleton, but maybe cycling API keys.
type apiKeyManager struct {
	key string
	mu  sync.RWMutex
}

var apiKey apiKeyManager

func NewAPIKeyManager(key string) {
	apiKey = apiKeyManager{key: key}
}

func GetKey() string {
	apiKey.mu.RLock()
	defer apiKey.mu.RUnlock()
	return apiKey.key
}

func SetKey(key string) {
	apiKey.mu.Lock()
	defer apiKey.mu.Unlock()
	apiKey.key = key
}

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
	req, _ := http.NewRequest("POST", API_BASE_URL+"/mapping", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if key := GetKey(); key != "" {
		req.Header.Set("X-OPENFIGI-APIKEY", key)
	}
	slog.Debug(fmt.Sprintf("POST %s", API_BASE_URL+"/mapping"))

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
	req, _ := http.NewRequest("POST", API_BASE_URL+endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if key := GetKey(); key != "" {
		req.Header.Set("X-OPENFIGI-APIKEY", key)
	}
	slog.Debug(fmt.Sprintf("POST %s", API_BASE_URL+endpoint))

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
	return API_BASE_URL + "/mapping/values/" + property
}

// ========================= CODEGEN =========================
//go:generate go run gen/gen.go
