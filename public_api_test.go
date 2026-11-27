package branta_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/BrantaOps/branta-go/v4"
)

// Consumer-surface lock. Family helpers stay unexported; this fails if they
// regain exported names. Mirrors the other SDKs' public-API tests.
func TestFamilyHelpersAreUnexported(t *testing.T) {
	forbidden := map[string]bool{
		"PolygonEvmDestinationTypes":  true,
		"IsPolygonEvmDestinationType": true,
		"EthereumEvmDestinationTypes": true,
		"BscEvmDestinationTypes":      true,
		"TronDestinationTypes":        true,
		"TonDestinationTypes":         true,
		"SolanaDestinationTypes":      true,
		"SecretKeyURIFamilies":        true,
		"IsSecretKeyAddressType":      true,
		"EvmTokenRails":               true,
		"EVMTokenRails":               true,
		"EvmTokenRail":                true,
		"EVMTokenRail":                true,
		"EvmTokenType":                true,
		"EVMTokenType":                true,
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		name := info.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs["branta"]
	if pkg == nil {
		t.Fatal("branta package not found")
	}
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if forbidden[name.Name] {
								t.Errorf("%s must stay unexported", name.Name)
							}
						}
					case *ast.TypeSpec:
						if forbidden[s.Name.Name] {
							t.Errorf("%s must stay unexported", s.Name.Name)
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil && forbidden[d.Name.Name] {
					t.Errorf("%s must stay unexported", d.Name.Name)
				}
			}
		}
	}
}

func TestConsumerTypesAreReachable(t *testing.T) {
	if string(branta.UsdtEthereum) != "usdt_ethereum" {
		t.Fatal(branta.UsdtEthereum)
	}
	if string(branta.UsdcSolana) != "usdc_solana" {
		t.Fatal(branta.UsdcSolana)
	}
	if string(branta.UsdtTon) != "usdt_ton" {
		t.Fatal(branta.UsdtTon)
	}
	if string(branta.UsdtBase) != "usdt_base" || string(branta.UsdcArbitrum) != "usdc_arbitrum" {
		t.Fatal(branta.UsdtBase, branta.UsdcArbitrum)
	}
	if branta.Staging.URL() != "https://staging.guardrail.branta.pro" {
		t.Fatal(branta.Staging.URL())
	}

	opts := branta.NewBrantaApiOptions(branta.Localhost)
	opts.APIKey = "k"
	if opts.APIKey != "k" {
		t.Fatal(opts.APIKey)
	}

	svc := branta.NewBrantaService(branta.BrantaApiOptions{Environment: branta.Staging})
	if svc == nil {
		t.Fatal("nil service")
	}

	payment := branta.NewPaymentBuilder().
		AddDestination("bc1qexample", branta.BitcoinAddress).
		SetBrand("Shop", "https://example.com/logo.png", "https://example.com/logo-light.png").
		Build()
	if len(payment.Destinations) != 1 {
		t.Fatalf("%+v", payment)
	}
	var brand *branta.Platform = payment.Brand
	if brand == nil || brand.Name != "Shop" || payment.ChildPlatform != nil {
		t.Fatalf("%+v", payment)
	}
	//lint:ignore SA1019 the deprecated alias stays on the public surface.
	deprecated := branta.NewPaymentBuilder().SetChildPlatform("Shop", "", "").Build()
	if deprecated.Brand == nil || deprecated.Brand.Name != "Shop" {
		t.Fatalf("%+v", deprecated)
	}

	verify := branta.VerifyResult{Status: branta.VerifyUnknown, VerifyURL: "https://example/v2/verify/x"}
	if verify.Status != branta.VerifyUnknown || len(verify.MatchedDestinations) != 0 {
		t.Fatalf("%+v", verify)
	}

	err := &branta.LogoURLDomainMismatchError{Field: "platformLogoUrl"}
	if err.Error() == "" || err.Field != "platformLogoUrl" {
		t.Fatal(err)
	}
}
