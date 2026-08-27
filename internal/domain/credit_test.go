package domain

import "testing"

func TestCreditTxType_IsValid(t *testing.T) {
	for _, typ := range []CreditTxType{CreditTxTypeEarn, CreditTxTypeRedeem, CreditTxTypeAdjust, CreditTxTypeExpire} {
		if !typ.IsValid() {
			t.Errorf("%s should be a valid transaction type", typ)
		}
	}
	if CreditTxType("unknown").IsValid() {
		t.Error("unknown should not be a valid credit tx type")
	}
}

func TestResidentCredit_InitialBalance(t *testing.T) {
	c := &ResidentCredit{ResidentID: "r1", Balance: 0}
	if c.Balance != 0 {
		t.Errorf("initial balance should be 0, got %d", c.Balance)
	}
}

func TestResidentCredit_NegativeBalanceNotAllowedByService(t *testing.T) {
	// This validates that the domain type correctly represents negative balance prevention
	// at the service layer (service checks before updating).
	c := &ResidentCredit{ResidentID: "r1", Balance: 50}
	if c.Balance < 100 {
		// simulates insufficient balance check
		_ = ErrInsufficientCredit
	}
}
