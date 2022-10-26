package wallet

import (
	"crypto/sha256"
	"log"
	"bytes"
	"crypto/elliptic"
	"crypto/ecdsa"
	"crypto/rand"
	"golang.org/x/crypto/ripemd160"
	//"fmt"
)
type Wallet struct {
	PrivKey []byte
	PubKey []byte
}

const (
	checksumLength = 4
	version = byte(0x00)
)

func GeneratePairKey() ([]byte,[]byte) {
	ec := elliptic.P256()

	privKey, err := ecdsa.GenerateKey(ec, rand.Reader)

	if err != nil {
		log.Panic(err)
	}
	pubKey := append(privKey.PublicKey.X.Bytes(),privKey.PublicKey.Y.Bytes()...)

	return privKey.D.Bytes(), pubKey
}

func CreateWallet() *Wallet{
	pub, priv := GeneratePairKey()
	wallet := Wallet{pub, priv}

	return &wallet
}

func (w *Wallet) GetAddress() []byte{
	pubHash := w.HashPublicKey()
	versionPubHash := append([]byte{version}, pubHash ...)

	checkSum := CheckSum(versionPubHash[:])

	fullHash := append(versionPubHash, checkSum ...)

	address := encodeBase58(fullHash[:])

	return address[:]
}

func (w *Wallet) HashPublicKey() []byte {
	hashSha256 := sha256.Sum256(w.PubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(hashSha256[:])

	if err != nil {
		log.Panic(err)
	}

	ripemd160 := hasher.Sum(nil)
	return ripemd160[:]

}

func CheckSum(versionPubHash []byte)  []byte{
	hashFirst := sha256.Sum256(versionPubHash)
	hashSecond := sha256.Sum256(hashFirst[:])

	return hashSecond[:checksumLength]
}

func ValidateAddress(address []byte) bool {

	fullHash := DecodeBase58(address)
	
	lengthVersionPubHash := len(fullHash)-checksumLength

	checkSum := fullHash[lengthVersionPubHash:]
	
	versionPubHash := fullHash[:lengthVersionPubHash]

	checkSumForHash := CheckSum(versionPubHash)

	return bytes.Compare(checkSum, checkSumForHash) == 0
}