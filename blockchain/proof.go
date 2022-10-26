package blockchain

import(
	"math/big"
	"encoding/binary"
	"crypto/sha256"
	"math"
	"bytes"
	"log"
)

const (
	difficult = 12
)
type ProofOfWork struct {
	Block *Block
	Target *big.Int
}

func NewProof(block *Block) *ProofOfWork {
	target := big.NewInt(1)

	// func (z *Int) Lsh(x *Int, n uint) *Int
	target.Lsh(target, 256-difficult)

	proof := ProofOfWork{block, target}
	return &proof
}

func (proof *ProofOfWork) InitData(nonce int) []byte {
	data := bytes.Join(
		[][]byte {
			proof.Block.PreviousHash,
			proof.Block.HashTransaction(),
			Int64ToByte(int64(nonce)),
			Int64ToByte(int64(proof.Block.Heigh)),
		}, []byte{}, )

	return data
}

func (proof *ProofOfWork) Run() (int, []byte){
	var nonce int
	var hash []byte

	nonce = 0
	for nonce < math.MaxInt64 {
		data := proof.InitData(nonce)
		hashData := sha256.Sum256(data)

		hashBigInt := new(big.Int)
		hashBigInt.SetBytes(hashData[:])
		hash = hashData[:]
		if proof.Target.Cmp(hashBigInt) == 1 {
			break
		}
		nonce++
	}

	return nonce, hash[:]
}

func Int64ToByte(num int64) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)

	if err != nil {
		log.Panic(err)
	}

	return buf.Bytes()
}

func (pow *ProofOfWork) Validate() bool {
	var intHash big.Int

	data := pow.InitData(pow.Block.Nonce)

	hash := sha256.Sum256(data)
	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.Target) == -1
}