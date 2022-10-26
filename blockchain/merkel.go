package blockchain

import (
	"crypto/sha256"
	"log"
)

type MerkelTree struct {
	RootNode *MerkelNode
}

type MerkelNode struct {
	Data []byte
	Right *MerkelNode
	Left *MerkelNode
}

func NewMerkelNode(left, right *MerkelNode, data []byte) *MerkelNode {
	node := MerkelNode{}
	
	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		dataHash := append(left.Data, right.Data ...)
		hash := sha256.Sum256(dataHash)
		node.Data = hash[:]
	}

	node.Right = right
	node.Left = left

	return &node
}

/**

- Create list node from list data and right, left are nil
- Inoder to create a merkel tree we have 2 node at least.
- Total of node is even numbers
- Algorithm
	+ Total node: 1 2 3 4 5 6 7 8 9
	+ step 1: add a node to satisfy the total of node is even numbers
	+ nodes: 1 2 3 4 5 6 7 8 9 9
	+ step 2: Create a merkel node from 2 adjacent nodes.
	+ nodes: 12 34 56 78  99
	+ total of node is still greater then 1, go back to step 1.
	+ step 1: 12 34 56 78 99 99
	+ step 2: 1234 5678 9999
	+ step 1 : 1234 5678 9999 9999
	+ step 2: 12345678 99999999
	+ step 2: 1234567899999999 => Break 
**/
func NewMerkelTree(datas [][]byte) *MerkelTree {
	var nodes []MerkelNode

	for _, data := range datas {
		node := NewMerkelNode(nil, nil, data)
		nodes = append(nodes, *node)
	}

	if len(nodes) == 0 {
		log.Panic("No merkel nodes")
	}
	
	for (len(nodes) > 1) {
		if len(nodes) % 2 == 1 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}
		
		var level []MerkelNode
		for i := 0; i < len(nodes); i = i+2 {
			node := NewMerkelNode(&nodes[i], &nodes[i+1], []byte{})
			level = append(level, *node)
		}

		nodes = level
	}

	return &MerkelTree{&nodes[0]}
}