// Command branta_example is an end-to-end usage demo.
//
//	BRANTA_API_KEY=<staging-api-key> go run ./examples/branta_example
//
// Without BRANTA_API_KEY set, the read-only lookups still run (against staging),
// but the AddPayment call at the end will fail with ErrUnauthorized.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/BrantaOps/branta-go"
)

func main() {
	ctx := context.Background()
	opts := branta.BrantaClientOptions{
		BaseURL:       branta.Staging,
		DefaultAPIKey: os.Getenv("BRANTA_API_KEY"),
		Privacy:       branta.PrivacyLoose,
	}
	service := branta.NewBrantaService(opts)

	fmt.Println("Get Payments ----------------------------")
	result, err := service.GetPayments(ctx, "address1", "", nil)
	if err != nil {
		fmt.Println("Lookup failed:", err)
	} else {
		for _, payment := range result.Payments {
			fmt.Printf("Payment: %+v\n", payment)
		}
		fmt.Println("Verify URL:", result.VerifyURL)
	}

	fmt.Println("Get ZK Payments -------------------------")
	zkAddress := "pQerSFV+fievHP+guYoGJjx1CzFFrYWHAgWrLhn5473Z19M6+WMScLd1hsk808AEF/x+GpZKmNacFBf5BbQ=="
	result, err = service.GetPayments(ctx, zkAddress, "1234", nil)
	if err != nil {
		fmt.Println("ZK lookup failed:", err)
	} else {
		for _, payment := range result.Payments {
			fmt.Printf("Payment: %+v\n", payment)
		}
	}

	fmt.Println("Get Payments By QR Code ------------------")
	result, err = service.GetPaymentsByQRCode(ctx, "bitcoin:address1", nil)
	if err != nil {
		fmt.Println("QR lookup failed:", err)
	} else {
		fmt.Println("Verify URL:", result.VerifyURL)
	}

	fmt.Println("Add Payment ------------------------------")
	payment := branta.NewPaymentBuilder().
		SetDescription("Test description").
		AddMetadata("test_key", "test value").
		SetTTL(4000).
		AddDestination("address2", "").
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
