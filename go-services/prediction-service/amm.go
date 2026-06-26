package main

import "math"

// LMSR (Logarithmic Market Scoring Rule) automated market maker.
//
// Unlike a constant-product AMM, LMSR is purpose-built for prediction markets:
//   - Outcome prices ALWAYS sum to 1.0, so each price reads directly as a probability.
//   - The market maker's worst-case loss is bounded by b*ln(n) (n = number of outcomes).
//
// Cost function:   C(q) = b * ln( e^(qYes/b) + e^(qNo/b) )
// Instant price:   p_i  = e^(q_i/b) / ( e^(qYes/b) + e^(qNo/b) )
// Trade cost:      cost = C(q_after) - C(q_before)
//
// b is the "liquidity parameter": larger b => deeper market, prices move less per trade.

// lmsrCost evaluates the LMSR cost function using the log-sum-exp trick so that
// large share counts don't overflow math.Exp.
func lmsrCost(qYes, qNo, b float64) float64 {
	m := math.Max(qYes, qNo)
	return m + b*math.Log(math.Exp((qYes-m)/b)+math.Exp((qNo-m)/b))
}

// lmsrPrices returns the instantaneous probabilities for Yes and No.
// By construction yes + no == 1.0.
func lmsrPrices(qYes, qNo, b float64) (yes, no float64) {
	m := math.Max(qYes, qNo)
	eYes := math.Exp((qYes - m) / b)
	eNo := math.Exp((qNo - m) / b)
	sum := eYes + eNo
	return eYes / sum, eNo / sum
}

// lmsrCostToBuy returns the cost of acquiring `shares` of `outcome` ("Yes"/"No")
// given the current outstanding quantities. A negative `shares` value represents
// a sell and yields a negative (refund) cost.
func lmsrCostToBuy(qYes, qNo, b float64, outcome string, shares float64) float64 {
	before := lmsrCost(qYes, qNo, b)
	var after float64
	if outcome == OutcomeYes {
		after = lmsrCost(qYes+shares, qNo, b)
	} else {
		after = lmsrCost(qYes, qNo+shares, b)
	}
	return after - before
}
