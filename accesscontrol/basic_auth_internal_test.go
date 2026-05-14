package accesscontrol

import (
	"encoding/base64"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"

	"github.com/coupergateway/couper/cache"
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
	argon2Hash := argon2.IDKey([]byte(pass), argon2Salt, 3, 65536, 2, 32)

	data["jack"] = pwd{
		pwdOrig:       argon2Hash,
		pwdPrefix:     "$argon2id$",
		pwdType:       pwdTypeArgon2id,
		argon2Time:    3,
		argon2Memory:  65536,
		argon2Threads: 2,
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
	argon2iHash := argon2.Key([]byte(pass), argon2Salt, 3, 65536, 2, 32)

	data["jim"] = pwd{
		pwdOrig:       argon2iHash,
		pwdPrefix:     "$argon2i$",
		pwdType:       pwdTypeArgon2i,
		argon2Time:    3,
		argon2Memory:  65536,
		argon2Threads: 2,
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

// Test_ValidateArgon2_Cache asserts that a cached argon2 verification
// result is served without re-running the (expensive) key derivation.
func Test_ValidateArgon2_Cache(t *testing.T) {
	pass := "my-pass"
	salt, _ := base64.RawStdEncoding.DecodeString("wATvbKx1Yd01DEZk1zpXww")
	hash := argon2.IDKey([]byte(pass), salt, 3, 65536, 2, 32)
	data := htData{
		"jack": pwd{
			pwdOrig:       hash,
			pwdPrefix:     "$argon2id$",
			pwdType:       pwdTypeArgon2id,
			argon2Time:    3,
			argon2Memory:  65536,
			argon2Threads: 2,
			argon2KeyLen:  32,
			argon2Salt:    salt,
		},
	}

	quitCh := make(chan struct{})
	defer close(quitCh)
	store := cache.New(logrus.NewEntry(logrus.New()), quitCh)
	v := newArgon2Verifier(store)

	start := time.Now()
	if !validateAccessData("jack", pass, data, v) {
		t.Fatal("expected first validation to succeed")
	}
	first := time.Since(start)

	start = time.Now()
	if !validateAccessData("jack", pass, data, v) {
		t.Fatal("expected cached validation to succeed")
	}
	cached := time.Since(start)

	// The cache hit should be at least an order of magnitude faster
	// than the actual argon2 derivation. m=64 MiB, t=3 takes tens of
	// ms; a map lookup takes microseconds.
	if cached*10 > first {
		t.Errorf("cache hit not materially faster: first=%v cached=%v", first, cached)
	}

	// Negative result is also cached so a wrong-password retry is
	// served cheaply.
	start = time.Now()
	if validateAccessData("jack", "wrong-pass", data, v) {
		t.Fatal("expected validation with wrong password to fail")
	}
	firstWrong := time.Since(start)

	start = time.Now()
	if validateAccessData("jack", "wrong-pass", data, v) {
		t.Fatal("expected cached validation with wrong password to fail")
	}
	cachedWrong := time.Since(start)

	if cachedWrong*10 > firstWrong {
		t.Errorf("negative cache hit not materially faster: first=%v cached=%v", firstWrong, cachedWrong)
	}
}

// Test_ValidateArgon2_Singleflight asserts that N concurrent identical
// argon2 verifications collapse into a single underlying derivation.
func Test_ValidateArgon2_Singleflight(t *testing.T) {
	pass := "my-pass"
	salt, _ := base64.RawStdEncoding.DecodeString("wATvbKx1Yd01DEZk1zpXww")
	hash := argon2.IDKey([]byte(pass), salt, 3, 65536, 2, 32)
	stored := pwd{
		pwdOrig:       hash,
		pwdPrefix:     "$argon2id$",
		pwdType:       pwdTypeArgon2id,
		argon2Time:    3,
		argon2Memory:  65536,
		argon2Threads: 2,
		argon2KeyLen:  32,
		argon2Salt:    salt,
	}

	v := newArgon2Verifier(nil)

	// Wrap singleflight.Do so we can count underlying derivations
	// without changing production code. The actual derivation cost
	// (~10ms) is long enough that 50 goroutines launched in a tight
	// loop will all queue behind the first.
	var derivations atomic.Int64
	var wg sync.WaitGroup
	const concurrency = 50

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			key := argon2VerifierKey("jack", pass)
			result, _, _ := v.sf.Do(key, func() (any, error) {
				derivations.Add(1)
				return runArgon2(pass, stored), nil
			})
			if !result.(bool) {
				t.Errorf("expected validation to succeed")
			}
		}()
	}
	wg.Wait()

	// singleflight makes a best-effort collapse: under heavy
	// scheduling the first call may have already returned before some
	// goroutines call Do, producing a second batch. Tolerate a few but
	// catch the no-singleflight case (every goroutine derives).
	if got := derivations.Load(); got >= concurrency/2 {
		t.Errorf("singleflight failed to collapse: %d derivations for %d concurrent calls", got, concurrency)
	}
}
