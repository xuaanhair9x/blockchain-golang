package blockchain

import(
	//"crypto/sha256"
	"bytes"
	"time"
	//"fmt"
	"log"
	"encoding/gob"
)

type Block struct {
	Timestamp int64
	Hash []byte
	Transactions []*Transaction // define it as list pointer because we need to access to child element of transaction
	PreviousHash []byte
	Nonce int
	Heigh int
}

func (block *Block) HashTransaction() []byte {
	var txHashed [][]byte
	for _, tx := range block.Transactions {
		txHashed = append(txHashed, tx.Serialize())
	}
	// hashed := sha256.Sum256(bytes.Join(txHashed, []byte{}))
	// return hashed[:]
	merkelTree := NewMerkelTree(txHashed)
	return merkelTree.RootNode.Data
}

func CreateBlock(txs []*Transaction, prevHash []byte, heigh int) *Block {

	block := &Block{}
	block.Timestamp = time.Now().Unix()
	block.Hash = []byte{}
	block.Transactions = txs
	block.Nonce = 0
	block.Heigh = heigh
	block.PreviousHash = prevHash

	//block := &Block{time.Now().Unix(), []byte{}, []*Transaction{}, []byte{}, 0, 0}

	proof := NewProof(block)
	nonce, hash := proof.Run()

	block.Nonce = nonce
	block.Hash = hash

	return block
}

func (block *Block) SerializeBlock() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(block)
	
	Handle(err)

	return buf.Bytes()
}

func DeserializeBlock(encodeBlock []byte) *Block {
	var block Block
	// func NewReader(b []byte) *Reader 
	newReader := bytes.NewReader(encodeBlock)

	//func NewDecoder(r io.Reader) *Decoder
	dec := gob.NewDecoder(newReader)

	err := dec.Decode(&block)

	Handle(err)

	return &block
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
