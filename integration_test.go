package branta_test

import (
	"context"
	"os"
	"testing"

	"github.com/BrantaOps/branta-go"
)

func skipIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("BRANTA_SKIP_INTEGRATION") != "" {
		t.Skip("BRANTA_SKIP_INTEGRATION is set")
	}
}

func service(base branta.BrantaServerBaseURL, privacy branta.PrivacyMode) *branta.BrantaService {
	return branta.NewBrantaService(branta.BrantaClientOptions{
		BaseURL: base,
		Privacy: privacy,
	})
}

const notFound = "bitcoin:bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"

const zkOnChain = "bitcoin:bc1q6745z6cy3u0k9nprurh3x804c4r7u3u8vxca2n" +
	"?branta_id=z15b5EsbP5LHJrFco38%2BFp%2BHVaiopAY676NCKek8e1Q%2B4a370TyYhvloS8uLCUHfJ4CzeI%2FbOFmFDGp" +
	"AQszB0gu1pJ1HOQ%3D%3D&branta_secret=c6e9eb30-6258-4432-9847-bdcc4fd4b0db"

func TestProductionLooseOnChainReturnsPayment(t *testing.T) {
	skipIntegration(t)
	const onChain = "bitcoin:bc1qu3k6geqdjncaarsu2vq56tt8php5vsug9kasmq"
	result, err := service(branta.Production, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), onChain, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionLooseLightningReturnsPayment(t *testing.T) {
	skipIntegration(t)
	const lightning = "lightning:lnbc17760n1p4r4tqupp5yuapqmxldkc8smuwa6t8shkdg9gezulu0vc7htepfsvweph8kqfsdphgfexzmn5vysyge" +
		"tkv4kx7ur9wgsyc6t8dp6xu6twvusy27rpd4cxcegcqzzsxq97zvuqsp53564rg6w4xjqy7jamcfqxyy83a0j8nzfs0wpevs3" +
		"7t5ln49q6hrs9qxpqysgq47hpqmv34g25le8sceq9jdvul2nz7ucyu0vucv56nlfe40x7n3jsu8duxjrn6tgvdspt872crk9ze" +
		"atafznm9c57m039z7wyx6g3njsqkchkdh"
	result, err := service(branta.Production, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), lightning, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionLooseZkOnChainReturnsPayment(t *testing.T) {
	skipIntegration(t)
	result, err := service(branta.Production, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), zkOnChain, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionLooseZkLightningReturnsPayment(t *testing.T) {
	skipIntegration(t)
	const zkLightning = "lightning:lnbc17760n1p4r4flypp5k56kq3v2935rl3glkqu9vngfueud2zj87hjcff3t0kn0yrge0pfqdzjgfexzmn5vysz6gz" +
		"yv4mx2mr0wpjhygzvd9nksarwd9hxwgz6v4ex7gztdehhwmr9v3nk2gz90psk6urvv5cqzzsxq97zvuqsp5hut3t0l0s5mvp9yr" +
		"06v4253kqtf452z6c65s6g9sga445hc03v6s9qxpqysgqqm430zkk9uymjgvllr3aha88hc6q59etxasfqswn8r8pfm3dstlpp46" +
		"azv906xtcj3wzprxup5fxn65a5wymt7zzq9sw9qdzx8rgdhcpk80nrg"
	result, err := service(branta.Production, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), zkLightning, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionLooseNotFoundReturnsEmpty(t *testing.T) {
	skipIntegration(t)
	result, err := service(branta.Production, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), notFound, nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionStrictOnChainPlainTextReturnsEmpty(t *testing.T) {
	skipIntegration(t)
	const onChain = "bitcoin:bc1qu3k6geqdjncaarsu2vq56tt8php5vsug9kasmq"
	result, err := service(branta.Production, branta.PrivacyStrict).GetPaymentsByQRCode(context.Background(), onChain, nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionStrictLightningPlainTextReturnsEmpty(t *testing.T) {
	skipIntegration(t)
	const lightning = "lightning:lnbc17760n1p4r4tqupp5yuapqmxldkc8smuwa6t8shkdg9gezulu0vc7htepfsvweph8kqfsdphgfexzmn5vysyge" +
		"tkv4kx7ur9wgsyc6t8dp6xu6twvusy27rpd4cxcegcqzzsxq97zvuqsp53564rg6w4xjqy7jamcfqxyy83a0j8nzfs0wpevs3" +
		"7t5ln49q6hrs9qxpqysgq47hpqmv34g25le8sceq9jdvul2nz7ucyu0vucv56nlfe40x7n3jsu8duxjrn6tgvdspt872crk9ze" +
		"atafznm9c57m039z7wyx6g3njsqkchkdh"
	result, err := service(branta.Production, branta.PrivacyStrict).GetPaymentsByQRCode(context.Background(), lightning, nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionStrictZkOnChainReturnsPayment(t *testing.T) {
	skipIntegration(t)
	result, err := service(branta.Production, branta.PrivacyStrict).GetPaymentsByQRCode(context.Background(), zkOnChain, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProductionStrictZkLightningReturnsPayment(t *testing.T) {
	skipIntegration(t)
	const zkLightning = "lightning:lnbc17760n1p4r4flypp5k56kq3v2935rl3glkqu9vngfueud2zj87hjcff3t0kn0yrge0pfqdzjgfexzmn5vysz6gz" +
		"yv4mx2mr0wpjhygzvd9nksarwd9hxwgz6v4ex7gztdehhwmr9v3nk2gz90psk6urvv5cqzzsxq97zvuqsp5hut3t0l0s5mvp9yr" +
		"06v4253kqtf452z6c65s6g9sga445hc03v6s9qxpqysgqqm430zkk9uymjgvllr3aha88hc6q59etxasfqswn8r8pfm3dstlpp46" +
		"azv906xtcj3wzprxup5fxn65a5wymt7zzq9sw9qdzx8rgdhcpk80nrg"
	result, err := service(branta.Production, branta.PrivacyStrict).GetPaymentsByQRCode(context.Background(), zkLightning, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestStagingLooseOnChainReturnsPayment(t *testing.T) {
	skipIntegration(t)
	const onChain = "bitcoin:bc1qgw3dzmhnyvcswc9r0v0z0ajtp8ulm4nuyeahwr"
	result, err := service(branta.Staging, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), onChain, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestStagingLooseZkOnChainReturnsPayment(t *testing.T) {
	skipIntegration(t)
	result, err := service(branta.Staging, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), zkOnChain, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestStagingLooseNotFoundReturnsEmpty(t *testing.T) {
	skipIntegration(t)
	result, err := service(branta.Staging, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), notFound, nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestStagingLooseLightningReturnsPayment(t *testing.T) {
	skipIntegration(t)
	const lightning = "lightning:lnbc25830n1p4quq9ppp5zszvpgxtu6uwyur6sf7rayc0meqprqlkv30xjzclh6nzm7gavd8sdzh2d6xzemfdenjqsnjv" +
		"9h8gcfq95sygetkv4kx7ur9wgsyc6t8dp6xu6twvusy27rpd4cxcefq9pfhgct8d9hxw2gcqzzsxqzursp5fcfx5st7x8rgxra42" +
		"j47hskmzkcz96mx84xcnvs9lpsmjyzqhw2q9qxpqysgq06lxdc93jjpuqsal9unlfct6wuv0v53yxa8kksl85g3qdw7qks7z9jkq3" +
		"9c6wgzar72luwd38sfj0klyqv0zgns4rq7nafnd8qeuudcqql7at4"
	result, err := service(branta.Staging, branta.PrivacyLoose).GetPaymentsByQRCode(context.Background(), lightning, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestStagingStrictOnChainPlainTextReturnsEmpty(t *testing.T) {
	skipIntegration(t)
	const onChain = "bitcoin:bc1qgw3dzmhnyvcswc9r0v0z0ajtp8ulm4nuyeahwr"
	result, err := service(branta.Staging, branta.PrivacyStrict).GetPaymentsByQRCode(context.Background(), onChain, nil)
	if err != nil || len(result.Payments) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestStagingStrictZkOnChainReturnsPayment(t *testing.T) {
	skipIntegration(t)
	result, err := service(branta.Staging, branta.PrivacyStrict).GetPaymentsByQRCode(context.Background(), zkOnChain, nil)
	if err != nil || len(result.Payments) == 0 {
		t.Fatalf("%+v %v", result, err)
	}
}
