package withdrawReader

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
	withdrawTopic    = "withdraws"
	groupID          = "12"
)

type WithdrawReader struct {
	billingWorker BillingWorker
}

type BillingWorker interface {
	Invoice(walletID string, transactionType string, currency string, amount float64) (int, error)
	Withdraw(walletID string, transactionType string, currency string, amount float64) (int, error)
}

func New(billingWorker BillingWorker) *WithdrawReader {
	return &WithdrawReader{
		billingWorker: billingWorker,
	}
}

func (r *WithdrawReader) Read() {
	const op = "withdrawReader.Read"

	withdrawReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{bootstrapServers},
		CommitInterval: 0,
		GroupID:        "11",
		Topic:          withdrawTopic,
		Partition:      0,
		MaxBytes:       10e6,
	})
	defer withdrawReader.Close()

	for {
		select {
		case <-context.Background().Done():
			fmt.Println("Context canceled. Exiting...")
			return
		default:
			// Read a message from Kafka
			message, err := withdrawReader.ReadMessage(context.Background())
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
			_, err = r.billingWorker.Withdraw(value.WalletID, "Withdraw", value.Currency, value.Amount)
			if err != nil {
				fmt.Errorf("%s: %w", op, err)
				continue
			}
		}
	}
}
