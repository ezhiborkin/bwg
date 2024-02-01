package handler

import (
	br "gwapi/internal/lib/balance"
	"gwapi/internal/lib/iwrequest"
	ts "gwapi/internal/lib/transaction"
	wl "gwapi/internal/lib/wallet"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	billingWorker BillingWorker
}

type BillingWorker interface {
	Invoice(walletID string, currency string, amount float64) error
	Withdraw(walletID string, currency string, amount float64) error
	Balance(wallet_id string) (br.BalanceResponse, error)
	Transaction(id string) (ts.TransactionResponse, error)
	Wallet() (wl.WalletResponse, error)
}

func New(billingWorker BillingWorker) *Handler {
	return &Handler{
		billingWorker: billingWorker,
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	router.Use(cors.Default())

	router.POST("/wallet", h.createWallet)
	router.GET("/balance/:id", h.getBalance)
	router.POST("/invoice", h.createInvoice)
	router.POST("/withdraw", h.createWithdraw)
	router.GET("/transaction/:id", h.getTransaction)

	// router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}

// @Summary Get transaction info
// @Description Get transaction info based on id
// @Tags APIs
// @Produce json
// @Param filename query string true "File name to filter errors"
// @Success 200 {object} transaction.TransactionResponse
// @Failure 500
// @Router /transaction/:id [get]
func (h *Handler) getTransaction(c *gin.Context) {
	const op = "handler.Transaction"

	id := c.Param("id")

	result, err := h.billingWorker.Transaction(id)
	if err != nil {
		c.JSON(500, gin.H{op: "failed to get transaction"})
	}

	c.JSON(200, result)
}

func (h *Handler) createInvoice(c *gin.Context) {
	const op = "handler.createInvoice"

	var request iwrequest.IWRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{op: err.Error()})
		return
	}

	err := h.billingWorker.Invoice(request.WalletID, request.Currency, request.Amount)
	if err != nil {
		c.JSON(500, gin.H{op: "internal error"})
		return
	}

	c.JSON(200, gin.H{"invoice": request})
}

func (h *Handler) createWithdraw(c *gin.Context) {
	const op = "handler.createWithdraw"

	var request iwrequest.IWRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.billingWorker.Withdraw(request.WalletID, request.Currency, request.Amount)
	if err != nil {
		c.JSON(500, gin.H{op: "internal error"})
		return
	}

	c.JSON(200, gin.H{"withdraw": request})
}

func (h *Handler) createWallet(c *gin.Context) {
	const op = "handler.createWallet"

	result, err := h.billingWorker.Wallet()
	if err != nil {
		c.JSON(500, gin.H{op: "failed to create wallet"})
	}

	c.JSON(200, gin.H{"wallet": result})
}

func (h *Handler) getBalance(c *gin.Context) {
	const op = "handler.getBalance"

	wallet_id := c.Param("id")

	result, err := h.billingWorker.Balance(wallet_id)
	if err != nil {
		c.JSON(500, gin.H{op: "failed to get balance"})
	}

	c.JSON(200, result)
}
