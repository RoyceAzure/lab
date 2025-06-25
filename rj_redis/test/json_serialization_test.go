package test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// 測試各種基本類型的結構體
type BasicTypes struct {
	IntVal     int     `json:"int_val"`
	Int8Val    int8    `json:"int8_val"`
	Int16Val   int16   `json:"int16_val"`
	Int32Val   int32   `json:"int32_val"`
	Int64Val   int64   `json:"int64_val"`
	UintVal    uint    `json:"uint_val"`
	Uint8Val   uint8   `json:"uint8_val"`
	Uint16Val  uint16  `json:"uint16_val"`
	Uint32Val  uint32  `json:"uint32_val"`
	Uint64Val  uint64  `json:"uint64_val"`
	Float32Val float32 `json:"float32_val"`
	Float64Val float64 `json:"float64_val"`
	BoolVal    bool    `json:"bool_val"`
	StringVal  string  `json:"string_val"`
}

// 測試 decimal.Decimal 類型
type DecimalTypes struct {
	DecimalVal decimal.Decimal `json:"decimal_val"`
}

// 測試指針類型
type PointerTypes struct {
	IntPtr     *int             `json:"int_ptr"`
	FloatPtr   *float64         `json:"float_ptr"`
	BoolPtr    *bool            `json:"bool_ptr"`
	StringPtr  *string          `json:"string_ptr"`
	DecimalPtr *decimal.Decimal `json:"decimal_ptr"`
}

func TestBasicTypesSerialization(t *testing.T) {
	// 創建測試數據
	original := BasicTypes{
		IntVal:     -123,
		Int8Val:    -8,
		Int16Val:   -16,
		Int32Val:   -32,
		Int64Val:   -64,
		UintVal:    123,
		Uint8Val:   8,
		Uint16Val:  16,
		Uint32Val:  32,
		Uint64Val:  64,
		Float32Val: 3.14159,
		Float64Val: math.Pi,
		BoolVal:    true,
		StringVal:  "test string",
	}

	// 序列化
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded BasicTypes
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	// 驗證所有字段
	require.Equal(t, original, decoded)
}

func TestDecimalSerialization(t *testing.T) {
	// 創建測試數據
	original := DecimalTypes{
		DecimalVal: decimal.NewFromFloat(3.14159),
	}

	// 序列化
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded DecimalTypes
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	// 驗證
	require.True(t, original.DecimalVal.Equal(decoded.DecimalVal))
}

func TestPointerTypesSerialization(t *testing.T) {
	// 創建測試數據
	intVal := 123
	floatVal := 3.14159
	boolVal := true
	stringVal := "test"
	decimalVal := decimal.NewFromFloat(3.14159)

	original := PointerTypes{
		IntPtr:     &intVal,
		FloatPtr:   &floatVal,
		BoolPtr:    &boolVal,
		StringPtr:  &stringVal,
		DecimalPtr: &decimalVal,
	}

	// 序列化
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded PointerTypes
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	// 驗證
	require.Equal(t, *original.IntPtr, *decoded.IntPtr)
	require.Equal(t, *original.FloatPtr, *decoded.FloatPtr)
	require.Equal(t, *original.BoolPtr, *decoded.BoolPtr)
	require.Equal(t, *original.StringPtr, *decoded.StringPtr)
	require.True(t, original.DecimalPtr.Equal(*decoded.DecimalPtr))
}

func TestNullPointerSerialization(t *testing.T) {
	// 創建包含空指針的測試數據
	original := PointerTypes{}

	// 序列化
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded PointerTypes
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	// 驗證所有指針都為 nil
	require.Nil(t, decoded.IntPtr)
	require.Nil(t, decoded.FloatPtr)
	require.Nil(t, decoded.BoolPtr)
	require.Nil(t, decoded.StringPtr)
	require.Nil(t, decoded.DecimalPtr)
}

func TestMapSerialization(t *testing.T) {
	// 創建包含各種類型的 map
	original := map[string]interface{}{
		"int":     123,
		"float":   3.14159,
		"bool":    true,
		"string":  "test",
		"decimal": decimal.NewFromFloat(3.14159),
	}

	// 序列化
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded map[string]interface{}
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	// 驗證基本類型
	require.Equal(t, float64(123), decoded["int"])
	require.Equal(t, 3.14159, decoded["float"])
	require.Equal(t, true, decoded["bool"])
	require.Equal(t, "test", decoded["string"])

	// decimal.Decimal 會被轉換為字符串
	require.IsType(t, "", decoded["decimal"])
}

func TestSliceSerialization(t *testing.T) {
	// 創建包含各種類型的切片
	original := []interface{}{
		123,
		3.14159,
		true,
		"test",
		decimal.NewFromFloat(3.14159),
	}

	// 序列化
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// 反序列化
	var decoded []interface{}
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	// 驗證
	require.Equal(t, float64(123), decoded[0])
	require.Equal(t, 3.14159, decoded[1])
	require.Equal(t, true, decoded[2])
	require.Equal(t, "test", decoded[3])
	require.IsType(t, "", decoded[4])
}
