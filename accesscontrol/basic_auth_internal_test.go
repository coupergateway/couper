package accesscontrol

import (
	"context"
	"encoding/base64"
	goerrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/coupergateway/couper/errors"
)

func Test_Apr1MD5(t *testing.T) {
	var exp, res string

	exp = "$apr1$NPXZYWba$/ebZ19mhDyKnsuM/cRaxq0"
	res = string(apr1MD5("xxx", "NPXZYWba", "$apr1$"))
	if exp != res {
		t.Errorf("Got unexpected password: '%s', want '%s'", res, exp)
	}

	exp = "$apr1$4z8NMYQV$TexsH1pVjUbkarHcVB2q/0"
	res = string(apr1MD5("s", "4z8NMYQV", "$apr1$"))
	if exp != res {
		t.Errorf("Got unexpected password: '%s', want '%s'", res, exp)
	}
}

func Test_ValidateAccessData(t *testing.T) {
	var data htData = make(htData)
	var pass string = "my-pass"

	// $ htpasswd -bm .htpasswd john my-pass
	// john:$apr1$9zGWAElT$VQXJ4anNzh6qGRCfHdrYF0
	data["john"] = pwd{
		pwdOrig:   []byte("$apr1$9zGWAElT$VQXJ4anNzh6qGRCfHdrYF0"),
		pwdPrefix: "$apr1$",
		pwdSalt:   "9zGWAElT",
		pwdType:   pwdTypeApr1,
	}

	if !mustValidate(t, "john", pass, data) {
		t.Error("Unexpected validation failure")
	}

	// $ htpasswd -bB .htpasswd jane my-pass
	//jane:$2y$05$/uonQYUtwmVv.6AF38IhGeqlvIMPIM5jevzIQ.8RBENUgkCqbJYTm
	data["jane"] = pwd{
		pwdOrig:   []byte("$2y$05$/uonQYUtwmVv.6AF38IhGeqlvIMPIM5jevzIQ.8RBENUgkCqbJYTm"),
		pwdPrefix: "$2y$",
		pwdSalt:   "05",
		pwdType:   pwdTypeBcrypt,
	}

	if !mustValidate(t, "jane", pass, data) {
		t.Error("Unexpected validation failure")
	}

	if mustValidate(t, "foo", "bar", data) {
		t.Error("Unexpected validation success")
	}

	// php -r 'echo crypt("my-pass")."\n";'
	// $1$drjdAXLW$P9cBlaFpBbi2xszjrmUV11
	data["jock"] = pwd{
		pwdOrig:   []byte("$1$drjdAXLW$P9cBlaFpBbi2xszjrmUV11"),
		pwdPrefix: "$1$",
		pwdSalt:   "drjdAXLW",
		pwdType:   pwdTypeMD5,
	}

	if !mustValidate(t, "jock", pass, data) {
		t.Error("Unexpected validation failure")
	}

	// argon2id: generate a known hash for "my-pass"
	argon2Salt, err := base64.RawStdEncoding.DecodeString("wATvbKx1Yd01DEZk1zpXww")
	if err != nil {
		t.Fatalf("failed to decode argon2 salt: %v", err)
	}
	argon2Hash := argon2.IDKey([]byte(pass), argon2Salt, 3, 65536, 4, 32)

	data["jack"] = pwd{
		pwdOrig:       argon2Hash,
		pwdPrefix:     "$argon2id$",
		pwdType:       pwdTypeArgon2id,
		argon2Time:    3,
		argon2Memory:  65536,
		argon2Threads: 4,
		argon2KeyLen:  32,
		argon2Salt:    argon2Salt,
	}

	if !mustValidate(t, "jack", pass, data) {
		t.Error("Unexpected validation failure for argon2id")
	}

	if mustValidate(t, "jack", "wrong-pass", data) {
		t.Error("Unexpected validation success for argon2id with wrong password")
	}

	// argon2i: generate a known hash for "my-pass"
	argon2iHash := argon2.Key([]byte(pass), argon2Salt, 3, 65536, 4, 32)

	data["jim"] = pwd{
		pwdOrig:       argon2iHash,
		pwdPrefix:     "$argon2i$",
		pwdType:       pwdTypeArgon2i,
		argon2Time:    3,
		argon2Memory:  65536,
		argon2Threads: 4,
		argon2KeyLen:  32,
		argon2Salt:    argon2Salt,
	}

	if !mustValidate(t, "jim", pass, data) {
		t.Error("Unexpected validation failure for argon2i")
	}

	if mustValidate(t, "jim", "wrong-pass", data) {
		t.Error("Unexpected validation success for argon2i with wrong password")
	}
}

// mustValidate keeps the table style of the validation tests. An error means
// that Couper did not run the derivation, which no table case provokes.
func mustValidate(t *testing.T, user, pass string, data htData) bool {
	t.Helper()

	valid, err := validateAccessData(context.Background(), user, pass, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return valid
}

// Test_Argon2_BoundsConcurrentDerivations covers the limit that closes the
// amplification. Each caller sends a different password, thus each attempt runs
// its own derivation. Without the limit the peak memory would follow the number
// of requests.
func Test_Argon2_BoundsConcurrentDerivations(t *testing.T) {
	const (
		bound   = 2
		callers = 8
	)

	originalSem, originalDerive := argon2Sem, argon2Derive
	argon2Sem = make(chan struct{}, bound)
	defer func() { argon2Sem, argon2Derive = originalSem, originalDerive }()

	ba, err := NewBasicAuth("ba", "", "", "testdata/htpasswd", nil)
	if err != nil {
		t.Fatal(err)
	}

	var inFlight, peak int64
	argon2Derive = func(plainPass string, p pwd) bool {
		current := atomic.AddInt64(&inFlight, 1)
		for {
			seen := atomic.LoadInt64(&peak)
			if current <= seen || atomic.CompareAndSwapInt64(&peak, seen, current) {
				break
			}
		}

		time.Sleep(20 * time.Millisecond) // hold the slot, so an overlap becomes visible
		atomic.AddInt64(&inFlight, -1)

		return runArgon2(plainPass, p)
	}

	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "http://couper.io/", nil)
			req.SetBasicAuth("jack", fmt.Sprintf("wrong-pass-%d", i)) // unique, so each attempt derives

			_ = ba.Validate(req)
		}(i)
	}
	wg.Wait()

	got := atomic.LoadInt64(&peak)
	if got > bound {
		t.Errorf("concurrent derivations: want at most %d, got %d", bound, got)
	}
	if got < 2 {
		t.Errorf("concurrent derivations: want an overlap to prove the limit applies, got %d", got)
	}
}

// Test_Argon2_AbandonsCanceledRequest shows that a request which already ended
// runs no derivation. Its log must also not report a credential mismatch.
func Test_Argon2_AbandonsCanceledRequest(t *testing.T) {
	ba, err := NewBasicAuth("ba", "", "", "testdata/htpasswd", nil)
	if err != nil {
		t.Fatal(err)
	}

	original := argon2Derive
	defer func() { argon2Derive = original }()

	var derivations int64
	argon2Derive = func(_ string, _ pwd) bool {
		atomic.AddInt64(&derivations, 1)

		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "http://couper.io/", nil).WithContext(ctx)
	req.SetBasicAuth("jack", "my-pass")

	vErr := ba.Validate(req)
	if !goerrors.Is(vErr, context.Canceled) {
		t.Errorf("Expected the context error, got: %v", vErr)
	}
	if !strings.Contains(vErr.(*errors.Error).LogError(), "argon2 verification abandoned") {
		t.Errorf("Expected the log to name the abandoned derivation, got: %q", vErr.(*errors.Error).LogError())
	}
	if got := atomic.LoadInt64(&derivations); got != 0 {
		t.Errorf("derivations: want none for a request that ended, got %d", got)
	}
}
