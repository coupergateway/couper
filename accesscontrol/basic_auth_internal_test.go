package accesscontrol

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/crypto/argon2"
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

	if !validateAccessData("john", pass, data, nil) {
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

	if !validateAccessData("jane", pass, data, nil) {
		t.Error("Unexpected validation failure")
	}

	if validateAccessData("foo", "bar", data, nil) {
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

	if !validateAccessData("jock", pass, data, nil) {
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

	if !validateAccessData("jack", pass, data, nil) {
		t.Error("Unexpected validation failure for argon2id")
	}

	if validateAccessData("jack", "wrong-pass", data, nil) {
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

	if !validateAccessData("jim", pass, data, nil) {
		t.Error("Unexpected validation failure for argon2i")
	}

	if validateAccessData("jim", "wrong-pass", data, nil) {
		t.Error("Unexpected validation success for argon2i with wrong password")
	}
}

// Test_Argon2Verifier_CollapsesConcurrentAttempts drives the real Validate path:
// a retry storm with one credential must cost fewer derivations than requests,
// and every caller must still get the correct result.
func Test_Argon2Verifier_CollapsesConcurrentAttempts(t *testing.T) {
	const callers = 8

	ba, err := NewBasicAuth("ba", "", "", "testdata/htpasswd", nil)
	if err != nil {
		t.Fatal(err)
	}

	var derivations, entered int64
	release := make(chan struct{})

	ba.verifier.derive = func(plainPass string, p pwd) bool {
		atomic.AddInt64(&derivations, 1)
		<-release // hold the flight open so the other callers join it

		return runArgon2(plainPass, p)
	}

	results := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "http://couper.io/", nil)
			req.SetBasicAuth("jack", "my-pass")

			atomic.AddInt64(&entered, 1)
			results[i] = ba.Validate(req)
		}(i)
	}

	for atomic.LoadInt64(&entered) < callers {
		runtime.Gosched()
	}
	close(release)
	wg.Wait()

	for i, rErr := range results {
		if rErr != nil {
			t.Errorf("caller %d: unexpected validation error: %v", i, rErr)
		}
	}

	if got := atomic.LoadInt64(&derivations); got < 1 || got >= callers {
		t.Errorf("derivations: want at least 1 and fewer than %d, got %d", callers, got)
	}
}

// Test_Argon2Verifier_NilReceiver keeps the direct derivation path working for
// the tests that build htData without a BasicAuth instance.
func Test_Argon2Verifier_NilReceiver(t *testing.T) {
	var verifier *argon2Verifier

	salt := []byte("0123456789abcdef")
	p := pwd{
		pwdOrig:       argon2.IDKey([]byte("my-pass"), salt, 1, 8, 1, 32),
		pwdType:       pwdTypeArgon2id,
		argon2Time:    1,
		argon2Memory:  8,
		argon2Threads: 1,
		argon2KeyLen:  32,
		argon2Salt:    salt,
	}

	if !verifier.validateArgon2("my-pass", p) {
		t.Error("Unexpected validation failure")
	}
	if verifier.validateArgon2("wrong-pass", p) {
		t.Error("Unexpected validation success")
	}
}
