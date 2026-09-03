package accesscontrol

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// argon2Sem limits the argon2 derivations that run at the same time. The limit
// applies to the process, and thus to all basic_auth blocks together. The peak
// memory then follows this limit, and not the number of requests. The limit is
// the number of cores, because one derivation keeps one core busy.
var argon2Sem = make(chan struct{}, max(1, runtime.GOMAXPROCS(0)))

// argon2Derive lets the test count the derivations that run at the same time.
var argon2Derive = runArgon2

const (
	pwdPrefixApr1     = "$apr1$"
	pwdPrefixBcrypt2a = "$2a$"
	pwdPrefixBcrypt2b = "$2b$"
	pwdPrefixBcrypt2x = "$2x$"
	pwdPrefixBcrypt2y = "$2y$"
	pwdPrefixMD5      = "$1$"
	pwdPrefixArgon2id = "$argon2id$"
	pwdPrefixArgon2i  = "$argon2i$"
)

const (
	pwdTypeUnknown = iota
	pwdTypeApr1
	pwdTypeBcrypt
	pwdTypeMD5
	pwdTypeArgon2id
	pwdTypeArgon2i
)

const (
	aprCharacters    = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	aprMd5DigestSize = 16
	aprMuddleRounds  = 1000
)

// These are the recommended maxima for the argon2 parameters in an htpasswd
// entry. They are 2x the highest values of the OWASP Password Storage Cheat
// Sheet (m=46 MiB, t=5, p=1). An entry above them still loads, but Couper warns
// at startup. Each request pays the cost of these parameters. A large value
// therefore lets a caller use much memory of the gateway.
const (
	argon2MaxMemory  uint32 = 94208 // KiB, 2x OWASP 46 MiB
	argon2MaxTime    uint32 = 10    // 2x OWASP 5
	argon2MaxThreads uint8  = 2     // 2x OWASP 1
)

var pwdPrefixes = map[string]int{
	pwdPrefixApr1:     pwdTypeApr1,
	pwdPrefixBcrypt2a: pwdTypeBcrypt,
	pwdPrefixBcrypt2b: pwdTypeBcrypt,
	pwdPrefixBcrypt2x: pwdTypeBcrypt,
	pwdPrefixBcrypt2y: pwdTypeBcrypt,
	pwdPrefixMD5:      pwdTypeMD5,
	pwdPrefixArgon2id: pwdTypeArgon2id,
	pwdPrefixArgon2i:  pwdTypeArgon2i,
}

type htData map[string]pwd

type pwd struct {
	pwdOrig       []byte
	pwdPrefix     string
	pwdSalt       string
	pwdType       int
	argon2Time    uint32
	argon2Memory  uint32
	argon2Threads uint8
	argon2KeyLen  uint32
	argon2Salt    []byte
}

func getPwdType(pass string) int {
	for p, t := range pwdPrefixes {
		if strings.HasPrefix(pass, p) {
			return t
		}
	}

	return pwdTypeUnknown
}

// validateAccessData reports whether the credentials match an htpasswd entry.
// An error is not a mismatch. It shows that Couper did not run the argon2
// derivation, because the request ended before a slot became free.
func validateAccessData(ctx context.Context, plainUser, plainPass string, data htData) (bool, error) {
	for user, pass := range data {
		if user == plainUser {
			switch pass.pwdType {
			case pwdTypeApr1:
				fallthrough
			case pwdTypeMD5:
				if subtle.ConstantTimeCompare(apr1MD5(plainPass, pass.pwdSalt, pass.pwdPrefix), pass.pwdOrig) == 1 {
					return true, nil
				}
			case pwdTypeBcrypt:
				if err := bcrypt.CompareHashAndPassword(pass.pwdOrig, []byte(plainPass)); err == nil {
					return true, nil
				}
			case pwdTypeArgon2id, pwdTypeArgon2i:
				valid, err := validateArgon2(ctx, plainPass, pass)
				if err != nil {
					return false, err
				}
				if valid {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// validateArgon2 derives the argon2 key for plainPass. Then it compares the key
// with the stored hash. The derivation starts only after argon2Sem gives a slot.
// If the request ends first, the caller leaves the queue and no derivation runs.
func validateArgon2(ctx context.Context, plainPass string, p pwd) (bool, error) {
	// A request that already ended must not start a derivation. Without this
	// test the select below can take either case, because both are ready.
	if err := ctx.Err(); err != nil {
		return false, err
	}

	select {
	case argon2Sem <- struct{}{}:
		defer func() { <-argon2Sem }()
	case <-ctx.Done():
		return false, ctx.Err()
	}

	return argon2Derive(plainPass, p), nil
}

func runArgon2(plainPass string, p pwd) bool {
	var key []byte
	switch p.pwdType {
	case pwdTypeArgon2id:
		key = argon2.IDKey([]byte(plainPass), p.argon2Salt, p.argon2Time, p.argon2Memory, p.argon2Threads, p.argon2KeyLen)
	case pwdTypeArgon2i:
		key = argon2.Key([]byte(plainPass), p.argon2Salt, p.argon2Time, p.argon2Memory, p.argon2Threads, p.argon2KeyLen)
	default:
		return false
	}

	return subtle.ConstantTimeCompare(key, p.pwdOrig) == 1
}

// parseArgon2 decodes an argon2 htpasswd entry in PHC format. The warnings name
// the parameters above the recommended maxima. Such an entry still loads, thus
// an upgrade cannot stop a deployment that runs. The caller shows the warnings
// to the operator, who can then lower the cost. A parameter that makes the entry
// unusable gives an error. This applies if it is absent, if it is not a number,
// or if t or p is below 1. argon2 panics on the last condition.
func parseArgon2(password, prefix string) (pwd, []string, error) {
	// PHC format: $argon2id$v=19$m=65536,t=3,p=2$<base64-salt>$<base64-hash>
	// After stripping the prefix ($argon2id$ or $argon2i$), we have:
	// v=19$m=65536,t=3,p=2$<base64-salt>$<base64-hash>
	remainder := strings.TrimPrefix(password, prefix)
	parts := strings.Split(remainder, "$")
	if len(parts) != 4 {
		return pwd{}, nil, fmt.Errorf("expected 4 parts, got %d", len(parts))
	}

	// parts[0] = "v=19"
	if parts[0] != "v=19" {
		return pwd{}, nil, fmt.Errorf("unsupported argon2 version: %s", parts[0])
	}

	// parts[1] = "m=65536,t=3,p=2" (order-independent)
	var memory, time, threads uint64
	params := make(map[string]string)
	for _, kv := range strings.Split(parts[1], ",") {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			return pwd{}, nil, fmt.Errorf("invalid argon2 parameter: %s", kv)
		}
		params[pair[0]] = pair[1]
	}

	var parseErr error
	var warnings []string
	if v, ok := params["m"]; ok {
		memory, parseErr = strconv.ParseUint(v, 10, 32)
	} else {
		return pwd{}, nil, fmt.Errorf("missing argon2 parameter: m")
	}
	if parseErr != nil {
		return pwd{}, nil, fmt.Errorf("invalid argon2 parameter m: %w", parseErr)
	}
	if uint32(memory) > argon2MaxMemory {
		warnings = append(warnings, fmt.Sprintf("argon2 parameter m=%d KiB exceeds the recommended maximum of %d KiB", memory, argon2MaxMemory))
	}

	if v, ok := params["t"]; ok {
		time, parseErr = strconv.ParseUint(v, 10, 32)
	} else {
		return pwd{}, nil, fmt.Errorf("missing argon2 parameter: t")
	}
	if parseErr != nil {
		return pwd{}, nil, fmt.Errorf("invalid argon2 parameter t: %w", parseErr)
	}
	if time < 1 {
		return pwd{}, nil, fmt.Errorf("invalid argon2 parameter t: must be >= 1")
	}
	if uint32(time) > argon2MaxTime {
		warnings = append(warnings, fmt.Sprintf("argon2 parameter t=%d exceeds the recommended maximum of %d", time, argon2MaxTime))
	}

	if v, ok := params["p"]; ok {
		threads, parseErr = strconv.ParseUint(v, 10, 8)
	} else {
		return pwd{}, nil, fmt.Errorf("missing argon2 parameter: p")
	}
	if parseErr != nil {
		return pwd{}, nil, fmt.Errorf("invalid argon2 parameter p: %w", parseErr)
	}
	if threads < 1 {
		return pwd{}, nil, fmt.Errorf("invalid argon2 parameter p: must be >= 1")
	}
	if uint8(threads) > argon2MaxThreads {
		warnings = append(warnings, fmt.Sprintf("argon2 parameter p=%d exceeds the recommended maximum of %d", threads, argon2MaxThreads))
	}

	// parts[2] = base64-encoded salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return pwd{}, nil, fmt.Errorf("invalid argon2 salt encoding: %w", err)
	}

	// parts[3] = base64-encoded hash
	hash, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return pwd{}, nil, fmt.Errorf("invalid argon2 hash encoding: %w", err)
	}

	pwdType := pwdTypeArgon2id
	if prefix == pwdPrefixArgon2i {
		pwdType = pwdTypeArgon2i
	}

	return pwd{
		pwdOrig:       hash,
		pwdPrefix:     prefix,
		pwdType:       pwdType,
		argon2Time:    uint32(time),
		argon2Memory:  uint32(memory),
		argon2Threads: uint8(threads),
		argon2KeyLen:  uint32(len(hash)),
		argon2Salt:    salt,
	}, warnings, nil
}

func apr1MD5(pass, salt, pref string) []byte {
	var passLen int = len(pass)

	h := md5.New()

	h.Write([]byte(pass + salt + pass))
	bin := h.Sum(nil)

	h.Reset()
	h.Write([]byte(pass + pref + salt))

	for i := passLen; i > 0; i -= aprMd5DigestSize {
		if i > aprMd5DigestSize {
			h.Write(bin[0:aprMd5DigestSize])
		} else {
			h.Write(bin[0:i])
		}
	}

	for i := passLen; i > 0; i >>= 1 {
		if (i & 1) == 1 {
			h.Write([]byte{0})
		} else {
			h.Write([]byte(pass[0:1]))
		}
	}

	sum := h.Sum(nil)

	for i := 0; i < aprMuddleRounds; i++ {
		h.Reset()

		if (i & 1) == 1 {
			h.Write([]byte(pass))
		} else {
			h.Write(sum)
		}

		if (i % 3) != 0 {
			h.Write([]byte(salt))
		}

		if (i % 7) != 0 {
			h.Write([]byte(pass))
		}

		if (i & 1) == 1 {
			h.Write(sum)
		} else {
			h.Write([]byte(pass))
		}

		copy(sum, h.Sum(nil))
	}

	buf := bytes.Buffer{}
	buf.Grow(len(pref) + len(salt) + 1 + 22)
	buf.WriteString(pref)
	buf.WriteString(salt)
	buf.WriteByte('$')

	add := func(a, b, c byte, last bool) {
		v := (uint(a) << 16) + (uint(b) << 8) + uint(c)

		iterations := 4
		if last {
			iterations = 2
		}

		for i := 0; i < iterations; i++ {
			buf.WriteByte(aprCharacters[v&0x3f])
			v >>= 6
		}
	}

	add(sum[0], sum[6], sum[12], false)
	add(sum[1], sum[7], sum[13], false)
	add(sum[2], sum[8], sum[14], false)
	add(sum[3], sum[9], sum[15], false)
	add(sum[4], sum[10], sum[5], false)
	add(0, 0, sum[11], true)

	return buf.Bytes()
}
