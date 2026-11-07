package math

import (
	"fmt"
	"math"
	"math/big"

	"github.com/goccy/googlesqlite/internal/value"
)

// BIGNUMERIC uses a signed 256-bit integer scaled by 10^-38. Its
// minimum is one unit farther from zero than its maximum.
var minBigNumeric = new(big.Rat).SetFrac(
	new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255)),
	new(big.Int).Exp(big.NewInt(10), big.NewInt(38), nil),
)

func UNARY_MINUS(x value.Value) (value.Value, error) {
	switch x := x.(type) {
	case value.IntValue:
		if x == math.MinInt64 {
			return nil, fmt.Errorf("INT64 overflow in unary minus")
		}
		return value.IntValue(0).Sub(x)
	case value.FloatValue:
		return -x, nil
	case *value.NumericValue:
		if x.IsBigNumeric && x.Cmp(minBigNumeric) <= 0 {
			return nil, fmt.Errorf("BIGNUMERIC overflow in unary minus")
		}
		return &value.NumericValue{Rat: new(big.Rat).Neg(x.Rat), IsBigNumeric: x.IsBigNumeric}, nil
	default:
		return nil, fmt.Errorf("unsupported unary minus operand %T", x)
	}
}

func UNARY_MINUS_INT32(x value.Value) (value.Value, error) {
	if x == value.IntValue(math.MinInt32) {
		return nil, fmt.Errorf("INT32 overflow in unary minus")
	}
	return UNARY_MINUS(x)
}
