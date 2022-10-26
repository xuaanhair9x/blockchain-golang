package blockchain


import(
	"bytes"
	"encoding/gob"
	"crypto/sha256"
	"github.com/hai-nguyen/golang-blockchain/wallet"
	"encoding/hex"
	"crypto/rand"
	"math/big"
	"log"
	"fmt"
	"strings"
	"errors"
)

type Transaction struct {
	Id []byte
	Inputs []TxInput
	Outputs []TxOutput
}

func CoinBaseTx(dataGenesis string, pubKey []byte)  *Transaction {
	tx := &Transaction{}

	var inputs []TxInput
	var outputs []TxOutput
	var pubKeyHash []byte
	
	if dataGenesis == "" {
		pubKeyHash = make([]byte, 12)
		_, err := rand.Read(pubKeyHash)
		Handle(err)
	} else {
		pubKeyHash = []byte(dataGenesis)
	}
	input := &TxInput{[]byte{}, -1, []byte{}, pubKeyHash}
	inputs = append(inputs, *input )

	output := &TxOutput{30, pubKey}
	outputs = append(outputs, *output)

	tx.Inputs = inputs
	tx.Outputs = outputs

	tx.SetID()

	return tx
}

func NewTransaction(wlFrom *wallet.Wallet, to []byte, amount int, chain *Blockchain) (*Transaction, error) {
	var outputs []TxOutput
	var inputs []TxInput
	
	spendOut, acc := chain.FindSpendableOutputs(wlFrom.HashPublicKey(), amount)
	
	for txIdEnc, outIndexes := range spendOut {
		ixId, err := hex.DecodeString(txIdEnc)
		Handle(err)
		for _, outId := range outIndexes {
			inputs = append(inputs, TxInput{ixId, outId, nil, wlFrom.PubKey})
		}
	}

	//outputs = append(outputs,TxOutput{ amount, wlTo.HashPublicKey()})
	outputs = append(outputs,*NewTXOutput(amount, to))

	
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc - amount, wlFrom.GetAddress()))
	}
    //TxOutput{ acc - amount, wlFrom.HashPublicKey()
	transaction := &Transaction{nil, inputs, outputs}
	
	err := chain.SignTransaction(transaction, wlFrom.PrivKey)
	
	if err != nil {
		Handle(err)
	}

	transaction.SetId()
	
	return transaction, nil
}

func (tx *Transaction) TrimmedCopy() *Transaction {
	var inputs []TxInput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.Id, in.Out, []byte{}, nil })
	} 
	
	return &Transaction{nil, inputs, tx.Outputs}
}

func (tx *Transaction) Serialize() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	
	return buf.Bytes()
}

func DeserializeTransaction(txEnc []byte) *Transaction {
	var tx Transaction
	newReader := bytes.NewReader(txEnc)
	dec := gob.NewDecoder(newReader)
	err := dec.Decode(&tx)
	Handle(err)
	return &tx
}

func (tx *Transaction) sign(privateKey []byte, prevTxs map[string]*Transaction) error {
	
	if tx.IsCoinBase() {
		return errors.New("The transaction is coinbase .....")
	}

	// Check to make sure that all input have its transaction in prevTxs
	for _, in := range tx.Inputs {
		txIdEnc := hex.EncodeToString(in.Id)
		if prevTxs[txIdEnc] == nil {
			return errors.New("There is no transaction for this input...")
		}

	}

	copyTx := tx.TrimmedCopy()

	/**
	Example: txCopy have 2 inputs, outputs is the same with txOrigin
	txCopy = {
		Id: nill
		Inputs:
		[0]	Id: abc
			Out: 1
			Sig: []
			PubHash: []
		[1]	Id: bcd
			Out: 0
			Sig: []
			PubHash: []
	}

	round 1: 
	txCopy = {
		Id: nill
		Inputs:
		[0]	Id: abc
			Out: 1
			Sig: []
			PubHash: ffffff => Get this pubhash from PrevTxs, it is Public key of output
		[1]	Id: bcd
			Out: 0
			Sig: []
			PubHash: []
	}

	round 2: 
	txCopy = {
		Id: nill
		Inputs:
		[0]	Id: abc
			Out: 1
			Sig: []
			PubHash: []
		[1]	Id: bcd
			Out: 0
			Sig: []
			PubHash: xxxxx => Get this pubhash from PrevTxs, it is Public key of output
	}
	hash(txCopy) as data to sign

	**/
	for index, in := range copyTx.Inputs {
		txIdEnc := hex.EncodeToString(in.Id)
		// Replace public key for address

		copyTx.Inputs[index].Signiture = []byte{}
		copyTx.Inputs[index].PubKey = prevTxs[txIdEnc].Outputs[in.Out].PubKeyHash

		copyTx.SetID()

		//dataSign := copyTx.Id
		
		dataSign := []byte(fmt.Sprintf("%x\n", copyTx))

		r, s, err := signEcdsa(rand.Reader, privateKey, dataSign)
		Handle(err)
	
		var signiture []byte
		signiture = append(signiture, r.Bytes() ...)
		signiture = append(signiture, s.Bytes() ...)

		tx.Inputs[index].Signiture = signiture

		copyTx.Id = nil
		copyTx.Inputs[index].PubKey = []byte{}
	}

	return nil
}

func (tx *Transaction) Verify(prevTxs map[string]*Transaction) bool {
	
	if tx.IsCoinBase() {
		return true
	}

	// Check to make sure that all input have its transaction in prevTxs
	for index, in := range tx.Inputs {
		txIdEnc := hex.EncodeToString(in.Id)
		if prevTxs[txIdEnc] == nil {
			log.Panic("There is no transaction for this input tx: %s, indexInput: %d", tx.Id, index)
		}
	}

	// Check to make sure that all input have its transaction in prevTxs
	copyTx := tx.TrimmedCopy()

	/**
	- Set copyTxs like sign function => Data to sign
	- Get r,s  *big.Int from signiture
	- Get x,y  *big.Int from public key
	- verify(x,y,r,s,dataSign)
	**/
	for index, in := range tx.Inputs {
		signiture := in.Signiture
		txIdEnc := hex.EncodeToString(in.Id)
		// Replace public key for address

		pubKey := in.PubKey
		copyTx.Inputs[index].Signiture = []byte{}
		copyTx.Inputs[index].PubKey = prevTxs[txIdEnc].Outputs[in.Out].PubKeyHash

		r := big.Int{}
		s := big.Int{}
		lenSig := len(signiture)
		r.SetBytes(signiture[:(lenSig/2)])
		s.SetBytes(signiture[(lenSig/2):])

		x := big.Int{}
		y := big.Int{}
		lenKey := len(pubKey)
		x.SetBytes(pubKey[:(lenKey/2)])
		y.SetBytes(pubKey[(lenSig/2):])

		copyTx.SetID()

		//dataSign := copyTx.Id
		dataSign := []byte(fmt.Sprintf("%x\n", copyTx))

		if verifyEcdsa(&x, &y, dataSign, &r, &s) == false {
			return false
		}

		copyTx.Id = nil
		copyTx.Inputs[index].PubKey = []byte{}
	}

	return true
}

func (tx *Transaction) SetID()  {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(tx)
	
	Handle(err)

	hash := sha256.Sum256(buf.Bytes())
	tx.Id = hash[:]
}

func (tx *Transaction) IsCoinBase() bool {
	if len(tx.Inputs[0].Id) == 0 && len(tx.Inputs) == 1 && tx.Inputs[0].Out == -1 {
		return true
	}

	return false
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.Id))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.Id))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signiture))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
