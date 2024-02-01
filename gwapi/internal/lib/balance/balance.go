package balance

type BalanceResponse struct {
	Balance []BalanceEntry `json:"balances"`
}

type BalanceEntry struct {
	WalletID     string  `json:"wallet_id"`
	Currency     string  `json:"currency"`
	Amount       float64 `json:"amount"`
	FrozenAmount float64 `json:"frozen_amount"`
}
