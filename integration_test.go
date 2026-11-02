package branta_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/BrantaOps/branta-go"
)

func skipIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("BRANTA_RUN_INTEGRATION") != "1" {
		t.Skip("Live lookups skipped unless BRANTA_RUN_INTEGRATION=1")
	}
}

func service() *branta.BrantaService {
	opts := branta.NewBrantaApiOptions(branta.Staging)
	opts.APIKey = os.Getenv("BRANTA_API_KEY")
	client := branta.NewBrantaClient(opts)
	client.SetIntegrationMetricsToken(os.Getenv("BRANTA_INTEGRATION_METRICS_TOKEN"))
	return branta.NewBrantaServiceWithDeps(opts, client, branta.AESEncryptionService{}, branta.GuidSecretGenerator{})
}

func TestStagingExampleQRs(t *testing.T) {
	skipIntegration(t)
	ctx := context.Background()
	svc := service()

	match := []string{
		"bitcoin:bc1qgw3dzmhnyvcswc9r0v0z0ajtp8ulm4nuyeahwr?branta_id=6SpYCe23r79x0t8rNvmjWqhrg8hen4xz%2FYSXhNJ9HLSYrTRPnm%2BSwAuf6NOKobnwE%2B3MQBNkyTnCtIWpKKX2E9ZUWVedig%3D%3D&branta_secret=05eb145d-b5cb-452a-9256-0bdfd9a15ffa",
		"bitcoin:tb1qshra2mlujc9wfscvn4d8aqpyqgxzl55853xg7s?branta_id=fa05KT1L6zIAYx2sHcImF3R5RilwMAXAvx%2F5u2YVBIdvnp30drV7bszFzvpdoA%2FeluN0PXbsxcsD38twzw4%2BL8MFoHxkGQ%3D%3D&branta_secret=0621110e-1ad6-483d-80b6-78697c9f6896",
		"lightning:lnbc10u1p4w23tzpp50uc6g4020tt3ffp5xfpp83k0lcv2spytpymuupqrnsj0eth8jqgsdqhgfexzmn5vys8xarpva5kuecxq97zvuqsp5tj2a5fd84lhdddzfqz2lwmjqg7n9gp28yy5vtqd78muyzdt5tmsq9qypsqqcqpf6yv66fkcq609xzk0fcqx82hd7fsjhwrzj8xmq743v5lrtssrj95r94hyeumqeejgqvtxqy7ks7ec0uc5lc5zxg240xtwgzjtwkmh3aqqm8gyny",
		"ark1qztwuhx7vpytv4dcfeqjsepx3kxstnarelmry2smrfdgt2dv87fas7gtqkk",
		"sp1qqgste7k9hx0qftg6qmwlkqtwuy6cycyavzmzj85c6qdfhjdpdjtdgqjuexzk6murw56suy3e0rd2cgqvycxttddwsvgxe2usfpxumr70xc9pkqwv",
		"bitcoin:bc1q6745z6cy3u0k9nprurh3x804c4r7u3u8vxca2n?lightning=lnbc10u1p4w23tzpp5s6jc8gvun27un7g7q9kmh2vak78r83766heesd9fepa5s56hrpsqdqhgfexzmn5vys8xarpva5kuecxq97zvuqsp5gt4k9wwdgf85g066a05zrnlkh3jc2ng3mh9mw6yw7z03l4kmqzys9qypsqqcqpf7pk6xjfksy8sqhv24es2qgtp0e88wvt2gdv2zceealhjwh0memhzktgnfhdvzc9a6kt74w5jf5c8v9t20k38snawrgngr0sjw9a58fqqn9lwtq&branta_id=NxkrlFgIvGg6f8YZmFAf8qTUza8nXfn%2Bb9H3W7czFAo5Lp4cSpgRSmJgx7P9vaOYu7lkP6XXKvTshQfkXxxRrxJ%2BE7zdpA%3D%3D&branta_secret=6acf2637-511a-4b89-ba2c-f5914d7e669c",
	}
	for _, qr := range match {
		result, err := svc.Verify(ctx, qr, nil)
		if err != nil || result.Status != branta.VerifyMatch || result.VerifyURL == "" || len(result.Payments) == 0 {
			t.Fatalf("match %s: %+v %v", qr[:24], result, err)
		}
	}

	unknown, err := svc.Verify(ctx, "bitcoin:bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4", nil)
	if err != nil || unknown.Status != branta.VerifyUnknown || len(unknown.Payments) != 0 {
		t.Fatalf("unknown %+v %v", unknown, err)
	}

	tamperedQR := "bitcoin:bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4?branta_id=6SpYCe23r79x0t8rNvmjWqhrg8hen4xz%2FYSXhNJ9HLSYrTRPnm%2BSwAuf6NOKobnwE%2B3MQBNkyTnCtIWpKKX2E9ZUWVedig%3D%3D&branta_secret=05eb145d-b5cb-452a-9256-0bdfd9a15ffa"
	tampered, err := svc.Verify(ctx, tamperedQR, nil)
	if err != nil || tampered.Status != branta.VerifyTampered || len(tampered.Payments) != 0 || strings.Contains(tampered.VerifyURL, "#") {
		t.Fatalf("tampered %+v %v", tampered, err)
	}
}
