1. Genrate address of wallet
    - Hash PublicKey
        + hash1 = Hash256(pubkey)
        + hashPub = ripemd160(hash1)
    - Add version Hash to the first of hash PublicKey
        + versionPubHash = version + hashPub
    - Create checksum
        + length checksum = 4
        + hash1 = Hash256(versionPubHash)
        + hash2 = Hash256(hash1)
        + checksum = hash2[:4] get the first 4 element
    - FullHash = versionPubHash + checksum
    - Address = base58Encode(FullHash)
2. Validate Address
    - FullHash = base58decode
    - checkSum = FullHash[len(FullHash)-4:]
    - versionPubHash = FullHash[:len(FullHash)-4]
    - create checksum from versionPubHash
        + hash1 = Hash256(versionPubHash)
        + hash2 = Hash256(hash1)
        + checksumForCheck = hash2[:4]
    - compare checksumForCheck with checkSum

3. Structure of a block
    - Timestamp : int64
    - hash : byte
    - transactions : []ListTransactions
    - Previous hash : byte
    - None : int
    - Heigh : int

4. Structure of a transaction
    - id : byte
    - inputs : list TxInput
    - outputs : list TxOutput
        + TxInput : 
            ID (id of transaction)
            Out (index of Output of transaction)
            Signiture ()
            PublickeyHash 
        + TxOutput :
            value : amout of coint
            PublickeyHash (Address)

5. Proof of work
    - target bigInt
    - block

6. Tx for Coin base 
    - input {[], -1, nil, random 24 byte}
    - output (100, to)# blockchain-golang
