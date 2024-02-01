package service

import (
	"billing/internal/lib/balance"
	"billing/internal/lib/transaction"
	"fmt"
	"log/slog"
)

type Service struct {
	log                 *slog.Logger
	walletCreator       WalletCreator
	balanceProvider     BalanceProvider
	billingProvider     BillingProvider
	transactionProvider TransactionProvider
}

func New(
	log *slog.Logger,
	walletCreator WalletCreator,
	balanceProvider BalanceProvider,
	billingProvider BillingProvider,
	transactionProvider TransactionProvider,
) *Service {
	return &Service{
		log:                 log,
		walletCreator:       walletCreator,
		balanceProvider:     balanceProvider,
		billingProvider:     billingProvider,
		transactionProvider: transactionProvider,
	}
}

type WalletCreator interface {
	CreateWallet() (string, string, error)
}

type BalanceProvider interface {
	GetBalance(walletID string) ([]balance.BalanceResponse, error)
}

type BillingProvider interface {
	PerformWithdrawTransaction(walletID string, transactionType string, currency string, amount float64) (int, error)
	PerformInvoiceTransaction(walletID string, transactionType string, currency string, amount float64) (int, error)
}

type TransactionProvider interface {
	GetTransaction(id int) (*transaction.Transaction, error)
}

func (s *Service) GetTransaction(id int) (*transaction.Transaction, error) {
	const op = "service.GetTransaction"

	transact, err := s.transactionProvider.GetTransaction(id)
	if err != nil {
		return &transaction.Transaction{}, fmt.Errorf("%s: %w", op, err)
	}

	return transact, nil
}

func (s *Service) CreateWallet() (string, string, error) {
	const op = "service.CreateWallet"

	id, account_id, err := s.walletCreator.CreateWallet()
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	return id, account_id, nil
}

func (s *Service) GetBalance(walletID string) ([]balance.BalanceResponse, error) {
	const op = "service.GetBalance"

	balances, err := s.balanceProvider.GetBalance(walletID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return balances, nil
}

func (s *Service) Withdraw(walletID string, transactionType string, currency string, amount float64) (int, error) {
	const op = "service.Withdraw"

	id, err := s.billingProvider.PerformWithdrawTransaction(walletID, transactionType, currency, amount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Service) Invoice(walletID string, transactionType string, currency string, amount float64) (int, error) {
	const op = "service.Invoice"

	id, err := s.billingProvider.PerformInvoiceTransaction(walletID, transactionType, currency, amount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}
