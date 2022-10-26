package blockchain

import(
	"encoding/gob"
	"bytes"
	"log"
	"crypto/sha256"
	"github.com/hai-nguyen/golang-blockchain/wallet"
)

type TxInput struct {
	Id []byte
	Out int
	Signiture []byte
	PubKey []byte
}

type TxOutputs struct {
	Outputs []TxOutput
}

type TxOutput struct {
	Value int
	PubKeyHash []byte
}

func (tx *Transaction) SetId()  {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	hash := sha256.Sum256(buf.Bytes())
	tx.Id = hash[:]
}

func (tx *Transaction) SerializeOutput() []byte  {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx.Outputs)
	if err != nil {
		log.Panic(err)
	}
	
	return buf.Bytes()
}

func (outputs *TxOutputs) Serialize() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(outputs)
	if err != nil {
		log.Panic(err)
	}
	
	return buf.Bytes()
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := wallet.DecodeBase58(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}


func NewTXOutput(value int, address []byte) *TxOutput {	
	txo := &TxOutput{value, nil}
	txo.Lock(address)

	return txo
}


func DeserializeOutput(encodeOutputs []byte) *TxOutputs {

	outputs := &TxOutputs{}
	// func NewReader(b []byte) *Reader 
	newReader := bytes.NewReader(encodeOutputs)

	//func NewDecoder(r io.Reader) *Decoder
	dec := gob.NewDecoder(newReader)

	err := dec.Decode(outputs)

	if err != nil {
		log.Panic(err)
	}
	
	return outputs
}

func (out *TxOutput) CheckLockOutput(pubKeyHash []byte)  bool {
	if bytes.Compare(out.PubKeyHash, pubKeyHash)  == 0 {
		return true;
	}
	return false
}