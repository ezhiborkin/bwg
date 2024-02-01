package transaction

import "time"

type Transaction struct {
	WalletID    string    `json:"wallet_id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Amount      float64   `json:"amount"`
	DateCreated time.Time `json:"date_created"`
}
