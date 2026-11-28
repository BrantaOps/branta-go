// Command branta_example is an end-to-end usage demo.
//
//	BRANTA_API_KEY=<staging-api-key> go run ./examples/branta_example
//
// Without BRANTA_API_KEY set, verify still runs. AddPayment fails with ErrUnauthorized.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/BrantaOps/branta-go/v4"
)

func main() {
	ctx := context.Background()
	opts := branta.BrantaApiOptions{
		Environment: branta.Staging,
		APIKey:      os.Getenv("BRANTA_API_KEY"),
	}
	service := branta.NewBrantaService(opts)

	fmt.Println("Verify ----------------------------------")
	result, err := service.Verify(ctx, "bitcoin:bc1qexample", nil)
	if err != nil {
		fmt.Println("Verify failed:", err)
		return
	}
	fmt.Println("Status:", result.Status)
	fmt.Println("Verify URL:", result.VerifyURL)

	fmt.Println("Add Payment ------------------------------")
	payment := branta.NewPaymentBuilder().
		SetDescription("Test description").
		AddMetadata("test_key", "test value").
		SetTTL(4000).
		AddDestination("bc1qexampleaddress", branta.BitcoinAddress).
		Build()

	added, err := service.AddPayment(ctx, payment, nil)
	if err != nil {
		fmt.Println("Add payment failed:", err, "(needs BRANTA_API_KEY to succeed)")
		return
	}
	fmt.Printf("Payment: %+v\n", added.Payment)
	fmt.Println("Secret:", added.Secret)
	fmt.Println("Verify URL:", added.VerifyURL)
}
