package wallet

import (
	"github.com/btcsuite/btcutil/base58"
)

func EncodeBase58(input []byte) []byte {
	encoded := base58.Encode(input)
	return []byte(encoded)
}

func DecodeBase58(input []byte) []byte  {
	decode := base58.Decode(string(input[:]))
	return decode
}