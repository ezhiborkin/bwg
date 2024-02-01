package service

import (
	"context"
	"encoding/json"
	"fmt"
	"gwapi/internal/config"
	br "gwapi/internal/lib/balance"
	"gwapi/internal/lib/iwrequest"
	ts "gwapi/internal/lib/transaction"
	wl "gwapi/internal/lib/wallet"
	"io"
	"log/slog"
	"net/http"

	"github.com/segmentio/kafka-go"
)

type Service struct {
	log *slog.Logger
}

func New(log *slog.Logger, cfg config.Config) *Service {
	return &Service{
		log: log,
	}
}

func (s *Service) Wallet() (wl.WalletResponse, error) {
	const op = "service.Wallet"

	var result wl.WalletResponse

	url := "http://localhost:8082/wallet"

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return wl.WalletResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return wl.WalletResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return wl.WalletResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return wl.WalletResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}

func (s *Service) Balance(wallet_id string) (br.BalanceResponse, error) {
	const op = "service.Balance"

	url := "http://localhost:8082/balance/" + wallet_id

	var result br.BalanceResponse

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return br.BalanceResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return br.BalanceResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return br.BalanceResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return br.BalanceResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}

func (s *Service) Invoice(walletID string, currency string, amount float64) error {
	const op = "service.Invoice"

	value := iwrequest.IWRequest{
		WalletID: walletID,
		Currency: currency,
		Amount:   amount,
	}

	invoiceWriter := &kafka.Writer{
		Addr:     kafka.TCP("kafka:9093"),
		Topic:    "invoices",
		Balancer: &kafka.LeastBytes{},
	}

	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal struct to JSON: %w", op, err)
	}

	message := kafka.Message{
		Key:   []byte("invoice-key"),
		Value: []byte(jsonValue),
	}

	err = invoiceWriter.WriteMessages(context.Background(), message)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) Withdraw(walletID string, currency string, amount float64) error {
	const op = "service.Withdraw"

	value := iwrequest.IWRequest{
		WalletID: walletID,
		Currency: currency,
		Amount:   amount,
	}

	withdrawWriter := &kafka.Writer{
		Addr:     kafka.TCP("kafka:9093"),
		Topic:    "withdraws",
		Balancer: &kafka.LeastBytes{},
	}

	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%s: failed to marshal struct to JSON: %w", op, err)
	}

	message := kafka.Message{
		Key:   []byte("invoice-key"),
		Value: []byte(jsonValue),
	}

	err = withdrawWriter.WriteMessages(context.Background(), message)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) Transaction(id string) (ts.TransactionResponse, error) {
	const op = "service.Transaction"

	var result ts.TransactionResponse

	url := "http://localhost:8082/transaction/" + id

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ts.TransactionResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ts.TransactionResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ts.TransactionResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return ts.TransactionResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}
