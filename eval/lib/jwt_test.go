package lib_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/test"
)

func TestJwtSignStatic(t *testing.T) {
	tests := []struct {
		name     string
		hcl      string
		jspLabel string
		claims   string
		want     string
	}{
		{
			"HS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.Hf-ZtIlsxR2bDOdAEMaDHaOBmfVWTQi9U68yV4YHW9w",
		},
		{
			"HS256 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key_file = "testdata/secret.txt"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.Hf-ZtIlsxR2bDOdAEMaDHaOBmfVWTQi9U68yV4YHW9w",
		},
		{
			"HS384 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS384"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.fYh9VOfe9jv926lwyjNMHr98zqesELs-2v0feqqDByor7rlnHHUdWdZXkTcPRaDa",
		},
		{
			"HS384 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS384"
					key_file = "testdata/secret.txt"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.fYh9VOfe9jv926lwyjNMHr98zqesELs-2v0feqqDByor7rlnHHUdWdZXkTcPRaDa",
		},
		{
			"HS512 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS512"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.LCzGZMYxwLAra2tHNDFBCSKVzMdZeZGxhgGuVr0e9mHDXMqpyOiCBWN2JB-9aDUNPHobwxEWofPY8M9icL3YXQ",
		},
		{
			"HS512 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS512"
					key_file = "testdata/secret.txt"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.LCzGZMYxwLAra2tHNDFBCSKVzMdZeZGxhgGuVr0e9mHDXMqpyOiCBWN2JB-9aDUNPHobwxEWofPY8M9icL3YXQ",
		},
		{
			"RS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS256"
					key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.oSS8rC1KonyZ-JZTZhkqZb5bN0_2Lrbl4J33nLgWroc5vDvmLW0KnX0RQfXy0OjX4uBBYTThActqqqM6vidaXmBfsQ77uB9narWeAptRnKqEPlY-onTHDmTMCz7vQ9wbLT7Aa6MYlhRqKX5adpPPbwBUuhm2I-yMF80nSmFpSk0",
		},
		{
			"RS256 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS256"
					key_file = "testdata/rsa_priv.pem"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.oSS8rC1KonyZ-JZTZhkqZb5bN0_2Lrbl4J33nLgWroc5vDvmLW0KnX0RQfXy0OjX4uBBYTThActqqqM6vidaXmBfsQ77uB9narWeAptRnKqEPlY-onTHDmTMCz7vQ9wbLT7Aa6MYlhRqKX5adpPPbwBUuhm2I-yMF80nSmFpSk0",
		},
		{
			"RS384 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS384"
					key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.GaKMl5KxXXTlOUgfZy-Cs81jvVhp-2TjEZIg58qnjIbHH7P0YtIr8p4ikTHQhie7oXs5iwzQPdPqwJSUYHlia3t118mv1ie0IWjqmXhOcWsEODQYHfshIfKaqfIZaF7WFBZXfNdXX4g-8_aUrNPevZ_6vVhHq2844cjaKH-XGl4",
		},
		{
			"RS384 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS384"
					key_file = "testdata/rsa_priv.pem"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzM4NCIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.GaKMl5KxXXTlOUgfZy-Cs81jvVhp-2TjEZIg58qnjIbHH7P0YtIr8p4ikTHQhie7oXs5iwzQPdPqwJSUYHlia3t118mv1ie0IWjqmXhOcWsEODQYHfshIfKaqfIZaF7WFBZXfNdXX4g-8_aUrNPevZ_6vVhHq2844cjaKH-XGl4",
		},
		{
			"RS384 / key",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS512"
					key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.flU1adXUEaZuqkFwhcgJ8U3OXYOTC6RQCWw9rb7nkTNzt7XrU13EPtlxH5_7lpAvyBn4iyOCiJd19y1paupyeYbHEgUGsVXa4Iu1jQ8I7C41ejLNybdg7XpRzf3zt6tMC3W9Bp0TYRqrykTiQ0W4pg0sGJCV-e30dSDgkfuS_TM",
		},
		{
			"RS384 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS512"
					key_file = "testdata/rsa_priv.pem"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.flU1adXUEaZuqkFwhcgJ8U3OXYOTC6RQCWw9rb7nkTNzt7XrU13EPtlxH5_7lpAvyBn4iyOCiJd19y1paupyeYbHEgUGsVXa4Iu1jQ8I7C41ejLNybdg7XpRzf3zt6tMC3W9Bp0TYRqrykTiQ0W4pg0sGJCV-e30dSDgkfuS_TM",
		},
		{
			"jwt / HS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					signing_ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.Hf-ZtIlsxR2bDOdAEMaDHaOBmfVWTQi9U68yV4YHW9w",
		},
		{
			"jwt / HS256 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "HS256"
					key_file = "testdata/secret.txt"
					signing_ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.Hf-ZtIlsxR2bDOdAEMaDHaOBmfVWTQi9U68yV4YHW9w",
		},
		{
			"jwt / RS256 / key",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "RS256"
					signing_key = <<EOF
-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDGSd+sSTss2uOuVJKpumpFAamlt1CWLMTAZNAabF71Ur0P6u83
3RhAIjXDSA/QeVitzvqvCZpNtbOJVegaREqLMJqvFOUkFdLNRP3f9XjYFFvubo09
tcjX6oGEREKDqLG2MfZ2Z8LVzuJc6SwZMgVFk/63rdAOci3W9u3zOSGj4QIDAQAB
AoGAMzI1rw0FW1J0wLkTWQFJmOGSBLhs9Sk/75DX7kqWxe6D5A07kIfkUALFMNN1
SdVa4R10uibXkULdxRLKJ6YEPLGAN3UmdbnBGxZ+fHAKY3PxM5lL9d7ET08A0u/8
6vB+GZ8w0eqsp4EFzmXI5LS63cRo9GA5iliGpKWtd2IUA2UCQQDnZHJTHW21vrXv
GqXoPxOoQAflxvnHYDgNQcRJxlEokFmSK405n7G2//NrsSnXYmUsA/wdh9YsAYZ3
4xy6hKE3AkEA22Aw58FnypcRAKBTqEWHv957szAmz9R6mLJqG7283YWXL0VGDOuR
qdC4QjMrix3O8WbJxGNaVCrvYKVtKEfPpwJAGGWw4C6UKLuI90L6BzjPW8gUjRej
sm/kuREcHyM3320I5K6O32qFFGR8R/iQDtOjEzcAWCTAYjdu9CkQGGJvlQJAHpCR
X8jfmCdiFA9CeKBvYHk0DOw5jB1Tk3DQPds6tDaHsOta7jPoEJvnADo25+QYUCP9
GqKpFC8DORjzU3hl4wJACEzmqzAco2M4mVc+PxPX0b3LHaREyXURd+faFXUecxSF
BShcGHZl9nzWDtEZzgdX7cbG5nRUo1+whzBQdYoQmg==
-----END RSA PRIVATE KEY-----
					EOF
					signing_ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.oSS8rC1KonyZ-JZTZhkqZb5bN0_2Lrbl4J33nLgWroc5vDvmLW0KnX0RQfXy0OjX4uBBYTThActqqqM6vidaXmBfsQ77uB9narWeAptRnKqEPlY-onTHDmTMCz7vQ9wbLT7Aa6MYlhRqKX5adpPPbwBUuhm2I-yMF80nSmFpSk0",
		},
		{
			"jwt / RS256 / key_file",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "RS256"
					signing_key_file = "testdata/rsa_priv.pem"
					signing_ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub":"12345"}`,
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJUSEVfQVVESUVOQ0UiLCJpc3MiOiJ0aGVfaXNzdWVyIiwic3ViIjoiMTIzNDUifQ.oSS8rC1KonyZ-JZTZhkqZb5bN0_2Lrbl4J33nLgWroc5vDvmLW0KnX0RQfXy0OjX4uBBYTThActqqqM6vidaXmBfsQ77uB9narWeAptRnKqEPlY-onTHDmTMCz7vQ9wbLT7Aa6MYlhRqKX5adpPPbwBUuhm2I-yMF80nSmFpSk0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err != nil {
				t.Fatal(err)
			}
			claims, err := stdlib.JSONDecode(cty.StringVal(tt.claims))
			if err != nil {
				t.Fatal(err)
			}

			hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()

			token, err := hclContext.Functions[lib.FnJWTSign].Call([]cty.Value{cty.StringVal(tt.jspLabel), claims})
			if err != nil {
				t.Fatal(err)
			}
			if token.AsString() != tt.want {
				t.Errorf("Expected %q, got: %#v", tt.want, token.AsString())
			}
		})
	}
}

func TestJwtSignDynamic(t *testing.T) {
	tests := []struct {
		name     string
		hcl      string
		jspLabel string
		headers  map[string]interface{}
		claims   string
		wantTTL  int64
		wantMeth string
	}{
		{
			"ttl 1h",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "1h"
					claims = {
						x-method = request.method
						x-status = backend_responses.default.status
						exp = 1234567890
					}
				}
			}
			`,
			"MyToken",
			map[string]interface{}{"alg": "HS256", "typ": "JWT"},
			`{"sub": "12345"}`,
			3600,
			http.MethodPost,
		},
		{
			"ttl 60.6s",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "60.6s"
					claims = {
						x-method = request.method
						x-status = backend_responses.default.status
					}
				}
			}
			`,
			"MyToken",
			map[string]interface{}{"alg": "HS256", "typ": "JWT"},
			`{"sub": "12345"}`,
			60,
			http.MethodPost,
		},
		{
			"user-defined header",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "1h"
					headers = {
						kid = "key-id"
						foo = [request.method, backend_responses.default.status]
						alg = "none"  // overriden
						typ = "HWT"   // overriden
					}
					claims = {
						x-method = "GET"
						x-status = 200
					}
				}
			}
			`,
			"MyToken",
			map[string]interface{}{"alg": "HS256", "typ": "JWT", "kid": "key-id", "foo": []interface{}{"GET", 200}},
			`{"sub": "12345"}`,
			3600,
			http.MethodGet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := test.New(t)

			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			helper.Must(err)

			claims, err := stdlib.JSONDecode(cty.StringVal(tt.claims))
			helper.Must(err)

			req := httptest.NewRequest(tt.wantMeth, "http://1.2.3.4/", nil)
			*req = *req.WithContext(context.Background())
			beresp := &http.Response{
				Request:    req,
				StatusCode: http.StatusOK,
			}

			evalCtx := cf.Context.Value(request.ContextType).(*eval.Context).
				WithClientRequest(req).
				WithBeresps(beresp)

			now := time.Now().Unix()
			token, err := evalCtx.HCLContext().Functions[lib.FnJWTSign].Call([]cty.Value{cty.StringVal(tt.jspLabel), claims})
			helper.Must(err)

			tokenParts := strings.Split(token.AsString(), ".")
			if len(tokenParts) != 3 {
				t.Errorf("Needs 3 parts, got: %d", len(tokenParts))
			}

			joseHeader, err := base64.RawURLEncoding.DecodeString(tokenParts[0])
			helper.Must(err)

			var resultHeaders map[string]interface{}
			err = json.Unmarshal(joseHeader, &resultHeaders)
			helper.Must(err)

			if fmt.Sprint(tt.headers) != fmt.Sprint(resultHeaders) {
				t.Errorf("Headers:\n\tWant: %#v\n\tGot:  %#v", tt.headers, resultHeaders)
			}

			body, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
			helper.Must(err)

			var resultClaims map[string]interface{}
			err = json.Unmarshal(body, &resultClaims)
			helper.Must(err)

			if resultClaims["exp"] == nil {
				t.Errorf("Expected exp claim, got: %#v", body)
			}
			exp := resultClaims["exp"].(float64)
			if !fuzzyEqual(int64(exp)-now, tt.wantTTL, 1) {
				t.Errorf(string(body))
				t.Errorf("Expected %d, got: %d", tt.wantTTL, int64(exp)-now)
			}
			if resultClaims["x-method"] == nil {
				t.Errorf("Expected x-method claim, got: %#v", body)
			}
			if resultClaims["x-method"] != tt.wantMeth {
				t.Errorf("Expected: %s, got: %s", tt.wantMeth, resultClaims["x-method"])
			}

			if resultClaims["x-status"] == nil {
				t.Errorf("Expected x-status claim, got: %#v", body)
			}
			status := int64(resultClaims["x-status"].(float64))
			if status != 200 {
				t.Errorf("Expected: %d, got: %d", http.StatusOK, status)
			}
		})
	}
}

func TestJwtSignConfigError(t *testing.T) {
	tests := []struct {
		name     string
		hcl      string
		jspLabel string
		claims   string
		wantErr  string
	}{
		{
			"multiple jwt_signing_profile blocks with same label",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "1h"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"configuration error: jwt_signing_profile block with label MyToken already defined",
		},
		{
			"unsupported signature algorithm",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "invalid"
					key = "$3cRe4"
					ttl = "0"
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"configuration error: MyToken: algorithm is not supported",
		},
		{
			"missing signing key or key_file",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS256"
					ttl = "0"
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"configuration error: MyToken: jwt_signing_profile key: read error: required: configured attribute or file",
		},
		{
			"Invalid ttl value",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "invalid"
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"configuration error: MyToken: time: invalid duration \"invalid\"",
		},
		{
			"invalid PEM key format",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "RS256"
					key = "invalid"
					ttl = 0
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"configuration error: MyToken: invalid Key: Key must be PEM encoded PKCS1 or PKCS8 private key",
		},
		{
			"jwt / missing signing key or key_file",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "RS256"
					key_file = "testdata/rsa_priv.pem"
					signing_ttl = "0"
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub": "12345"}`,
			"configuration error: MySelfSignedToken: jwt signing key: read error: required: configured attribute or file",
		},
		{
			"jwt / Invalid signing_ttl value",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "HS256"
					signing_key = "$3cRe4"
					signing_ttl = "invalid"
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub": "12345"}`,
			"configuration error: MySelfSignedToken: time: invalid duration \"invalid\"",
		},
		{
			"jwt / invalid PEM key format",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "RS256"
					signing_key = "invalid"
					signing_ttl = "0"
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub": "12345"}`,
			"configuration error: MySelfSignedToken: invalid Key: Key must be PEM encoded PKCS1 or PKCS8 private key",
		},
		{
			"jwt_signing_profile and jwt with same label",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					signing_ttl = "1h"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
				jwt_signing_profile "MySelfSignedToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MySelfSignedToken",
			`{"sub":"12345"}`,
			"configuration error: jwt_signing_profile or jwt with label MySelfSignedToken already defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			_, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			if err == nil {
				t.Error("expected an error, got nothing")
				return
			}
			logErr, _ := err.(errors.GoError)
			if logErr == nil {
				t.Error("logErr should not be nil")
			} else if logErr.LogError() != tt.wantErr {
				t.Errorf("\nwant:\t%s\ngot:\t%v", tt.wantErr, logErr.LogError())
			}
		})
	}
}

func TestJwtSignError(t *testing.T) {
	tests := []struct {
		name     string
		hcl      string
		jspLabel string
		claims   string
		wantErr  string
	}{
		{
			"missing jwt_signing_profile definitions",
			`
			server "test" {
				endpoint "/" {
					response {
						body = jwt_sign()
					}
				}
			}
			definitions {
				jwt "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"MyToken",
			`{"sub": "12345"}`,
			"missing jwt_signing_profile or jwt definitions",
		},
		{
			"No profile for label",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"NoProfileForThisLabel",
			`{"sub":"12345"}`,
			"missing jwt_signing_profile or jwt for given label: NoProfileForThisLabel",
		},
		{
			"argument claims no object",
			`
			server "test" {
			}
			definitions {
				jwt_signing_profile "MyToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					ttl = "0"
				}
			}
			`,
			"MyToken",
			`"no object"`,
			"json: cannot unmarshal string into Go value of type map[string]interface {}",
		},
		{
			"jwt / No profile for label",
			`
			server "test" {
			}
			definitions {
				jwt "MySelfSignedToken" {
					signature_algorithm = "HS256"
					key = "$3cRe4"
					signing_ttl = "0"
					claims = {
					  iss = to_lower("The_Issuer")
					  aud = to_upper("The_Audience")
					}
				}
			}
			`,
			"NoProfileForThisLabel",
			`{"sub": "12345"}`,
			"missing jwt_signing_profile or jwt for given label: NoProfileForThisLabel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(st *testing.T) {
			helper := test.New(st)
			cf, err := configload.LoadBytes([]byte(tt.hcl), "couper.hcl")
			helper.Must(err)
			claims, err := stdlib.JSONDecode(cty.StringVal(tt.claims))
			helper.Must(err)

			hclContext := cf.Context.Value(request.ContextType).(*eval.Context).HCLContext()

			_, err = hclContext.Functions[lib.FnJWTSign].Call([]cty.Value{cty.StringVal(tt.jspLabel), claims})
			if err == nil {
				t.Error("expected an error, got nothing")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Want:\t%q\nGot:\t%q", tt.wantErr, err.Error())
			}
		})
	}
}
