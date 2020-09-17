package accesscontrol

import "testing"

func Test_Apr1MD5(t *testing.T) {
	var exp, res string

	exp = "$apr1$NPXZYWba$/ebZ19mhDyKnsuM/cRaxq0"
	res = string(_Apr1MD5("xxx", "NPXZYWba", "$apr1$"))
	if exp != res {
		t.Errorf("Got unexpected password: '%s', want '%s'", res, exp)
	}

	exp = "$apr1$4z8NMYQV$TexsH1pVjUbkarHcVB2q/0"
	res = string(_Apr1MD5("s", "4z8NMYQV", "$apr1$"))
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

	if !validateAccessData("john", pass, data) {
		t.Error("Unexpected validation failed")
	}

	// $ htpasswd -bB .htpasswd jane my-pass
	//jane:$2y$05$/uonQYUtwmVv.6AF38IhGeqlvIMPIM5jevzIQ.8RBENUgkCqbJYTm
	data["jane"] = pwd{
		pwdOrig:   []byte("$2y$05$/uonQYUtwmVv.6AF38IhGeqlvIMPIM5jevzIQ.8RBENUgkCqbJYTm"),
		pwdPrefix: "$2y$",
		pwdSalt:   "05",
		pwdType:   pwdTypeBcrypt,
	}

	if !validateAccessData("jane", pass, data) {
		t.Error("Unexpected validation failed")
	}
}
