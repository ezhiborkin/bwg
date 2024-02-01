package invoiceReader

import (
	"billing/internal/lib/iwrequest"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

const (
	bootstrapServers = "kafka:9093"
	invoiceTopic     = "invoices"
	groupID          = "12"
)

type InvoiceReader struct {
	billingWorker BillingWorker
}

type BillingWorker interface {
	Invoice(walletID string, transactionType string, currency string, amount float64) (int, error)
	Withdraw(walletID string, transactionType string, currency string, amount float64) (int, error)
}

func New(billingWorker BillingWorker) *InvoiceReader {
	return &InvoiceReader{
		billingWorker: billingWorker,
	}
}

func (r *InvoiceReader) Read() {
	const op = "invoiceReader.Read"

	invoiceReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{bootstrapServers},
		CommitInterval: 0,
		GroupID:        "10",
		Topic:          invoiceTopic,
		Partition:      0,
		MaxBytes:       10e6,
	})
	defer invoiceReader.Close()

	for {
		select {
		case <-context.Background().Done():
			fmt.Println("Context canceled. Exiting...")
			return
		default:
			// Read a message from Kafka
			message, err := invoiceReader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			fmt.Println(message)

			var value iwrequest.IWRequest

			// Deserialize the JSON message into the struct
			err = json.Unmarshal(message.Value, &value)
			if err != nil {
				log.Printf("Error decoding message value: %v", err)
				return
			}

			// Process the received message
			_, err = r.billingWorker.Invoice(value.WalletID, "Invoice", value.Currency, value.Amount)
			if err != nil {
				fmt.Errorf("%s: %w", op, err)
				continue
			}
		}
	}
}
