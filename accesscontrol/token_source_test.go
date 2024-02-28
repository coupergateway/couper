package accesscontrol_test

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	ac "github.com/coupergateway/couper/accesscontrol"
	"github.com/coupergateway/couper/eval/lib"
	"github.com/coupergateway/couper/internal/test"
)

func Test_NewTokenSource_error(t *testing.T) {
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	sExpr := hcl.StaticExpr(cty.StringVal("s"), hcl.Range{})

	type testCase struct {
		name   string
		bearer bool
		dpop   bool
		cookie string
		header string
		value  hcl.Expression
	}

	for _, tc := range []testCase{
		{
			"bearer + dpop", true, true, "", "", nilExpr,
		},
		{
			"bearer + cookie", true, false, "c", "", nilExpr,
		},
		{
			"bearer + header", true, false, "", "h", nilExpr,
		},
		{
			"bearer + value", true, false, "", "", sExpr,
		},
		{
			"dpop + cookie", false, true, "c", "", nilExpr,
		},
		{
			"dpop + header", false, true, "", "h", nilExpr,
		},
		{
			"dpop + value", false, true, "", "", sExpr,
		},
		{
			"cookie + header", false, false, "c", "h", nilExpr,
		},
		{
			"cookie + value", false, false, "c", "", sExpr,
		},
		{
			"header + value", false, false, "", "h", sExpr,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			ts, err := ac.NewTokenSource(tc.bearer, tc.dpop, tc.cookie, tc.header, tc.value)
			if ts != nil {
				subT.Fatal("expected nil token source")
			}
			if err == nil {
				subT.Fatal("expected error")
			}
			if err.Error() != "only one of bearer, cookie, header or token_value attributes is allowed" {
				subT.Errorf("wrong error message: %q", err.Error())
			}
		})
	}
}

func Test_TokenValue(t *testing.T) {
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	sExpr := hcl.StaticExpr(cty.StringVal("asdf"), hcl.Range{})

	type testCase struct {
		name      string
		bearer    bool
		dpop      bool
		cookie    string
		header    string
		value     hcl.Expression
		reqHeader http.Header
		expToken  string
		expErrMsg string
	}

	for _, tc := range []testCase{
		{
			"default", false, false, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer asdf"}}, "asdf", "",
		},
		{
			"default, missing authorization header", false, false, "", "", nilExpr, http.Header{}, "", "missing authorization header",
		},
		{
			"default, missing token in bearer authorization header", false, false, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer "}}, "", "token required",
		},
		{
			"default, different auth scheme", false, false, "", "", nilExpr, http.Header{"Authorization": []string{"Foo Bar"}}, "", `auth scheme "Bearer" required in authorization header`,
		},
		{
			"bearer", true, false, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer asdf"}}, "asdf", "",
		},
		{
			"bearer, missing authorization header", true, false, "", "", nilExpr, http.Header{}, "", "missing authorization header",
		},
		{
			"bearer, missing token in bearer authorization header", true, false, "", "", nilExpr, http.Header{"Authorization": []string{"Bearer "}}, "", "token required",
		},
		{
			"bearer, different auth scheme", true, false, "", "", nilExpr, http.Header{"Authorization": []string{"Foo Bar"}}, "", `auth scheme "Bearer" required in authorization header`,
		},
		{
			"dpop", false, true, "", "", nilExpr, http.Header{"Authorization": []string{"DPoP asdf"}}, "asdf", "",
		},
		{
			"dpop, missing authorization header", false, true, "", "", nilExpr, http.Header{}, "", "missing authorization header",
		},
		{
			"dpop, missing token in bearer authorization header", false, true, "", "", nilExpr, http.Header{"Authorization": []string{"DPoP "}}, "", "token required",
		},
		{
			"dpop, different auth scheme", false, true, "", "", nilExpr, http.Header{"Authorization": []string{"Foo Bar"}}, "", `auth scheme "DPoP" required in authorization header`,
		},
		{
			"cookie", false, false, "c", "", nilExpr, http.Header{"Cookie": []string{"c=asdf"}}, "asdf", "",
		},
		{
			"cookie, missing c cookie", false, false, "c", "", nilExpr, http.Header{"Cookie": []string{"foo=bar"}}, "", "token required",
		},
		{
			"header", false, false, "", "h", nilExpr, http.Header{"H": []string{"asdf"}}, "asdf", "",
		},
		{
			"header, missing h header", false, false, "", "h", nilExpr, http.Header{"Foo": []string{"bar"}}, "", "token required",
		},
		{
			"authorization header", false, false, "", "authorization", nilExpr, http.Header{"Authorization": []string{"Bearer asdf"}}, "asdf", "",
		},
		{
			"authorization header, missing authorization header", false, false, "", "authorization", nilExpr, http.Header{}, "", "missing authorization header",
		},
		{
			"authorization header, missing token in bearer authorization header", false, false, "", "authorization", nilExpr, http.Header{"Authorization": []string{"Bearer "}}, "", "token required",
		},
		{
			"authorization header, different auth scheme", false, false, "", "authorization", nilExpr, http.Header{"Authorization": []string{"Foo Bar"}}, "", `auth scheme "Bearer" required in authorization header`,
		},
		{
			"value", false, false, "", "", sExpr, http.Header{}, "asdf", "",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			ts, err := ac.NewTokenSource(tc.bearer, tc.dpop, tc.cookie, tc.header, tc.value)
			helper.Must(err)

			req, err := http.NewRequest(http.MethodGet, "/foo", nil)
			helper.Must(err)
			req.Header = tc.reqHeader

			token, err := ts.TokenValue(req)
			if err != nil {
				msg := err.Error()
				if tc.expErrMsg == "" {
					subT.Errorf("expected no error, but got %q", msg)
				} else if tc.expToken != "" {
					subT.Errorf("expected no token, but got %q", token)
				} else if msg != tc.expErrMsg {
					subT.Errorf("expected error message: %q, got %q", tc.expErrMsg, msg)
				}
			} else if token != tc.expToken {
				subT.Errorf("expected token: %q, got %q", tc.expToken, token)
			}
		})
	}
}

func Test_ValidateTokenClaims_no_dpop(t *testing.T) {
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	sExpr := hcl.StaticExpr(cty.StringVal("asdf"), hcl.Range{})

	type testCase struct {
		name   string
		bearer bool
		dpop   bool
		cookie string
		header string
		value  hcl.Expression
	}

	for _, tc := range []testCase{
		{
			"default", false, false, "", "", nilExpr,
		},
		{
			"bearer", true, false, "", "", nilExpr,
		},
		{
			"cookie", false, false, "c", "", nilExpr,
		},
		{
			"header", false, false, "", "h", nilExpr,
		},
		{
			"header", false, false, "", "", sExpr,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			ts, err := ac.NewTokenSource(tc.bearer, tc.dpop, tc.cookie, tc.header, tc.value)
			helper.Must(err)

			req, err := http.NewRequest(http.MethodGet, "/foo", nil)
			helper.Must(err)

			err = ts.ValidateTokenClaims("", nil, req)
			helper.Must(err)
		})
	}
}

func Test_ValidateTokenClaims_errors(t *testing.T) {
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})

	type testCase struct {
		name      string
		reqHeader http.Header
		expErrMsg string
	}

	for _, tc := range []testCase{
		{
			"dpop, missing DPoP header", http.Header{}, "missing DPoP request header field",
		},
		{
			"dpop, too many DPoP header fields", http.Header{"Dpop": []string{"a", "b"}}, "too many DPoP request header fields",
		},
		{
			"dpop, empty DPoP proof", http.Header{"Dpop": []string{""}}, "empty DPoP proof",
		},
		{
			"dpop, invalid DPoP proof", http.Header{"Dpop": []string{"invalid"}}, "DPoP proof parse error: token is malformed: token contains an invalid number of segments",
		},
		{
			"dpop, DPoP proof JWT HS256", http.Header{"Dpop": []string{"eyJ0eXAiOiJkcG9wK2p3dCIsImFsZyI6IkhTMjU2In0.eyJodG0iOiJHRVQiLCJodHUiOiIvZm9vIiwiaWF0IjoxNTE2MjM5MDIyLCJhdGgiOiJpbnZhbGlkIiwianRpIjoic29tZV9pZCJ9.CnWfVGUQukTlMtV95bZL0BRTP8vmHf6qjfJbOSmsgAQ"}}, "DPoP proof parse error: token signature is invalid: signing method HS256 is invalid",
		},
		{
			"dpop, missing alg header", http.Header{"Dpop": []string{"eyJ0eXAiOiJkcG9wK2p3dCJ9.eyJhdGgiOiJmb28iLCJodG0iOiJHRVQiLCJodHUiOiIvZm9vIiwiaWF0IjoxMjU3ODk0MDAwLCJqdGkiOiJzb21lX2lkIn0.rxa11aHpQYMZNXWED1cxdIoNJKMcjVzMbpa6fl9t8vwBTYHpsPSAoYHtJHmbi0qSDRTkZdqpb-xLUTyPQovtsK7dbpL8oprBjRaLIDAMKUHjt_s6zq8_lJwwBo_QYseY4l14aA8gM5p3OJQjB-VClnU-jXIcHIITFA7HUXNV-nw4eDuqE5GBmeTy07ZjfdArwysCajBLz9am3gKcYDFsrxZ1a789_a9ilpBVseU9jplMa26eDoNtk6UdkY48kDCGtnnmzigjqbPo2A_iPzS-w1b2svsQwyxEf51rrO5ViE2Dw9sWFJBrKfhLAM5T1L_4C0FIWjadnBz_y4yYTNMRdQ"}}, "DPoP proof parse error: token is unverifiable: signing method (alg) is unspecified",
		},
		// cannot test this as errors from ParseWithClaims() (here: DPoP proof too old) are treated first
		// {
		// 	"dpop, missing typ header", http.Header{"Dpop": []string{"eyJhbGciOiJSUzI1NiIsImp3ayI6eyJrdHkiOiJSU0EiLCJuIjoidTFTVTFMZlZMUEhDb3pNeEgyTW80bGdPRWVQek5tMHRSZ2VMZXpWNmZmQXQwZ3VuVlRMdzdvbkxSbnJxMF9Jelc3eVdSN1Frcm1CTDdqVEtFbjV1LXFLaGJ3S2ZCc3RJcy1iTVkyWmtwMThnblR4S0x4b1MydEZjekdrUExQZ2l6c2t1ZW1NZ2hSbmlXYW9MY3llaGtkM3FxR0VsdldfVkRMNUFhV1RnMG5MVmtqUm85ei00MFJRenVWYUU4QWtBRm14WnpvdzN4LVZKWUtkanlra0owaVQ5d0NTMERSVFh1MjY5VjI2NFZmXzNqdnJlZFppS1JrZ3dsTDl4TkF3eFhGZzB4X1hGdzAwNVVXVlJJa2RnY0tXVGpwQlAyZFB3Vlo0V1dDLTlhR1ZkLUd5bjFvMENMZWxmNHJFakdvWGJBQUVnQXFlR1V4cmNJbGJqWGZiY213IiwiZSI6IkFRQUIifX0.eyJodG0iOiJHRVQiLCJodHUiOiIvZm9vIiwiaWF0IjoxNTE2MjM5MDIyLCJhdGgiOiJpbnZhbGlkIiwianRpIjoic29tZV9pZCJ9.ru82hTBpo5Qi0mSN2fk4o7YcY0_IBX858CO4V2rOMSRC5woJsT1cY08ZA_tDAEPCh-CdEn_RzAA_lvsWKwx4baSu34mGIPWswMgjLiMpZgP2njaExBj5h8rAb3DdmA8YkhNIFotIbZhdyzGw1bD5QexMfIvMcsH8RyfDcd2fzId8YjKPdv-74Z-tQwt-cnZhmWBRr_9Auh2Ykd-f4AEvdx2UjKhH1SkrfYFQ6p6GuGq9HtT1l6Vm6it7XI0aY_qQgCTpDSqjtSG8qLU60ZufX07-Db76M13wu_mHZe26oTkdm514CYgjsrF4uuqObbYwi4Pz8ioQLHu9lVLX5aNQJg"}}, "missing DPoP proof JOSE header parameter typ",
		// },
		{
			"dpop, unsupported alg PS512", http.Header{"Dpop": []string{"eyJ0eXAiOiJkcG9wK2p3dCIsImFsZyI6IlBTNTEyIn0.eyJodG0iOiJHRVQiLCJodHUiOiIvZm9vIiwiaWF0IjoxNTE2MjM5MDIyLCJhdGgiOiJpbnZhbGlkIiwianRpIjoic29tZV9pZCJ9.FrYwq5ZOHIE0TYf__RCxEps8guRkLOIlfo2s7xWCkakXnFex1GcZy02vNV9KPYgLF8C6u2rh9kfGOniOq0OPDZv_KsweYBjd_WALIyPqwSsif-TD5y-j0Ncyk9oi-lalWOtVm_HV_3QrZzV3dA8_7_rSoJsaRRw71jcXU_JKEvN31YfoE460ggHCpiWW8P8wMDzo_xYjyISusYNtPML5Pmv_lVeRYCnxTOF5IwF8ea72XjVlyvwIofa_eUR3wlrwSRNXOGyrCpLtZ-tff_sJ0dwPfMHbC82q4x3bflO0RdlZKr-uL2HKX6yD6YSXy05COnGDq2U_gJ6NTtuuo2wyTw"}}, "DPoP proof parse error: token signature is invalid: signing method PS512 is invalid",
		},
		{
			"dpop, unsupported alg none", http.Header{"Dpop": []string{"eyJhbGciOiJub25lIiwidHlwIjoiZHBvcCtqd3QifQ.eyJhdGgiOiJLOGZ0RG1KQks0VkJlVW1ZakJPU05Bcy1odEYwT3V6WDNhQ3hweWdsOVIwIiwiaHRtIjoiR0VUIiwiaHR1IjoiL2ZvbyIsImlhdCI6MTY3ODcyMTY5OSwianRpIjoic29tZV9pZCJ9.hUB_tQO0XdSrq2x6L0ZYjbOSXp9y_Rp3z5G7JAJNgfKZ7zTfyi9r_Ad9O7OJngnYC85k6ZVhRTaPUK6yQejv7ROKDhFLIctLagGIUy5WALI2wahxPEDkF8nYNq1Lli4DzDKagf0SCrrEmfLg6m24IMaYQkDEv43h_ANOjM98aVjAkUZFKaricTk0qT8WKxrl-jCv8WbqAYkJL5Pe-WUUBezNxl7hdOYkmCyR04o9apLJCNWPzqjNzWADb6f-rKWyDD8AKJZgWmeKz7l-f5EokMrFfyQP2NKl7vRk4OEesHrAm93SY-0zDv8ytW307fvCfB0vDdekYC7EG7wEwr6eOg"}}, "DPoP proof parse error: token signature is invalid: signing method none is invalid",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			ts, err := ac.NewTokenSource(false, true, "", "", nilExpr)
			helper.Must(err)

			req, err := http.NewRequest(http.MethodGet, "/foo", nil)
			helper.Must(err)
			req.Header = tc.reqHeader

			err = ts.ValidateTokenClaims("", nil, req)
			if err != nil {
				msg := err.Error()
				if msg != tc.expErrMsg {
					subT.Errorf("expected error message: %q, got: %q", tc.expErrMsg, msg)
				}
			} else {
				subT.Errorf("expected err: %q, got no error", tc.expErrMsg)
			}
		})
	}
}

func Test_ValidateTokenClaims_RSA(t *testing.T) {
	accessToken := "the_token"
	hash := sha256.Sum256([]byte(accessToken))
	ath := base64.RawURLEncoding.EncodeToString(hash[:])
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	h := test.New(t)

	privKeyPEMBytes := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC7VJTUt9Us8cKj
MzEfYyjiWA4R4/M2bS1GB4t7NXp98C3SC6dVMvDuictGeurT8jNbvJZHtCSuYEvu
NMoSfm76oqFvAp8Gy0iz5sxjZmSnXyCdPEovGhLa0VzMaQ8s+CLOyS56YyCFGeJZ
qgtzJ6GR3eqoYSW9b9UMvkBpZODSctWSNGj3P7jRFDO5VoTwCQAWbFnOjDfH5Ulg
p2PKSQnSJP3AJLQNFNe7br1XbrhV//eO+t51mIpGSDCUv3E0DDFcWDTH9cXDTTlR
ZVEiR2BwpZOOkE/Z0/BVnhZYL71oZV34bKfWjQIt6V/isSMahdsAASACp4ZTGtwi
VuNd9tybAgMBAAECggEBAKTmjaS6tkK8BlPXClTQ2vpz/N6uxDeS35mXpqasqskV
laAidgg/sWqpjXDbXr93otIMLlWsM+X0CqMDgSXKejLS2jx4GDjI1ZTXg++0AMJ8
sJ74pWzVDOfmCEQ/7wXs3+cbnXhKriO8Z036q92Qc1+N87SI38nkGa0ABH9CN83H
mQqt4fB7UdHzuIRe/me2PGhIq5ZBzj6h3BpoPGzEP+x3l9YmK8t/1cN0pqI+dQwY
dgfGjackLu/2qH80MCF7IyQaseZUOJyKrCLtSD/Iixv/hzDEUPfOCjFDgTpzf3cw
ta8+oE4wHCo1iI1/4TlPkwmXx4qSXtmw4aQPz7IDQvECgYEA8KNThCO2gsC2I9PQ
DM/8Cw0O983WCDY+oi+7JPiNAJwv5DYBqEZB1QYdj06YD16XlC/HAZMsMku1na2T
N0driwenQQWzoev3g2S7gRDoS/FCJSI3jJ+kjgtaA7Qmzlgk1TxODN+G1H91HW7t
0l7VnL27IWyYo2qRRK3jzxqUiPUCgYEAx0oQs2reBQGMVZnApD1jeq7n4MvNLcPv
t8b/eU9iUv6Y4Mj0Suo/AU8lYZXm8ubbqAlwz2VSVunD2tOplHyMUrtCtObAfVDU
AhCndKaA9gApgfb3xw1IKbuQ1u4IF1FJl3VtumfQn//LiH1B3rXhcdyo3/vIttEk
48RakUKClU8CgYEAzV7W3COOlDDcQd935DdtKBFRAPRPAlspQUnzMi5eSHMD/ISL
DY5IiQHbIH83D4bvXq0X7qQoSBSNP7Dvv3HYuqMhf0DaegrlBuJllFVVq9qPVRnK
xt1Il2HgxOBvbhOT+9in1BzA+YJ99UzC85O0Qz06A+CmtHEy4aZ2kj5hHjECgYEA
mNS4+A8Fkss8Js1RieK2LniBxMgmYml3pfVLKGnzmng7H2+cwPLhPIzIuwytXywh
2bzbsYEfYx3EoEVgMEpPhoarQnYPukrJO4gwE2o5Te6T5mJSZGlQJQj9q4ZB2Dfz
et6INsK0oG8XVGXSpQvQh3RUYekCZQkBBFcpqWpbIEsCgYAnM3DQf3FJoSnXaMhr
VBIovic5l0xFkEHskAjFTevO86Fsz1C2aSeRKSqGFoOQ0tmJzBEs1R6KqnHInicD
TQrKhArgLXX4v3CddjfTRJkFWDbE/CkvKZNOrcf1nhaGCPspRJj2KUkj1Fhl9Cnc
dn/RsYEONbwQSjIfMPkvxF+8HQ==
-----END PRIVATE KEY-----`)
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyPEMBytes)
	h.Must(err)
	pubKey := privKey.PublicKey
	jwk := test.RSAPubKeyToJWK(pubKey)
	// to check removal of non-required members
	jwk["non_required"] = "foo"
	jkt := ac.JwkToJKT(jwk)

	jwkMissingKty := copyJWK(jwk)
	delete(jwkMissingKty, "kty")
	jwkMissingN := copyJWK(jwk)
	delete(jwkMissingN, "n")
	jwkMissingE := copyJWK(jwk)
	delete(jwkMissingE, "e")

	type testCase struct {
		name         string
		tokenClaims  map[string]interface{}
		proofHeaders map[string]interface{}
		proofClaims  jwt.MapClaims
		expErrMsg    string
	}

	for _, tc := range []testCase{
		{
			"dpop, missing jwk header", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256"}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: missing jwk JOSE header parameter or wrong type",
		},
		{
			"dpop, jwk header wrong type", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": "invalid"}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: missing jwk JOSE header parameter or wrong type",
		},
		{
			"dpop, jwk header missing kty", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwkMissingKty}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: jwk JOSE header parameter missing kty property or wrong type",
		},
		{
			"dpop, jwk header missing n", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwkMissingN}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: jwk JOSE header parameter missing n property or wrong type",
		},
		{
			"dpop, jwk header missing e", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwkMissingE}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: jwk JOSE header parameter missing e property or wrong type",
		},
		{
			"dpop, missing jti claim", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{}, "DPoP proof parse error: token has invalid claims: missing DPoP proof claim jti",
		},
		{
			"dpop, missing htm claim", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id"}, "DPoP proof parse error: token has invalid claims: missing DPoP proof claim htm",
		},
		{
			"dpop, missing htu claim", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET"}, "DPoP proof parse error: token has invalid claims: missing DPoP proof claim htu",
		},
		{
			"dpop, missing iat claim", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo"}, "DPoP proof parse error: token has invalid claims: missing DPoP proof claim iat",
		},
		{
			"dpop, missing ath claim", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix()}, "DPoP proof parse error: token has invalid claims: missing DPoP proof claim ath",
		},
		{
			"dpop, typ mismatch", nil, map[string]interface{}{"typ": "invalid_type", "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "DPoP proof typ JOSE header parameter mismatch",
		},
		{
			"dpop, htm mismatch", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "mismatch", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "DPoP proof parse error: token has invalid claims: DPoP proof htm claim mismatch",
		},
		{
			"dpop, htu mismatch", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/mismatch", "iat": time.Now().Unix(), "ath": ath}, "DPoP proof parse error: token has invalid claims: DPoP proof htu claim mismatch",
		},
		{
			"dpop, proof too old", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix() - 12, "ath": ath}, "DPoP proof parse error: token has invalid claims: DPoP proof too old",
		},
		{
			"dpop, proof too new", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix() + 12, "ath": ath}, "DPoP proof parse error: token has invalid claims: DPoP proof too new",
		},
		{
			"dpop, ath mismatch", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": "invalid"}, "DPoP proof parse error: token has invalid claims: DPoP proof ath claim mismatch",
		},
		{
			"dpop, missing cnf in access token claims", nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "missing DPoP access token cnf claim or wrong type",
		},
		{
			"dpop, cnf wrong type", map[string]interface{}{"cnf": "invalid"}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "missing DPoP access token cnf claim or wrong type",
		},
		{
			"dpop, missing cnf.jkt property", map[string]interface{}{"cnf": map[string]interface{}{}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "DPoP access token cnf claim missing jkt property or wrong type",
		},
		{
			"dpop, cnf.jkt property wrong type", map[string]interface{}{"cnf": map[string]interface{}{"jkt": 1234}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "DPoP access token cnf claim missing jkt property or wrong type",
		},
		{
			"dpop, jkt mismatch", map[string]interface{}{"cnf": map[string]interface{}{"jkt": "invalid"}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "DPoP JWK thumbprint mismatch",
		},
		{
			"dpop RS256 ok", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "",
		},
		{
			"dpop RS384 ok", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS384", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "",
		},
		{
			"dpop RS512 ok", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS512", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			ts, err := ac.NewTokenSource(false, true, "", "", nilExpr)
			helper.Must(err)

			req, err := http.NewRequest(http.MethodGet, "/foo", nil)
			helper.Must(err)
			signatureAlgorithm := tc.proofHeaders["alg"].(string)
			proof, err := lib.CreateJWT(signatureAlgorithm, privKey, tc.proofClaims, tc.proofHeaders)
			helper.Must(err)
			req.Header.Set("DPoP", proof)

			err = ts.ValidateTokenClaims(accessToken, tc.tokenClaims, req)
			if err != nil {
				msg := err.Error()
				if tc.expErrMsg == "" {
					subT.Errorf("expected no error, but got: %q", msg)
				} else if msg != tc.expErrMsg {
					subT.Errorf("expected error message: %q, got: %q", tc.expErrMsg, msg)
				}
			} else if tc.expErrMsg != "" {
				subT.Errorf("expected err: %q, got no error", tc.expErrMsg)
			}
		})
	}
}

func copyJWK(src map[string]interface{}) map[string]interface{} {
	c := make(map[string]interface{}, len(src))
	for k, v := range src {
		c[k] = v
	}
	return c
}

func Test_ValidateTokenClaims_ECDSA(t *testing.T) {
	accessToken := "the_token"
	hash := sha256.Sum256([]byte(accessToken))
	ath := base64.RawURLEncoding.EncodeToString(hash[:])
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	h := test.New(t)

	privKeyPEMBytes256 := []byte(`-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgevZzL1gdAFr88hb2
OF/2NxApJCzGCEDdfSp6VQO30hyhRANCAAQRWz+jn65BtOMvdyHKcvjBeBSDZH2r
1RTwjmYSi9R/zpBnuQ4EiMnCqfMPWiZqB4QdbAd0E7oH50VpuZ1P087G
-----END PRIVATE KEY-----`)
	privKey256, err := jwt.ParseECPrivateKeyFromPEM(privKeyPEMBytes256)
	h.Must(err)
	pubKey256 := privKey256.PublicKey
	jwk256 := ecdsaPubKeyToJWK(pubKey256)
	jkt256 := ac.JwkToJKT(jwk256)

	jwkMissingCrv := copyJWK(jwk256)
	delete(jwkMissingCrv, "crv")
	jwkMissingX := copyJWK(jwk256)
	delete(jwkMissingX, "x")
	jwkMissingY := copyJWK(jwk256)
	delete(jwkMissingY, "y")

	privKeyPEMBytes384 := []byte(`-----BEGIN PRIVATE KEY-----
MIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIGbAgEBBDCAHpFQ62QnGCEvYh/p
E9QmR1C9aLcDItRbslbmhen/h1tt8AyMhskeenT+rAyyPhGhZANiAAQLW5ZJePZz
MIPAxMtZXkEWbDF0zo9f2n4+T1h/2sh/fviblc/VTyrv10GEtIi5qiOy85Pf1RRw
8lE5IPUWpgu553SteKigiKLUPeNpbqmYZUkWGh3MLfVzLmx85ii2vMU=
-----END PRIVATE KEY-----`)
	privKey384, err := jwt.ParseECPrivateKeyFromPEM(privKeyPEMBytes384)
	h.Must(err)
	pubKey384 := privKey384.PublicKey
	jwk384 := ecdsaPubKeyToJWK(pubKey384)
	jkt384 := ac.JwkToJKT(jwk384)

	privKeyPEMBytes512 := []byte(`-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIBiyAa7aRHFDCh2qga
9sTUGINE5jHAFnmM8xWeT/uni5I4tNqhV5Xx0pDrmCV9mbroFtfEa0XVfKuMAxxf
Z6LM/yKhgYkDgYYABAGBzgdnP798FsLuWYTDDQA7c0r3BVk8NnRUSexpQUsRilPN
v3SchO0lRw9Ru86x1khnVDx+duq4BiDFcvlSAcyjLACJvjvoyTLJiA+TQFdmrear
jMiZNE25pT2yWP1NUndJxPcvVtfBW48kPOmvkY4WlqP5bAwCXwbsKrCgk6xbsp12
ew==
-----END PRIVATE KEY-----`)
	privKey512, err := jwt.ParseECPrivateKeyFromPEM(privKeyPEMBytes512)
	h.Must(err)
	pubKey512 := privKey512.PublicKey
	jwk512 := ecdsaPubKeyToJWK(pubKey512)
	jkt512 := ac.JwkToJKT(jwk512)

	type testCase struct {
		name         string
		privKey      *ecdsa.PrivateKey
		tokenClaims  map[string]interface{}
		proofHeaders map[string]interface{}
		proofClaims  jwt.MapClaims
		expErrMsg    string
	}

	for _, tc := range []testCase{
		{
			"dpop, jwk header missing crv", privKey256, nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "ES256", "jwk": jwkMissingCrv}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: jwk JOSE header parameter missing crv property or wrong type",
		},
		{
			"dpop, jwk header missing x", privKey256, nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "ES256", "jwk": jwkMissingX}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: jwk JOSE header parameter missing x property or wrong type",
		},
		{
			"dpop, jwk header missing y", privKey256, nil, map[string]interface{}{"typ": ac.DpopTyp, "alg": "ES256", "jwk": jwkMissingY}, jwt.MapClaims{}, "DPoP proof parse error: token is unverifiable: error while executing keyfunc: jwk JOSE header parameter missing y property or wrong type",
		},
		{
			"dpop ES256 ok", privKey256, map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt256}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "ES256", "jwk": jwk256}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "",
		},
		{
			"dpop ES384 ok", privKey384, map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt384}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "ES384", "jwk": jwk384}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "",
		},
		{
			"dpop ES512 ok", privKey512, map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt512}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "ES512", "jwk": jwk512}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "/foo", "iat": time.Now().Unix(), "ath": ath}, "",
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			ts, err := ac.NewTokenSource(false, true, "", "", nilExpr)
			helper.Must(err)

			req, err := http.NewRequest(http.MethodGet, "/foo", nil)
			helper.Must(err)
			signatureAlgorithm := tc.proofHeaders["alg"].(string)
			proof, err := lib.CreateJWT(signatureAlgorithm, tc.privKey, tc.proofClaims, tc.proofHeaders)
			helper.Must(err)
			req.Header.Set("DPoP", proof)

			err = ts.ValidateTokenClaims(accessToken, tc.tokenClaims, req)
			if err != nil {
				msg := err.Error()
				if tc.expErrMsg == "" {
					subT.Errorf("expected no error, but got: %q", msg)
				} else if msg != tc.expErrMsg {
					subT.Errorf("expected error message: %q, got: %q", tc.expErrMsg, msg)
				}
			} else if tc.expErrMsg != "" {
				subT.Errorf("expected err: %q, got no error", tc.expErrMsg)
			}
		})
	}
}

func ecdsaPubKeyToJWK(key ecdsa.PublicKey) map[string]interface{} {
	x := base64.RawURLEncoding.EncodeToString(key.X.Bytes())
	y := base64.RawURLEncoding.EncodeToString(key.Y.Bytes())
	params := key.Params()
	return map[string]interface{}{
		"kty": "ECDSA",
		"crv": params.Name,
		"x":   x,
		"y":   y,
	}
}

func Test_ValidateTokenClaims_htu_normalize(t *testing.T) {
	accessToken := "the_token"
	hash := sha256.Sum256([]byte(accessToken))
	ath := base64.RawURLEncoding.EncodeToString(hash[:])
	nilExpr := hcl.StaticExpr(cty.NilVal, hcl.Range{})
	h := test.New(t)

	privKeyPEMBytes := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC7VJTUt9Us8cKj
MzEfYyjiWA4R4/M2bS1GB4t7NXp98C3SC6dVMvDuictGeurT8jNbvJZHtCSuYEvu
NMoSfm76oqFvAp8Gy0iz5sxjZmSnXyCdPEovGhLa0VzMaQ8s+CLOyS56YyCFGeJZ
qgtzJ6GR3eqoYSW9b9UMvkBpZODSctWSNGj3P7jRFDO5VoTwCQAWbFnOjDfH5Ulg
p2PKSQnSJP3AJLQNFNe7br1XbrhV//eO+t51mIpGSDCUv3E0DDFcWDTH9cXDTTlR
ZVEiR2BwpZOOkE/Z0/BVnhZYL71oZV34bKfWjQIt6V/isSMahdsAASACp4ZTGtwi
VuNd9tybAgMBAAECggEBAKTmjaS6tkK8BlPXClTQ2vpz/N6uxDeS35mXpqasqskV
laAidgg/sWqpjXDbXr93otIMLlWsM+X0CqMDgSXKejLS2jx4GDjI1ZTXg++0AMJ8
sJ74pWzVDOfmCEQ/7wXs3+cbnXhKriO8Z036q92Qc1+N87SI38nkGa0ABH9CN83H
mQqt4fB7UdHzuIRe/me2PGhIq5ZBzj6h3BpoPGzEP+x3l9YmK8t/1cN0pqI+dQwY
dgfGjackLu/2qH80MCF7IyQaseZUOJyKrCLtSD/Iixv/hzDEUPfOCjFDgTpzf3cw
ta8+oE4wHCo1iI1/4TlPkwmXx4qSXtmw4aQPz7IDQvECgYEA8KNThCO2gsC2I9PQ
DM/8Cw0O983WCDY+oi+7JPiNAJwv5DYBqEZB1QYdj06YD16XlC/HAZMsMku1na2T
N0driwenQQWzoev3g2S7gRDoS/FCJSI3jJ+kjgtaA7Qmzlgk1TxODN+G1H91HW7t
0l7VnL27IWyYo2qRRK3jzxqUiPUCgYEAx0oQs2reBQGMVZnApD1jeq7n4MvNLcPv
t8b/eU9iUv6Y4Mj0Suo/AU8lYZXm8ubbqAlwz2VSVunD2tOplHyMUrtCtObAfVDU
AhCndKaA9gApgfb3xw1IKbuQ1u4IF1FJl3VtumfQn//LiH1B3rXhcdyo3/vIttEk
48RakUKClU8CgYEAzV7W3COOlDDcQd935DdtKBFRAPRPAlspQUnzMi5eSHMD/ISL
DY5IiQHbIH83D4bvXq0X7qQoSBSNP7Dvv3HYuqMhf0DaegrlBuJllFVVq9qPVRnK
xt1Il2HgxOBvbhOT+9in1BzA+YJ99UzC85O0Qz06A+CmtHEy4aZ2kj5hHjECgYEA
mNS4+A8Fkss8Js1RieK2LniBxMgmYml3pfVLKGnzmng7H2+cwPLhPIzIuwytXywh
2bzbsYEfYx3EoEVgMEpPhoarQnYPukrJO4gwE2o5Te6T5mJSZGlQJQj9q4ZB2Dfz
et6INsK0oG8XVGXSpQvQh3RUYekCZQkBBFcpqWpbIEsCgYAnM3DQf3FJoSnXaMhr
VBIovic5l0xFkEHskAjFTevO86Fsz1C2aSeRKSqGFoOQ0tmJzBEs1R6KqnHInicD
TQrKhArgLXX4v3CddjfTRJkFWDbE/CkvKZNOrcf1nhaGCPspRJj2KUkj1Fhl9Cnc
dn/RsYEONbwQSjIfMPkvxF+8HQ==
-----END PRIVATE KEY-----`)
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyPEMBytes)
	h.Must(err)
	pubKey := privKey.PublicKey
	jwk := test.RSAPubKeyToJWK(pubKey)
	jkt := ac.JwkToJKT(jwk)

	type testCase struct {
		name         string
		reqUrl       string
		tokenClaims  map[string]interface{}
		proofHeaders map[string]interface{}
		proofClaims  jwt.MapClaims
	}

	for _, tc := range []testCase{
		{
			"http default", "http://www.example.com:80/path", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "http://www.example.com/path", "iat": time.Now().Unix(), "ath": ath},
		},
		{
			"https default", "https://www.example.com:443/path", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "https://www.example.com/path", "iat": time.Now().Unix(), "ath": ath},
		},
		{
			"empty port", "https://www.example.com:/path", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "https://www.example.com/path", "iat": time.Now().Unix(), "ath": ath},
		},
		{
			"non-default port", "https://www.example.com:8080/path", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "https://www.example.com:8080/path", "iat": time.Now().Unix(), "ath": ath},
		},
		{
			"several normalizations", "hTtPs://wWw.ExAmPlE.cOm:443/./b/../%63/%7bfoo%7d?query=to_be_ignored", map[string]interface{}{"cnf": map[string]interface{}{"jkt": jkt}}, map[string]interface{}{"typ": ac.DpopTyp, "alg": "RS256", "jwk": jwk}, jwt.MapClaims{"jti": "some_id", "htm": "GET", "htu": "https://www.example.com/c/%7Bfoo%7D", "iat": time.Now().Unix(), "ath": ath},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			ts, err := ac.NewTokenSource(false, true, "", "", nilExpr)
			helper.Must(err)

			req, err := http.NewRequest(http.MethodGet, tc.reqUrl, nil)
			helper.Must(err)
			signatureAlgorithm := tc.proofHeaders["alg"].(string)
			proof, err := lib.CreateJWT(signatureAlgorithm, privKey, tc.proofClaims, tc.proofHeaders)
			helper.Must(err)
			req.Header.Set("DPoP", proof)

			err = ts.ValidateTokenClaims(accessToken, tc.tokenClaims, req)
			if err != nil {
				msg := err.Error()
				subT.Errorf("expected no error, but got: %q", msg)
			}
		})
	}
}

func Test_JwkToString(t *testing.T) {
	type testCase struct {
		name string
		jwk  map[string]interface{}
		exp  string
	}

	for _, tc := range []testCase{
		{
			"RSA", map[string]interface{}{"kty": "RSA", "n": "qoterbnwbn", "e": "tbin", "non_required": "foo"}, `{"e":"tbin","kty":"RSA","n":"qoterbnwbn"}`,
		},
		{
			"ECDSA", map[string]interface{}{"kty": "ECDSA", "crv": "etin", "x": "qoterbnwbn", "y": "tbin", "non_required": "foo"}, `{"crv":"etin","kty":"ECDSA","x":"qoterbnwbn","y":"tbin"}`,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			s := ac.JwkToString(tc.jwk)
			if s != tc.exp {
				subT.Errorf("expected: %q\ngot: %q", tc.exp, s)
			}
		})
	}
}
