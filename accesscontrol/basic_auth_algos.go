package accesscontrol

import (
	"bytes"
	"crypto/md5"
	"crypto/subtle"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	pwdPrefixApr1     = "$apr1$"
	pwdPrefixBcrypt2a = "$2a$"
	pwdPrefixBcrypt2b = "$2b$"
	pwdPrefixBcrypt2x = "$2x$"
	pwdPrefixBcrypt2y = "$2y$"
	pwdPrefixMD5      = "$1$"
)

const (
	pwdTypeUnknown = iota
	pwdTypeApr1
	pwdTypeBcrypt
	pwdTypeMD5
)

const (
	aprCharacters    = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	aprMd5DigestSize = 16
	aprMuddleRounds  = 1000
)

var pwdPrefixes = map[string]int{
	pwdPrefixApr1:     pwdTypeApr1,
	pwdPrefixBcrypt2a: pwdTypeBcrypt,
	pwdPrefixBcrypt2b: pwdTypeBcrypt,
	pwdPrefixBcrypt2x: pwdTypeBcrypt,
	pwdPrefixBcrypt2y: pwdTypeBcrypt,
	pwdPrefixMD5:      pwdTypeMD5,
}

type htData map[string]pwd

type pwd struct {
	pwdOrig   []byte
	pwdPrefix string
	pwdSalt   string
	pwdType   int
}

func getPwdType(pass string) int {
	for p, t := range pwdPrefixes {
		if strings.HasPrefix(pass, p) {
			return t
		}
	}

	return pwdTypeUnknown
}

func validateAccessData(plainUser, plainPass string, data htData) bool {
	for user, pass := range data {
		if user == plainUser {
			switch pass.pwdType {
			case pwdTypeApr1:
				fallthrough
			case pwdTypeMD5:
				if subtle.ConstantTimeCompare(apr1MD5(plainPass, pass.pwdSalt, pass.pwdPrefix), pass.pwdOrig) == 1 {
					return true
				}
			case pwdTypeBcrypt:
				if err := bcrypt.CompareHashAndPassword(pass.pwdOrig, []byte(plainPass)); err == nil {
					return true
				}
			}
		}
	}

	return false
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
