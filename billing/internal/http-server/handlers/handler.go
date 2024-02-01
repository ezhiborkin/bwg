package handlers

import (
	"billing/internal/lib/balance"
	"billing/internal/lib/iwrequest"
	"billing/internal/lib/transaction"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	walletWorker        WalletWorker
	billingWorker       BillingWorker
	transactionProvider TransactionProvider
}

type WalletWorker interface {
	CreateWallet() (string, string, error)
	GetBalance(walletID string) ([]balance.BalanceResponse, error)
}

type BillingWorker interface {
	Invoice(walletID string, transactionType string, currency string, amount float64) (int, error)
	Withdraw(walletID string, transactionType string, currency string, amount float64) (int, error)
}

type TransactionProvider interface {
	GetTransaction(id int) (*transaction.Transaction, error)
}

func New(walletWorker WalletWorker, billingWorker BillingWorker, transactionProvider TransactionProvider) *Handler {
	return &Handler{
		walletWorker:        walletWorker,
		billingWorker:       billingWorker,
		transactionProvider: transactionProvider,
	}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	router.Use(cors.Default())

	router.GET("/balance/:id", h.getBalance)
	router.GET("/wallet", h.createWallet)
	router.POST("/invoice", h.postInvoice)
	router.POST("/withdraw", h.postWithdraw)
	router.GET("/transaction/:id", h.getTransaction)

	return router
}

func (h *Handler) getTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}
	fmt.Println(1)

	transaction, err := h.transactionProvider.GetTransaction(id)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"transaction": transaction})
}

func (h *Handler) getBalance(c *gin.Context) {
	wallet_id := c.Param("id")

	balance, err := h.walletWorker.GetBalance(wallet_id)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"balances": balance})
}

func (h *Handler) createWallet(c *gin.Context) {
	id, account_id, err := h.walletWorker.CreateWallet()
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"wallet_id": id, "account_id": account_id})
}

func (h *Handler) postInvoice(c *gin.Context) {
	var request iwrequest.IWRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction_id, err := h.billingWorker.Invoice(request.WalletID, "Invoice", request.Currency, request.Amount)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"transaction_id": transaction_id})
}

func (h *Handler) postWithdraw(c *gin.Context) {
	var request iwrequest.IWRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction_id, err := h.billingWorker.Withdraw(request.WalletID, "Withdraw", request.Currency, request.Amount)
	if err != nil {
		c.JSON(500, gin.H{"error": "internal error"})
		return
	}

	c.JSON(200, gin.H{"transaction_id": transaction_id})
}
