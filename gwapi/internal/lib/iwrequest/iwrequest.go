package iwrequest

type IWRequest struct {
	WalletID string  `json:"wallet_id" binding:"required"`
	Currency string  `json:"currency" binding:"required"`
	Amount   float64 `json:"amount" binding:"required"`
}
