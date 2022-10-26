package blockchain

import (
	"crypto/elliptic"
	"math/big"
	"io"
	"crypto/sha512"
	"crypto/aes"
	"crypto/cipher"
	//"fmt"
)


var one = new(big.Int).SetInt64(1)
const (
	aesIV = "IV for ECDSA CTR"
)
type zr struct{}

// Read replaces the contents of dst with zeros. It is safe for concurrent use.
func (zr) Read(dst []byte) (n int, err error) {
	for i := range dst {
		dst[i] = 0
	}
	return len(dst), nil
}
var zeroReader = zr{}


/** 
- dA is private key
- QA is public key
- k is ramdom key and it is not the same with dA.

** Goal: Create a pair (r, s)
- QA= dA × G
- P = k×G => P(x, y) => x = r
- s=k^(−1)(hash+dA×r)
**/
func signEcdsa(rand io.Reader, priv []byte, hash []byte) (r, s *big.Int, err error) {

	entropy := make([]byte, 32)
	_, err = io.ReadFull(rand, entropy)
	if err != nil {
		return
	}

	// Initialize an SHA-512 hash context; digest...
	md := sha512.New()
	md.Write(priv) // the private key,
	md.Write(entropy)        // the entropy,
	md.Write(hash)           // and the input hash;
	key := md.Sum(nil)[:32]  // and compute ChopMD-256(SHA-512),
	// which is an indifferentiable MAC.

	// Create an AES-CTR instance to use as a CSPRNG.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	// Create a CSPRNG that xors a stream of zeros with
	// the output of the AES-CTR instance.
	csprng := &cipher.StreamReader{
		R: zeroReader,
		S: cipher.NewCTR(block, []byte(aesIV)),
	}
	//fmt.Println(*csprng)
	
	curve := elliptic.P256()
	N := curve.Params().N

	var k, kInv *big.Int
	for {
		for {
			k, err = randFieldElement(curve, *csprng)
			if err != nil {
				r = nil
				return
			}

			kInv = fermatInverse(k, N) // N != 0

			r, _ = curve.ScalarBaseMult(k.Bytes())
			r.Mod(r, N)
			if r.Sign() != 0 {
				break
			}
		}

		e := hashToInt(hash, curve)

		s = new(big.Int).Mul(new(big.Int).SetBytes(priv), r)
		s.Add(s, e)
		s.Mul(s, kInv)
		s.Mod(s, N) // N != 0
		if s.Sign() != 0 {
			break
		}
	}

	return r,s,err 
}

// hashToInt converts a hash value to an integer. Per FIPS 186-4, Section 6.4,
// we use the left-most bits of the hash to match the bit-length of the order of
// the curve. This also performs Step 5 of SEC 1, Version 2.0, Section 4.1.3.
func hashToInt(hash []byte, c elliptic.Curve) *big.Int {
	orderBits := c.Params().N.BitLen()
	orderBytes := (orderBits + 7) / 8
	if len(hash) > orderBytes {
		hash = hash[:orderBytes]
	}

	ret := new(big.Int).SetBytes(hash)
	excess := len(hash)*8 - orderBits
	if excess > 0 {
		ret.Rsh(ret, uint(excess))
	}
	return ret
}

// fermatInverse calculates the inverse of k in GF(P) using Fermat's method
// (exponentiation modulo P - 2, per Euler's theorem). This has better
// constant-time properties than Euclid's method (implemented in
// math/big.Int.ModInverse and FIPS 186-4, Appendix C.1) although math/big
// itself isn't strictly constant-time so it's not perfect.
func fermatInverse(k, N *big.Int) *big.Int {
	two := big.NewInt(2)
	nMinus2 := new(big.Int).Sub(N, two)
	return new(big.Int).Exp(k, nMinus2, N)
}

// randFieldElement returns a random element of the order of the given
// curve using the procedure given in FIPS 186-4, Appendix B.5.1.
func randFieldElement(c elliptic.Curve, rand io.Reader) (k *big.Int, err error) {
	params := c.Params()
	// Note that for P-521 this will actually be 63 bits more than the order, as
	// division rounds down, but the extra bit is inconsequential.
	b := make([]byte, params.N.BitLen()/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}

	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}

/**
P=s^(−1)×hash×G+s^(−1)×r×QA
**/
func verifyEcdsa(xPub, yPub  *big.Int, hash []byte, r, s *big.Int) bool {
	c := elliptic.P256()
	N := c.Params().N

	if r.Sign() <= 0 || s.Sign() <= 0 {
		return false
	}
	if r.Cmp(N) >= 0 || s.Cmp(N) >= 0 {
		return false
	}

	e := hashToInt(hash, c)
	var w *big.Int
	w = new(big.Int).ModInverse(s, N) // s^(-1)

	// u1 = s^(-1)*hash mod N
	u1 := e.Mul(e, w)
	u1.Mod(u1, N)

	// u2 = s^(−1)×r mod N
	u2 := w.Mul(r, w)
	u2.Mod(u2, N)


	// Check if implements S1*g + S2*p
	var x, y *big.Int
	x1, y1 := c.ScalarBaseMult(u1.Bytes())
	x2, y2 := c.ScalarMult(xPub, yPub, u2.Bytes())
	x, y = c.Add(x1, y1, x2, y2)

	if x.Sign() == 0 && y.Sign() == 0 {
		return false
	}
	x.Mod(x, N)
	return x.Cmp(r) == 0
}