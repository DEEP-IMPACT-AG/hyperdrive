package common

import "testing"

func TestCalc(t *testing.T) {
	tests := map[string]int64{
		"x":         1,    // simple sequence
		"x-1":       0,    // sequence starting with 0
		"8000 + x":  8001, // sequence starting with 8001
		"2 * (x-1)": 0,    // even sequence starting with 0
	}
	for expr, expected := range tests {
		res, err := Eval(expr, 1);
		if err != nil {
			t.Error(err)
		}
		if res != expected {
			t.Errorf("Expecting %d got %d", expected, res)
		}
	}

}
