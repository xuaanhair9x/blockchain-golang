package wallet

import (
	"fmt"
	"io/ioutil"
	"encoding/gob"
	"bytes"
	"log"
	"os"
)

const walletFile = "./tmp/wallets_%s.data"

type Wallets struct {
	Wallets map[string]*Wallet
}

func CreateWallets(NodeID string) (*Wallets, error) {
	ws := Wallets{}
	ws.Wallets = make(map[string]*Wallet)
	err := ws.LoadFile(NodeID)

	return &ws, err
}

func (ws *Wallets) AddWallet(wallet *Wallet) string {
	addressStr := string(wallet.GetAddress())
	ws.Wallets[addressStr] = wallet
	return addressStr
}

func (ws *Wallets) GetAddresses() []string{
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) GetWallet(address string)  *Wallet {
	wallet := ws.Wallets[address]
	return wallet
}

func (ws *Wallets) LoadFile(NodeID string) error {
	var wallets Wallets
	path := fmt.Sprintf(walletFile, NodeID)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	
	fileContent, err := ioutil.ReadFile(path) // return []byte

	if err != nil {
        return err
	}

	// func NewReader(b []byte) *Reader 
	newReader := bytes.NewReader(fileContent)

	//func NewDecoder(r io.Reader) *Decoder
	dec := gob.NewDecoder(newReader)

	err = dec.Decode(&wallets)
	
    if err != nil {
        return err
	}
	
	ws.Wallets = wallets.Wallets

	return nil
}

func (ws *Wallets) SaveFile(NodeID string) {

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(ws)

	if err != nil {
        log.Panic(err)
	}
	path := fmt.Sprintf(walletFile, NodeID)
	// func WriteFile(filename string, data []byte, perm fs.FileMode) error
	err = ioutil.WriteFile(path, buf.Bytes(), 0666)

	if err != nil {
		log.Panic(err)
	}
}