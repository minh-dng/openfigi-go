package openfigi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/minh-dng/openfigi-go/constants"
)

type middleware func(http.HandlerFunc) http.HandlerFunc

// === MIDDLEWAREs ===
func chain(f http.HandlerFunc, middlewares ...middleware) http.HandlerFunc {
	for _, m := range slices.Backward(middlewares) {
		f = m(f)
	}
	return f
}

func method(method string) middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			next(w, r)
		}
	}
}

func jsonContentType() middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") != "application/json" {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				return
			}
			f(w, r)
		}
	}
}

// === HELPERs ===

func jsonDecode[T any](r *http.Request) (payload T, err error) {
	err = json.NewDecoder(r.Body).Decode(&payload)
	return
}

func shouldPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() { _ = recover() }()
	f()
	t.Errorf("should have panicked")
}

// === TESTs ===

func TestMapping(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/mapping", chain(mappingHandler, method("POST"), jsonContentType()))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	SetAPIBaseUrl(ts.URL)

	map_builder := MappingItem{}.GetBuilder(constants.IDTYPE_TICKER, "IBM")
	map_builder.SetExchCode(constants.EXCHCODE_US)
	map_item, _ := map_builder.Build()
	res, err := MappingRequest{map_item}.Fetch()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 response, got %d", len(res))
	}

	if len(res[0].Data) != 1 {
		t.Errorf("Expected 1 data item, got %d", len(res[0].Data))
	}

	if res[0].Data[0].FIGI != "BBG000BLNNH6" {
		t.Errorf("Expected FIGI to be BBG000BLNNH6, got %s", res[0].Data[0].FIGI)
	}
}

func TestMappingTooManyItems(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/mapping", chain(mappingHandler, method("POST"), jsonContentType()))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	SetAPIBaseUrl(ts.URL)

	map_builder := MappingItem{}.GetBuilder(constants.IDTYPE_TICKER, "IBM")
	map_builder.SetExchCode(constants.EXCHCODE_US)
	map_item, _ := map_builder.Build()
	map_bulk := MappingRequest{
		map_item, map_item, map_item, map_item, map_item, map_item, map_item, map_item, map_item, map_item, map_item}
	_, err := map_bulk.Fetch()
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if err.Error() != strconv.Itoa(http.StatusRequestEntityTooLarge) {
		t.Errorf("Expected code %d, got %s", http.StatusRequestEntityTooLarge, err.Error())
	}
}

func TestSearch(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/search", chain(searchHandler, method("POST"), jsonContentType()))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	SetAPIBaseUrl(ts.URL)

	builder := BaseItem{}.GetBuilder()
	builder.SetExchCode(constants.EXCHCODE_AU)
	item, err := builder.Build()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	res, err := item.Search("", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(res.Data) == 0 {
		t.Fatalf("Expected data, got none")
	}

	res, err = res.Next()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(res.Data) == 0 {
		t.Fatalf("Expected data, got none")
	}

	res, err = res.Next()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(res.Data) != 0 {
		t.Fatalf("Expected no data, got %d", len(res.Data))
	}
}

func TestFilter(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	mux.HandleFunc("/filter", chain(filterHandler, method("POST"), jsonContentType()))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	SetAPIBaseUrl(ts.URL)

	builder := BaseItem{}.GetBuilder()
	builder.SetExchCode(constants.EXCHCODE_AU)
	item, err := builder.Build()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	res, err := item.Filter("", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(res.SearchResponse.Data) == 0 {
		t.Fatalf("Expected data, got none")
	}

	res, err = res.Next()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(res.SearchResponse.Data) == 0 {
		t.Fatalf("Expected data, got none")
	}

	res, err = res.Next()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(res.SearchResponse.Data) != 0 {
		t.Fatalf("Expected no data, got %d", len(res.SearchResponse.Data))
	}
	if res.Total != 1589028 {
		t.Errorf("Expected total to be 1589028, got %d", res.Total)
	}
}

func TestValidateBaseItem(t *testing.T) {
	builder := BaseItem{}.GetBuilder()

	t.Run("exchCode and micCode", func(t *testing.T) {
		builder.SetExchCode(constants.EXCHCODE_AU)
		builder.SetMicCode(constants.MICCODE_ADRK)
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad exchCode", func(t *testing.T) {
		builder.SetExchCode("zigzagzig")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad micCode", func(t *testing.T) {
		builder.SetMicCode("zigzagzig")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad currency", func(t *testing.T) {
		builder.SetCurrency("zigzagzig")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad marketSecDes", func(t *testing.T) {
		builder.SetMarketSecDes("zigzagzig")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad securityType", func(t *testing.T) {
		builder.SetSecurityType("zigzagzig")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad securityType2", func(t *testing.T) {
		builder.SetSecurityType2("zigzagzig")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad stateCode", func(t *testing.T) {
		builder.SetStateCode("zigzagzig")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad Strike 1", func(t *testing.T) {
		builder.SetStrike([2]any{nil, nil})
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad Strike 2", func(t *testing.T) {
		shouldPanic(t, func() {
			builder.SetStrike([2]any{nil, "zigzagzig"})
		})
	})
	t.Run("bad ContractSize", func(t *testing.T) {
		builder.SetContractSize([2]any{10.0, 2.0})
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("bad expiration", func(t *testing.T) {
		shouldPanic(t, func() {
			builder.SetExpiration([2]any{123.0, nil})
		})
	})
	t.Run("expiration without option", func(t *testing.T) {
		builder.SetExpiration([2]any{"2023-01-01", "2024-01-01"})
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("maturity without pool", func(t *testing.T) {
		builder.SetMaturity([2]any{"2023-01-01", "2024-01-01"})
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
}

func TestValidateMappingItem(t *testing.T) {
	t.Run("bad idType", func(t *testing.T) {
		builder := MappingItem{}.GetBuilder("zigzagzig", "IBM")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("BASE_TICKER without securityType2", func(t *testing.T) {
		builder := MappingItem{}.GetBuilder(constants.IDTYPE_BASE_TICKER, "IBM")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
	t.Run("ID_EXCH_SYMBOL without securityType2", func(t *testing.T) {
		builder := MappingItem{}.GetBuilder(constants.IDTYPE_ID_EXCH_SYMBOL, "IBM")
		if _, err := builder.Build(); err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
}

func TestSuccessfulBaseItemBuild(t *testing.T) {
	t.Run("valid 1", func(t *testing.T) {
		builder := BaseItem{}.GetBuilder()

		builder.SetExchCode(constants.EXCHCODE_AU)
		builder.SetCurrency(constants.CURRENCY_AUD)
		builder.SetMarketSecDes(constants.MARKETSECDES_Comdty)
		builder.SetSecurityType(constants.SECURITYTYPE_OPTION)
		builder.SetSecurityType2(constants.SECURITYTYPE2_Option)
		builder.SetStrike([2]any{2.0, 10.0})
		builder.SetContractSize([2]any{2.0, 10.0})
		builder.SetCoupon([2]any{2.0, 10.0})
		builder.SetExpiration([2]any{"2021-01-01", "2022-01-01"})
		if _, err := builder.Build(); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
	t.Run("valid 2", func(t *testing.T) {
		builder := BaseItem{}.GetBuilder()

		builder.SetMicCode(constants.MICCODE_BMTF)
		builder.SetSecurityType2(constants.SECURITYTYPE2_Pool)
		builder.SetMaturity([2]any{"2021-01-01", "2022-01-01"})
		builder.SetStateCode(constants.STATECODE_AC)
		if _, err := builder.Build(); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// === HANDLERs ===

// Hash from test/search.json
const nextStartHash = "QW9JSVA0QUFBQ3hDUWtjd01EQXpTRnBZTlRJPSAx.bkS2vyvHXgyqLPy2gQtIsbny1f8sAEbgSqGTnDYyJ54="

// Can only call next once, then it will return no data. This hash is from the test/search-next.json
const finalStartHash = "QW9JSVA0QUFBQ3hDUWtjd01EQXpTakF3UkRrPSAy.CnWo3ObzIZ3gHQmYNGEKY4UFKYNoqyhJIcrWD0qP+xM="

func mappingHandler(w http.ResponseWriter, r *http.Request) {
	var max_jobs int
	if r.Header.Get("X-OPENFIGI-APIKEY") == "" {
		max_jobs = 10
	} else {
		max_jobs = 100
	}

	payload, err := jsonDecode[MappingRequest](r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(payload) > max_jobs {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	res := FIGIObject{
		FIGI:                "BBG000BLNNH6",
		Name:                "INTL BUSINESS MACHINES CORP",
		Ticker:              "IBM",
		ExchangeCode:        "US",
		CompositeFIGI:       "BBG000BLNNH6",
		SecurityType:        "Common Stock",
		MarketSector:        "Equity",
		ShareClassFIGI:      "BBG001S5S399",
		SecurityType2:       "Common Stock",
		SecurityDescription: "IBM",
	}

	json_res, _ := json.Marshal([]struct {
		Data []FIGIObject `json:"data"`
	}{{Data: []FIGIObject{res}}})

	w.Header().Set("Content-Type", "application/json")
	w.Write(json_res)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := jsonDecode[searchOrFilterRequest](r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if payload.Start == finalStartHash {
		w.Write([]byte(`{"data": []}`))
		return
	}

	var jsonFilePath string
	if payload.Start == "" {
		jsonFilePath = filepath.Join("test", "search.json")
	} else if payload.Start == nextStartHash {
		jsonFilePath = filepath.Join("test", "search-next.json")
	} else {
		fmt.Println(payload.Start, nextStartHash)
		panic("Unexpected query, bad hash")
	}
	fContent, err := os.ReadFile(jsonFilePath)
	if err != nil {
		panic(err)
	}
	w.Write(fContent)
}

func filterHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := jsonDecode[searchOrFilterRequest](r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// If next is END, our custom message, then return no data
	if payload.Start == finalStartHash {
		w.Write([]byte(`{"data": [], "total": 1589028}`))
		return
	}

	// filter is search with total,
	// the next hash is also different irl but in testing doesn't matter
	var jsonFilePath string
	if payload.Start == "" {
		jsonFilePath = filepath.Join("test", "search.json")
	} else if payload.Start == nextStartHash {
		jsonFilePath = filepath.Join("test", "search-next.json")
	} else {
		panic("Unexpected query, bad hash")
	}

	f, err := os.Open(jsonFilePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var searchRes SearchResponse
	if err := json.NewDecoder(f).Decode(&searchRes); err != nil {
		panic(err)
	}
	filterRes := FilterResponse{
		Total:          1589028,
		SearchResponse: searchRes,
	}
	res, _ := json.Marshal(filterRes)
	w.Write(res)
}
