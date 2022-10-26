package network
/**
Example: There are 3 nodes (3000, 4000, 5000)
3000 is core node.
When 4000 mine a block done, 4000 must start to send a block to other nodes.
Mechanism to send a block to others is:
1.Start 4000 -> SendVersion(3000, chain(4000))
- Send version: payload (version=1: default,bestHeight=5, nodeAddr=4000), cmd = version, destination: 3000
2. 3000: HandleVersion:
- compare heigh recieve with heigh of node.
- if heigh of node (4) < heigh receive (5) => SendGetBlock(4000)
- Send set block: payload {nodeAddr=3000}, cmd = getblock, destination: 4000
3. 4000: Handle getBlock: get all block hashes and send back to 3000
- GetBlockHashes
- SendInv: payload {nodeAddr=3000, kind="block", BlockHashes}, cmd inv, destionation: 3000
4. 3000: Handle Inv
- Push all blockhashes into blocksInTransit in order to update all block chain
- Use newest hash block to SendGetData and exclude newest block from blocksInTransit
- SendGetData (nodeAddr=3000,kind="block",hashBlock )
5. 4000: Handel GetData
- Get block by hash => block
- SendBlock: payload {nodeAddr=4000, block}
6. 3000: Handle block
- chain.Addblock(block)
- if len(blocksInTransit) > 0 
	+ BlockHash = blocksInTransit[0] and exclude BlockHash from blocksInTransit
	+ SendGetdata (nodeAddr=3000,kind="block",BlockHash )
- else 
	reindex

1.Start 5000 -> SendVersion(3000, chain(5000))
2. 3000: HandleVersion:
- compare heigh recieve with heigh of node.
- if heigh of node (5) > heigh receive (4) => SendVersion(5000, chain(3000))
- Send version: payload (version=1: default,bestHeight=5, nodeAddr=3000), cmd = version, destination: 5000
3. 5000: HandleVersion:
- compare heigh recieve with heigh of node.
- if heigh of node (4) < heigh receive (5) => SendGetBlock(3000)
- Send set block: payload {nodeAddr=5000}, cmd = getblock, destination: 3000

... like step 3

-------------- send tx ---------------
Function Send() in cli, network.SendTx(rootNode, tx)


**/


import(
	"syscall"
	"github.com/hai-nguyen/golang-blockchain/blockchain"
	"github.com/hai-nguyen/golang-blockchain/wallet"
	"gopkg.in/vrecan/death.v3"
	"os"
	"runtime"
	"fmt"
	"net"
	"bytes"
	"encoding/gob"
	"log"
	"io"
	"io/ioutil"
	"encoding/hex"
)

const (
	version = 1
	commandLength = 12
)

var (
	KnownNodes = []string{"localhost:3000"}
	nodeAddr string
	nodeID string
	blocksInTransit [][]byte
	memoryPool = make(map[string]*blockchain.Transaction)
)

type Version struct {
	NodeAddr string
	Version int
	BestHeight int
} 

type GetData struct {
	NodeAddr string
	Kind string
	Id []byte
}

type GetBlocks struct {
	NodeAddr string
}

type Block struct {
	NodeAddr string
	BlockEnc []byte
}

type Inv struct {
	NodeAddr string
	Kind string
	Items [][]byte
}

type Tx struct {
	NodeAddr string
	TxData []byte
}

/**
- Connect to the server
- Check err if not nil then remove the node to knownode
- Put data to network 
**/
func SendData(addr string, data []byte)  {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		
		fmt.Printf("% is not available.", addr)
		updatedNodes := []string{}
		for _, node := range KnownNodes {
			if (node != addr) {
				updatedNodes = append(updatedNodes, node)
			}
		}
		KnownNodes = updatedNodes
		return
	}
	defer conn.Close()

	// if _, err := conn.Write(data); err != nil {
	// 	log.Fatal(err)
	// }

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func CmdToByte(cmd string) []byte {
	var cmdConvert [commandLength]byte
	
	for index, c := range cmd {
		cmdConvert[index] = byte(c)
	}

	return cmdConvert[:]
}

func ByteToCmd(cmd []byte) string {
	var cmdConvert []byte

	for _, c := range cmd {
		if c != 0x00 {
			cmdConvert = append(cmdConvert, c)
		}
	}
	return fmt.Sprintf("%s",cmdConvert)
}

/**

- The goal of this function is compare height and update nodes 
- payload {version, bestheigh}

**/
func SendVersion(address string, chain *blockchain.Blockchain)  {
	bestHeigh := chain.BestHeight()
	payload := EncodeGob(Version{nodeAddr, version, bestHeigh})
	request := append( CmdToByte("version"), payload ...)
	SendData(address, request)
}

/**
- payload {txdata}
**/
func SendTx(address string, tx *blockchain.Transaction)  {
	payload := EncodeGob(Tx {nodeAddr, tx.Serialize()})
	request := append( CmdToByte("tx"), payload ...)
	SendData(address, request)
}

/**

- payload {kind, id(block or tx)}
**/
func SendGetData(address, kind string, id []byte)  {
	payload := EncodeGob(GetData{nodeAddr, kind, id})
	request := append( CmdToByte("getdata"), payload ...)
	SendData(address, request)
}

// Payload {}
func SendGetBlocks(address string)  {
	payload := EncodeGob(GetBlocks{nodeAddr})
	request := append( CmdToByte("getblocks"), payload ...)
	SendData(address, request)
}

// payload {blockdata}
func SendBlock(address string, block *blockchain.Block)  {
	payload := EncodeGob(Block{nodeAddr, block.SerializeBlock()})
	request := append( CmdToByte("block"), payload ...)
	SendData(address, request)
}

func SendInv(address, kind string, data [][]byte)  {
	payload := EncodeGob(Inv{nodeAddr, kind, data})
	request := append(CmdToByte("inv"), payload ...)
	SendData(address, request)
}

/**
- Recieve tx and put it into memoryPool
- If node address is root node, then sendInv (tx.Id) to other nodes except from node.
	Example: 5000 create a tx, sendtx to 3000 => sendInv to node 4000, 6000 ...
_ Else check memoryPool have 2 nodes at least and node accept mining coin => mini coin
**/
func HandleTx(miner string, chain *blockchain.Blockchain, request []byte)  {
	
	var payload Tx

	newReader := bytes.NewReader(request)
	dec := gob.NewDecoder(newReader)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	tx := blockchain.DeserializeTransaction(payload.TxData)
	txEncId := hex.EncodeToString(tx.Id)

	memoryPool[txEncId] = tx

	if (nodeAddr == KnownNodes[0]) {
		for _, node := range KnownNodes {
			if (node != payload.NodeAddr && node != nodeAddr) {
				fmt.Printf("payload.NodeAddr: %s \n",payload.NodeAddr)
				fmt.Printf("Node send Inv in handle Tx: %s \n", node)
				SendInv(node, "tx", [][]byte{tx.Id})
			}
		}
	}

	if (len(memoryPool) >= 2 && len(miner) > 0) {
		MineTx(miner, chain)
	}
}

/**

case kind = tx:
	- Get tx.Id from payload, Send back SendGetData (txID)
case kind = block:
	- Add all blockhashes to blocksInTransit
	- Send back SendGetData(blocksInTransit[0])
	- exclude blocksInTransit[0] from blocksInTransit
**/
func HandleInv(request []byte)  {

	var payload Inv

	newReader := bytes.NewReader(request)
	dec := gob.NewDecoder(newReader)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if (payload.Kind == "block") {
		blocksInTransit = payload.Items // block hashes
		SendGetData(payload.NodeAddr, "block", payload.Items[0])
	}

	if (payload.Kind == "tx") {
		SendGetData(payload.NodeAddr, "tx", payload.Items[0])
	}
}

/**

case kind = tx:
	- Get txId from payload and get txData from memoryPool
	- Send back SendTx (txData)
case kind = block:
	- Get BlockHash from payload and get blockData from database
	- Send back SendBlock(BlockData)
**/
func HandleGetData(chain *blockchain.Blockchain, request []byte )  {
	var payload GetData

	newReader := bytes.NewReader(request)
	dec := gob.NewDecoder(newReader)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Kind == "block" {
		hashBlock := payload.Id
		block := chain.GetBlockByHash(hashBlock)
		SendBlock(payload.NodeAddr, block)
	}

	if payload.Kind == "tx" {
		txId := payload.Id
		tx := memoryPool[hex.EncodeToString(txId)]
		SendTx(payload.NodeAddr, tx)
	}
}

/**

- Get all block hashes 
- Send back SendInv(blockHashes)
**/
func HandleGetBlocks(chain *blockchain.Blockchain, request []byte )  {

	var payload GetBlocks

	newReader := bytes.NewReader(request)
	dec := gob.NewDecoder(newReader)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	hashes := chain.GetAllHashesBlock()

	SendInv(payload.NodeAddr, "block", hashes)

}

/**

- Add block to chain
- Continute SendGetData if blocksInTransit still contain blockhashes
- if not reindex
**/
func HandleBlock(chain *blockchain.Blockchain, request []byte)  {
	var payload Block

	newReader := bytes.NewReader(request)
	dec := gob.NewDecoder(newReader)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	block := blockchain.DeserializeBlock(payload.BlockEnc)
	chain.AddBlock(block)
	fmt.Sprintf("Block %x was added success \n", block.Hash)

	if (len(blocksInTransit) > 0) {
		SendGetData(payload.NodeAddr, "block", blocksInTransit[0])
		blocksInTransit = blocksInTransit[1:]
	} else {
		chain.Reindex()
	}
}

/**

- Get bestHeight (Height of chain) and otherHeight (Height from other node) 
- To compare
	bestHeight < otherHeight: current chain of its node need to be updated.
		SendGetBlock(AddressFrom)
	bestHeight > otherHeight: chain of other node need to be updated.
		SendVersion(AddressFrom, chain)

- Check if node request doesn't exist in KnownNodes then update KnownNodes

**/
func HandleVersion(chain *blockchain.Blockchain, request []byte)  {
	var payload Version

	newReader := bytes.NewReader(request)
	dec := gob.NewDecoder(newReader)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	bestHeight := chain.BestHeight()
	otherHeight := payload.BestHeight

	if (bestHeight < otherHeight) {
		fmt.Println("bestHeight < otherHeight")
		SendGetBlocks(payload.NodeAddr)
	}

	if (bestHeight > otherHeight) {
		fmt.Println("bestHeight > otherHeight")
		SendVersion(payload.NodeAddr, chain)
	}
	
	checkNodeExist := false

	for _, node := range KnownNodes {
		if (node == payload.NodeAddr) {
			checkNodeExist = true
		}
	}

	if(!checkNodeExist) {
		KnownNodes = append(KnownNodes, payload.NodeAddr)
	}

}

/**

- Get txs from memory pool and verify
- Check len(valid tx), if it > 0 continue, else return
- Create tx coin base for miner
- process miner
- reindex
- remove tx is used for mine from memory pool
- SendInv to other node
- Check len(memory pool), if > 0 => continue mine

**/
func MineTx(miner string, chain *blockchain.Blockchain)  {
	
	var txs []*blockchain.Transaction

	for _, tx := range memoryPool {
		txs = append(txs, tx)
	}
	err := chain.VerifyTransactions(txs)
	
	if err != nil {
		fmt.Println("Transactions are not valid")
		log.Panic(err)
	}

	ws, err := wallet.CreateWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	
	wlMiner := ws.GetWallet(miner)

	txCb := blockchain.CoinBaseTx("", wlMiner.HashPublicKey())
	txs = append(txs, txCb)
	block := chain.MineBlock(txs)
	fmt.Println("Miner: New block was created: %s", block.Hash)

	for _, tx := range txs {
		txIdEnc := hex.EncodeToString(tx.Id)
		delete(memoryPool, txIdEnc)
	} 

	for _, node := range KnownNodes {
		if (node != nodeAddr) {
			SendInv(node, "block", [][]byte{block.Hash})
		}
	}

	if (len(memoryPool) > 0) {
		MineTx(miner, chain)
	}

}

func HandleConnection(conn net.Conn, chain *blockchain.Blockchain, miner string)  {
	req, err := ioutil.ReadAll(conn)
	defer conn.Close()
	
	if err != nil {
		log.Panic(err)
	}

	command := ByteToCmd(req[:commandLength])
	request := req[commandLength:]

	fmt.Printf("Received %s command\n", command)
	
	switch command {
		case "version":
			HandleVersion(chain, request)
		case "getblocks":
			HandleGetBlocks(chain, request)
		case "inv":
			HandleInv(request)
		case "getdata":
			HandleGetData(chain, request)
		case "block":
			HandleBlock(chain, request)
		case "tx":
			HandleTx(miner, chain, request)
		default:
			fmt.Println("Unknown command")
    }
}

/**

- Set node address : localhost:nodeID
- create server  
- Load blockchain by nodeID
- Defer close database
- Handle exit application when dead
- SendVersion if not is not a root
- Check connection accept => handle connection

**/
func StartServer(NodeId, minerAddress string)  {
	nodeAddr = fmt.Sprintf("localhost:%s",NodeId)

	fmt.Println(nodeAddr)

	nodeID = NodeId
	ln, err := net.Listen("tcp", nodeAddr)

	chain := blockchain.ContinueBlockchain(NodeId)
	defer chain.Db.Close()

	go CloseDB(chain)

	if (nodeAddr != KnownNodes[0]) {
		SendVersion(KnownNodes[0], chain)
	}

	if err != nil {
		log.Panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go HandleConnection(conn, chain, minerAddress)
	}
}

func EncodeGob(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func CloseDB(chain *blockchain.Blockchain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Db.Close()
	})
}