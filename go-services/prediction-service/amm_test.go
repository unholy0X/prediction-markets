package main

import (
	"math"
	"testing"
)

const eps = 1e-9

// A fresh market with equal shares must price both outcomes at exactly 50%.
func TestPricesStartAtFiftyFifty(t *testing.T) {
	yes, no := lmsrPrices(0, 0, 100)
	if math.Abs(yes-0.5) > eps || math.Abs(no-0.5) > eps {
		t.Fatalf("expected 50/50, got yes=%v no=%v", yes, no)
	}
}

// The defining property of LMSR: prices always sum to 1, so they are probabilities.
func TestPricesAlwaysSumToOne(t *testing.T) {
	cases := [][2]float64{{0, 0}, {100, 0}, {500, 120}, {1, 9999}, {-50, 50}}
	for _, c := range cases {
		yes, no := lmsrPrices(c[0], c[1], 100)
		if math.Abs((yes+no)-1.0) > eps {
			t.Errorf("prices do not sum to 1 for q=%v: yes=%v no=%v sum=%v", c, yes, no, yes+no)
		}
	}
}

// Buying Yes shares must push the Yes probability up.
func TestBuyingYesRaisesYesPrice(t *testing.T) {
	yesBefore, _ := lmsrPrices(0, 0, 100)
	yesAfter, _ := lmsrPrices(50, 0, 100)
	if yesAfter <= yesBefore {
		t.Fatalf("buying Yes did not raise Yes price: before=%v after=%v", yesBefore, yesAfter)
	}
}

// Cost must be strictly increasing in the number of shares bought (no free shares,
// and bigger orders cost more — slippage).
func TestCostIsMonotonic(t *testing.T) {
	c10 := lmsrCostToBuy(0, 0, 100, OutcomeYes, 10)
	c100 := lmsrCostToBuy(0, 0, 100, OutcomeYes, 100)
	if c10 <= 0 {
		t.Fatalf("buying shares should cost something, got %v", c10)
	}
	if c100 <= c10 {
		t.Fatalf("larger order should cost more: c10=%v c100=%v", c10, c100)
	}
}

// Marginal price paid (cost/shares) must lie between the pre- and post-trade
// instantaneous prices — a sanity bound on slippage.
func TestAveragePriceWithinBounds(t *testing.T) {
	shares := 100.0
	pBefore, _ := lmsrPrices(0, 0, 100)
	pAfter, _ := lmsrPrices(shares, 0, 100)
	avg := lmsrCostToBuy(0, 0, 100, OutcomeYes, shares) / shares
	if avg < pBefore-eps || avg > pAfter+eps {
		t.Fatalf("avg price %v not in [%v, %v]", avg, pBefore, pAfter)
	}
}

// Large share counts must not overflow (log-sum-exp stability check).
func TestNoOverflowAtScale(t *testing.T) {
	yes, no := lmsrPrices(1e6, 5e5, 100)
	if math.IsNaN(yes) || math.IsInf(yes, 0) || math.Abs((yes+no)-1.0) > eps {
		t.Fatalf("overflow at scale: yes=%v no=%v", yes, no)
	}
}
