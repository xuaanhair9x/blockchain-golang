package blockchain

import(
	"github.com/dgraph-io/badger"
	"log"
	"os"
	"fmt"
	"bytes"
	"encoding/hex"
	"errors"
)

const (
	dbPath      = "./tmp/blocks_%s"
	genesisData = "First Transaction from Genesis"
)

type Blockchain struct {
	Db *badger.DB
	Lasthash []byte
}

type BlockchainIterator struct {
	Db *badger.DB
	CurrentHash []byte
}

func DBexists(path string) bool {
	if _, err := os.Stat(path + "/MANIFEST"); os.IsNotExist(err) {
		return false
	}

	return true
}

func ContinueBlockchain(NodeID string)  *Blockchain {
	path := fmt.Sprintf(dbPath, NodeID)
	if !DBexists(path) {
		log.Panic("Blockchain is not existed.")
	}

	var lashHash []byte
	
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path

	db, err := badger.Open(opts)

	if err != nil {
		Handle(err)
	}
	
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)
		
		err = item.Value(func(v []byte) error {
			lashHash = v
			return nil
		})

		return nil
	  })
	  
	blockchain := Blockchain{db, lashHash}
	
	return &blockchain
}

func (blockchain *Blockchain) Iterator() *BlockchainIterator  {
	return &BlockchainIterator{blockchain.Db, blockchain.Lasthash}
}

func (iter *BlockchainIterator) Next() *Block {

	var block *Block

	iter.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		
		err = item.Value(func(value []byte) error {
			block = DeserializeBlock(value)
			return nil
		})
		

		return nil
	})

	iter.CurrentHash = block.PreviousHash

	return block
}

func (chain *Blockchain) BestHeight() int {
	
	var bestHeight int

	err := chain.Db.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte("lh"))
		Handle(err)

		var lashHash []byte
		err = item.Value(func(val []byte) error {
			lashHash = val
			return nil
		})
		Handle(err)

		item, err = txn.Get(lashHash)
		Handle(err)

		var lastBlockEnc []byte
		err = item.Value(func(val []byte) error {
			lastBlockEnc = val
			return nil
		})
		Handle(err)


		lastBlock := DeserializeBlock(lastBlockEnc)

		bestHeight = lastBlock.Heigh

		return nil
	})
	Handle(err)

	return bestHeight
}

func (chain *Blockchain) GetAllHashesBlock() [][]byte {

	iter := chain.Iterator()
	
	var hashes [][]byte

	for {
	
		block := iter.Next()

		hashes = append(hashes, block.Hash)

		if len(block.PreviousHash) == 0 {
			break
		}
	}

	return hashes
}

func (chain *Blockchain) GetBlockByHash(hash []byte) *Block  {
	var block Block

	err := chain.Db.View(func(txn *badger.Txn) error {

		item, err := txn.Get(hash)
		Handle(err)

		var blockEnc []byte
		err = item.Value(func(val []byte) error {
			blockEnc = val
			return nil
		})
		Handle(err)

		block = *DeserializeBlock(blockEnc)

		return nil
	})
	Handle(err)

	return &block
}

func InitBlockchain(pubKey []byte, NodeID string)  *Blockchain {
	// Check blockchain is exist.
	
	path := fmt.Sprintf(dbPath, NodeID)

	if DBexists(path) {
		log.Panic("Blockchain already existed.")
	}

	var lashHash []byte
	
	opts := badger.DefaultOptions(path)
	opts.Dir = path
	opts.ValueDir = path

	db, err := badger.Open(opts)

	if err != nil {
		Handle(err)
	}

	err = db.Update(func(txn *badger.Txn) error {

		txCoinBase := CoinBaseTx(genesisData, pubKey)
		genesisBlock := CreateBlock([]*Transaction{txCoinBase}, []byte{}, 0)
		fmt.Println("Genesis created")
		fmt.Printf("Hash: %x",genesisBlock.Hash)
	
		err := txn.Set(genesisBlock.Hash, genesisBlock.SerializeBlock())
		Handle(err)
		err = txn.Set([]byte("lh"), genesisBlock.Hash)
		
		lashHash = genesisBlock.Hash


		return err
	  })
	
	blockchain := Blockchain{db, lashHash}

	return &blockchain
}

func (chain *Blockchain) FindUTXO_Old(pubKey []byte) []TxOutput{
	
	upspendTxs := chain.FindUnspendTransaction(pubKey)

	var outputs []TxOutput
	for _, tx := range upspendTxs {
		for _, out := range tx.Outputs {
			if out.CheckLockOutput(pubKey) {
				outputs = append(outputs, out)
			}
		}
	}

	return outputs
}

func (chain *Blockchain) FindUTXO()  map[string]TxOutputs  {

	iter := chain.Iterator()
	utxos := make(map[string]TxOutputs)
	spendITXOs := make(map[string][]int)

	for {
	
		block := iter.Next()
		
		for _, tx := range block.Transactions {

			txEncId := hex.EncodeToString(tx.Id)
			outputs := &TxOutputs{}

			Outputs:
			for id, out := range tx.Outputs {
				if (spendITXOs[txEncId]) != nil {
					for _, OutIndex := range spendITXOs[txEncId] {
						if id == OutIndex { 
							continue Outputs
						}
					}
				}
				outputs.Outputs = append(outputs.Outputs, out)
			}
			
			utxos[txEncId] = *outputs
			
			if (!tx.IsCoinBase()) { // Because coinbase does not have inputs
				for _, in := range tx.Inputs {
					idInEnc := hex.EncodeToString(in.Id)
					spendITXOs[idInEnc] = append(spendITXOs[idInEnc], in.Out)
				}
			}

		}

		if len(block.PreviousHash) == 0 {
			break
		}
	}
	//log.Panic(222)
	return utxos
}

/**
Find unspend transaction for the address
...
**/
func (chain *Blockchain) FindUnspendTransaction(pubKeyHash []byte)  []Transaction {
	iter := chain.Iterator()
	var upspendTxs []Transaction
	spendITXOs := make(map[string][]int)

	for {
	
		block := iter.Next()
		
		for _, tx := range block.Transactions {

			txEncId := hex.EncodeToString(tx.Id)
			
			Outputs:
			for id, out := range tx.Outputs {
				if (spendITXOs[txEncId]) != nil {
					for _, OutIndex := range spendITXOs[txEncId] {
						if id == OutIndex { 
							continue Outputs
						}
					}
				}
				if (out.CheckLockOutput(pubKeyHash)) {
					upspendTxs = append(upspendTxs, *tx)
				}
			}
			
			if (!tx.IsCoinBase()) { // Because coinbase does not have inputs
				for _, in := range tx.Inputs {
					idInEnc := hex.EncodeToString(in.Id)
					spendITXOs[idInEnc] = append(spendITXOs[idInEnc], in.Out)
				}
			}

		}

		if len(block.PreviousHash) == 0 {
			break
		}
	}
	return upspendTxs
}

func (chain *Blockchain) FindSpendableTransaction(pubKeyHash []byte, amount int) (int, map[string][]int) {
	upspendTxs := chain.FindUnspendTransaction(pubKeyHash)
	acc := 0
	spendOut := make(map[string][]int)

	Accumulate:
	for _, tx := range upspendTxs {
		txIdEnc := hex.EncodeToString(tx.Id)
		for idOut, out := range tx.Outputs {
			if out.CheckLockOutput(pubKeyHash) && acc < amount {
				acc = acc + out.Value
				spendOut[txIdEnc] = append(spendOut[txIdEnc],idOut) // In a tx, there is a output at maximum for specific address
			}
		
			if (acc > amount) {
				break Accumulate
			}
			
		}
	}

	if acc < amount {
		Handle(errors.New("There is not enough coin for sending....."))
	}

	return acc, spendOut
}

func (chain *Blockchain) FindTransaction(txId []byte) *Transaction {

	iter := chain.Iterator()

	for {
		
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.Id, txId) == 0 {
				return tx
			}
		}

		if len(block.PreviousHash) == 0 {
			break
		}
	}

	return nil
}

func (chain *Blockchain) SignTransaction(tx *Transaction, privateKey []byte) error {
	
	prevTxs := make(map[string]*Transaction)

	// Get Prev txs from tx.Inputs
	for _, in := range tx.Inputs {
		tx := chain.FindTransaction(in.Id)
		if tx == nil {
			return errors.New("SignTransaction: PreTx is not existed .....")
		}
		txIdEnc := hex.EncodeToString(tx.Id)
		prevTxs[txIdEnc] = tx
	}

	err := tx.sign(privateKey, prevTxs)

	return err
}

func (chain *Blockchain) VerifyTransactions(txs []*Transaction) error {

	Verify:
	for _, tx := range txs {
		
		if tx.IsCoinBase() {
			continue Verify
		}

		prevTxs := make(map[string]*Transaction)
		// Get Prev txs from tx.Inputs
		for _, in := range tx.Inputs {
			txPrev := chain.FindTransaction(in.Id)
			if txPrev == nil {
				return errors.New("VerifyTransactions: PreTx is not existed .....")
			}
			txIdEnc := hex.EncodeToString(txPrev.Id)
			//fmt.Printf("txIdEnc: %s, txPrev: %x \n ", txIdEnc, txPrev )
			txPrev.String()
			prevTxs[txIdEnc] = txPrev
		}
		
		if tx.Verify(prevTxs) == false {
			return errors.New("Tx is invalid .....")
		}
	}

	return nil
}

/**

- Verify the transaction
- Get last hash from Database
- Get last encode block from Database base on last hash
- Deserialize the encode block
- Get heigh of block
- Create new block (txs, prevHash = lasthash, heigh + 1)
- Set last hash = new block hash 
- Save new block to database
- update last hash for blockchain.
**/
func (chain *Blockchain) MineBlock(txs []*Transaction) *Block {
	
	errVerify := chain.VerifyTransactions(txs)
	Handle(errVerify)
	
	newBlock := &Block{}

	err := chain.Db.Update(func(txn *badger.Txn) error {
		
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		var lashHash []byte
		err = item.Value(func(val []byte) error {
			lashHash = val
			return nil
		})
		Handle(err)

		item, err = txn.Get(lashHash)
		Handle(err)

		var lastBlockEnc []byte
		err = item.Value(func(val []byte) error {
			lastBlockEnc = val
			return nil
		})
		Handle(err)

		lastBlock := DeserializeBlock(lastBlockEnc)

		maxHeigh := lastBlock.Heigh

		// Mine block
		newBlock = CreateBlock(txs, lashHash, maxHeigh + 1)

		// Update block to database
		newBlockEnc := newBlock.SerializeBlock()
		err = txn.Set(newBlock.Hash, newBlockEnc)
		Handle(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.Lasthash = newBlock.Hash

		return err
	})	

	Handle(err)
	
	return newBlock
}

/**

- Check block is exited in database
- Save block to database
- Get current last block and compare with block
	if heightBlock > heightCurrentLastBlock
		update lastHash = hashBlock

**/
func (chain *Blockchain) AddBlock(block *Block) {

	err := chain.Db.Update(func(txn *badger.Txn) error {
		
		// Check if block is exited
		if _, err := txn.Get(block.Hash); err == nil {
			return nil
		}

		// Update block to database
		blockEnc := block.SerializeBlock()
		err := txn.Set(block.Hash, blockEnc)
		Handle(err)

		item, err := txn.Get(chain.Lasthash)
		Handle(err)

		var lastBlockEnc []byte
		err = item.Value(func(val []byte) error {
			lastBlockEnc = val
			return nil
		})
		Handle(err)

		lastBlock := DeserializeBlock(lastBlockEnc)
		maxHeigh := lastBlock.Heigh

		if (block.Heigh > maxHeigh) {
			err = txn.Set([]byte("lh"), block.Hash)
			chain.Lasthash = block.Hash
			Handle(err)
		}
		return err
	})	
	Handle(err)

}