package blockchain

import(
	"github.com/dgraph-io/badger"
	"encoding/hex"
	"errors"
	//"fmt"
	//"log"
)

var (
	utxoPrefix   = []byte("utxo-")
	prefixLength = len(utxoPrefix)
)

/**

Note: when we apply store utxo separate from blockchain, 
then Out in Input {Id, Out, Sig, Pubkey} will be set base on Outputs in utxo 

Update utxo from block
- Update available outputs of txs in Inputs of block, if its outputs spent then remove, else add
- Store new output of block into database


Outputs will be save with key = utxoPrefix + txID
**/
func (chain *Blockchain) Update(block *Block) error {
	
	err := chain.Db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if tx.IsCoinBase() == false{
				for _, in := range tx.Inputs {
					inID := append(utxoPrefix, in.Id ...)
	
					item, err := txn.Get(inID)
					Handle(err)
	
					var outs *TxOutputs
					err = item.Value(func(outsEnc []byte) error {
						outs = DeserializeOutput(outsEnc)
						return nil
					})
					Handle(err)
	
					outsUpdate := &TxOutputs{}
					for indexOut, out := range outs.Outputs {
						if (indexOut != in.Out) {
							outsUpdate.Outputs = append(outsUpdate.Outputs, out)
						}
					}
	
					if (len(outsUpdate.Outputs) == 0) {
						err = txn.Delete(inID)
					} else {
						err = txn.Set(inID, outsUpdate.Serialize())
					}
					Handle(err)
				}
			}

			outsNew := &TxOutputs{}
			inID := append(utxoPrefix, tx.Id ...)

			for _, out := range tx.Outputs {
				outsNew.Outputs = append(outsNew.Outputs, out)
			}
			err := txn.Set(inID, outsNew.Serialize())
			Handle(err)
		}

		return nil
	})

	return err
}


/**

This function is used for getting balance

Seeking all output of pubkeyHash
**/
func (chain *Blockchain) FindUnspentTransactions(pubKeyHash []byte) *TxOutputs{

	unspendOutputs := &TxOutputs{}
	
	err := chain.Db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			var outs *TxOutputs
			err := item.Value(func(outsEnc []byte) error {
				outs = DeserializeOutput(outsEnc)
				return nil
			})
			//fmt.Println(item.Key())
			for _, out := range outs.Outputs {
				//fmt.Println(out.Value)
				if out.CheckLockOutput(pubKeyHash) {
					unspendOutputs.Outputs = append(unspendOutputs.Outputs, out)
				}
			}
			
			Handle(err)
		}
		return nil
	})

	Handle(err)
	return unspendOutputs
}

/**

Find list outputs that contain enough coin for sending

Seeking outputs of pubkeyHash and satisfy that total coin of outputs is enough >= amount

spendOut [txId][]{outIndex}
**/
func (chain *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int)  (map[string][]int , int) {

	spendOut := make(map[string][]int)
	acc := 0

	err := chain.Db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		Work:
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			var outs *TxOutputs
			err := item.Value(func(outsEnc []byte) error {
				outs = DeserializeOutput(outsEnc)
				return nil
			})

			txId := k[prefixLength:]
			txIdEnc := hex.EncodeToString(txId)
			for indexOut, out := range outs.Outputs {
				if out.CheckLockOutput(pubKeyHash){
					spendOut[txIdEnc] = append(spendOut[txIdEnc], indexOut)
					acc = acc + out.Value
				}

				if (acc >= amount) {
					break Work
				}
			}
			Handle(err)
		}
		return nil
	})
	Handle(err)

	//log.Panic(acc)

	if acc < amount {
		Handle(errors.New("There is not enough coin for sending....."))
	}
	
	return spendOut, acc
}

/**

- Delete all record utxo
- Get UTXO from blockchain and update to utxo data 

**/
func (chain *Blockchain) Reindex() {
	
	err := chain.DeleteByPrefix(utxoPrefix)
	Handle(err)
	
	utxos := chain.FindUTXO()

	for idTxEnc, outs := range utxos {
		err := chain.Db.Update(func(txn *badger.Txn) error {
			key, err := hex.DecodeString(idTxEnc)
			//fmt.Printf("%s", idTxEnc)
			//fmt.Println()

			Handle(err)
			IdUtxo := append(utxoPrefix, key ...)
			err = txn.Set(IdUtxo, outs.Serialize())
			Handle(err)
			return nil
		})
		Handle(err)
	}
}

/**

- Create child function to delete records
- Set collection size, the size of colleting records, 
if it meet the maxnimun then move the collect record to child function
- Collection size is the maximum record that child function will handle

**/
func (chain *Blockchain)  DeleteByPrefix(prevFix []byte) error {
	deleteByKey := func (collections [][]byte) error {
		err := chain.Db.Update(func(txn *badger.Txn) error {
			for _, key := range collections {
				err := txn.Delete(key)
				Handle(err)
			}
			return nil
		})
		return err
	}

	collectionSize := 10000
	err := chain.Db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		collectionCount := 0
		collection := make([][]byte, 0, collectionSize)

		for it.Seek(prevFix); it.ValidForPrefix(prevFix); it.Next() {
			item := it.Item()
			key := item.Key()
			collection = append(collection, key)
			collectionCount ++

			if (collectionCount == collectionSize) {
				err := deleteByKey(collection)
				collectionCount = 0
				collection = make([][]byte, 0, collectionSize)
				Handle(err)
			}
		}

		if collectionCount > 0 {
			err := deleteByKey(collection)
			Handle(err)
		}
		return nil
	})

	return err
}
