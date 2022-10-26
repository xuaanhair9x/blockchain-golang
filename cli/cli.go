package cli

import (
	"flag"
    "fmt"
	"os"
	"log"
	"strconv"
	"runtime"
	"github.com/hai-nguyen/golang-blockchain/wallet"
	"github.com/hai-nguyen/golang-blockchain/blockchain"
	"github.com/hai-nguyen/golang-blockchain/network"
)

type CommandLine struct{}

func (cli *CommandLine) StartNode(address, nodeID string)  {
	network.StartServer(nodeID, address)
}

func (cli *CommandLine) Reindex(nodeID string) {

	chain := blockchain.ContinueBlockchain(nodeID)
	defer chain.Db.Close()

	chain.Reindex()

	fmt.Println("Reindex process done .....")
}

func (cli *CommandLine) GetBalance(address, nodeID string) {

	if !wallet.ValidateAddress([]byte(address)) {
		log.Panic("Addres From is invalid.")
	}

	chain := blockchain.ContinueBlockchain(nodeID)
	defer chain.Db.Close()
	
	ws, err := wallet.CreateWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	
	wlFrom := ws.GetWallet(address)

	utxos := chain.FindUnspentTransactions(wlFrom.HashPublicKey())
	
	balance := 0
	if utxos != nil {
		for _, out := range utxos.Outputs {
			balance = balance + out.Value
		}
	}

	fmt.Printf("The balance of address %s is: %d", address, balance)
	fmt.Println()
}

func (cli *CommandLine) CreateBlockChain(address, nodeID string)  {

	// Validate the address
	if !wallet.ValidateAddress([]byte(address)) {
		log.Panic("Addres is invalid.")
	}
	
	ws, err := wallet.CreateWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}

	wlFrom := ws.GetWallet(address)

	chain := blockchain.InitBlockchain(wlFrom.HashPublicKey(), nodeID)
	defer chain.Db.Close()

}

/**

- Valide 2 addresses
- Get block chain from DB
- Load 2 wallet for 2 addresses
- Create a tx for sending coin
- Mine block
**/
func (cli *CommandLine) Send(from, to string, amount int, miner, NODE_ID string)  {

	if !wallet.ValidateAddress([]byte(from)) {
		log.Panic("Addres From is invalid.")
	}

	// if !wallet.ValidateAddress([]byte(to)) {
	// 	log.Panic("Addres To is invalid.")
	// }
	
	chain := blockchain.ContinueBlockchain(NODE_ID)
	defer chain.Db.Close()
	
	ws, err := wallet.CreateWallets(NODE_ID)
	
	if err != nil {
		log.Panic(err)
	}
	
	wlFrom := ws.GetWallet(from)

	tx, err := blockchain.NewTransaction(wlFrom, []byte(to), amount, chain)

	if err != nil {
		log.Panic(err)
	}
	
	if (len(miner) > 0) {
		
		txCb := blockchain.CoinBaseTx("", wlFrom.HashPublicKey())
		
		block := chain.MineBlock([]*blockchain.Transaction{txCb, tx})
		
		err = chain.Update(block)	
		
		if err != nil {
			log.Panic(err)
		}

		fmt.Println("Create transaction sending success.")
		fmt.Printf("New transaction %x", tx.Id )
		fmt.Println()
		fmt.Println("New block was created: %x", block.Hash)
	} else {
		network.SendTx(network.KnownNodes[0], tx)
	}
	


}

func (cli *CommandLine) PrintChain(NODE_ID string) {
	chain := blockchain.ContinueBlockchain(NODE_ID)
	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Prev. hash: %x\n", block.PreviousHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		for _, transaction := range block.Transactions {
			fmt.Printf("TransactionID: %x\n", string(transaction.Id))
			for _, input := range transaction.Inputs {
				fmt.Printf("TransactionInputID: %x\n", input.Id)
				fmt.Printf("TransactionInputOut: %d\n", input.Out)
				fmt.Printf("TransactionInputSig: %x\n", input.Signiture)
			}
			for _, output := range transaction.Outputs {
				fmt.Printf("TransactionOutputValue: %d\n", output.Value)
				fmt.Printf("TransactionOutputPubKey: %x\n", output.PubKeyHash)
			}
		}
		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PreviousHash) == 0 {
			break
		}
	}

}

func (cli *CommandLine) PrintListAddresse(NODE_ID string)  {

	ws, err := wallet.CreateWallets(NODE_ID) // Return wallets
	
	if err != nil {
		log.Panic(err)
	}

	listAddress := ws.GetAddresses()

	for _, address := range listAddress {
		fmt.Println(address)
	}
	
}

func (cli *CommandLine) CreateWallet(NodeID string)  {

	wl := wallet.CreateWallet() // Return a wallet
	ws, err := wallet.CreateWallets(NodeID) // Return wallets
	
	if err != nil {
		log.Panic(err)
	}
	
	address := ws.AddWallet(wl)
	
	ws.SaveFile(NodeID)
	fmt.Printf("New address is: %s\n", address)

}

func ( cli *CommandLine) Run()  {

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env is not set!")
		runtime.Goexit()
	}

	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	printListAddressesCmd := flag.NewFlagSet("printlistaddresses", flag.ExitOnError )
	createBlockChainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError )
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	reIndexCmd := flag.NewFlagSet("reindex", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	createBlockchainAddress := createBlockChainCmd.String("address", "", "The address to send genesis block reward to")
	fromAddressSending := sendCmd.String("from","", "the address to get the coin for sending")
	toAddressSending := sendCmd.String("to","", "the address to send the coin")
	minerSending := sendCmd.String("miner", "", "Miner sending")
	amountSending := sendCmd.Int("amount",0,"the amount for sending")
	addressBalance := getBalanceCmd.String("address", "", "The address to show balance")
	startNodeMiner := startNodeCmd.String("address","","The address to mine")

	switch os.Args[1] {
    case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "reindex":
		err := reIndexCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "printlistaddresses":
		err := printListAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "createblockchain":
		err := createBlockChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}

    default:
        flag.PrintDefaults()
        os.Exit(1)
    }

	if sendCmd.Parsed() {
		cli.Send(*fromAddressSending, *toAddressSending, *amountSending, *minerSending, nodeID)
	}

	if createWalletCmd.Parsed() {
		cli.CreateWallet(nodeID)
	}

	if printListAddressesCmd.Parsed() {
		cli.PrintListAddresse(nodeID)
	}

	if createBlockChainCmd.Parsed() {
		cli.CreateBlockChain(*createBlockchainAddress, nodeID)
	}

	if getBalanceCmd.Parsed() {
		cli.GetBalance(*addressBalance, nodeID)
	}

	if printChainCmd.Parsed() {
		cli.PrintChain(nodeID)
	}

	if reIndexCmd.Parsed() {
		cli.Reindex(nodeID)
	}

	if startNodeCmd.Parsed() {
		cli.StartNode(*startNodeMiner, nodeID)
	}
}