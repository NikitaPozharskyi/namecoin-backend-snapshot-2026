package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type UTXO struct {
	TxID   string
	To     string
	Amount uint64
	Order  uint64
}

type TxInput struct {
	TxID  string `json:"txid"`
	Index uint32 `json:"index"`
}

type TxOutput struct {
	To     string `json:"to"`
	Amount uint64 `json:"amount"`
}

type Transaction struct {
	From      string          `json:"from"`
	Type      string          `json:"type"`
	Inputs    []TxInput       `json:"inputs"`
	Outputs   []TxOutput      `json:"outputs"`
	Amount    uint64          `json:"amount"`
	Payload   json.RawMessage `json:"payload"`
	PublicKey string          `json:"pk,omitempty"`
	TxID      string          `json:"txid,omitempty"`
	Signature string          `json:"signature,omitempty"`
}

type SignedTransaction struct {
	Type      string          `json:"type"`
	From      string          `json:"from"`
	Amount    uint64          `json:"amount"`
	Payload   json.RawMessage `json:"payload"`
	Inputs    []TxInput       `json:"inputs"`
	Outputs   []TxOutput      `json:"outputs"`
	PublicKey string          `json:"pk"`
	TxID      string          `json:"txId"`
	Signature string          `json:"signature"`
}

type BlockHeader struct {
	Height    uint64 `json:"height"`
	PrevHash  []byte `json:"prevHash"`
	TxRoot    []byte `json:"txRoot"`
	Timestamp int64  `json:"timestamp"`
	Nonce     uint64 `json:"nonce"`
	Miner     string `json:"miner"`
}

type Block struct {
	Header       BlockHeader   `json:"header"`
	Transactions []Transaction `json:"transactions"`
	Hash         []byte        `json:"hash"`
}

type NameRecord struct {
	Owner     string
	Domain    string
	IP        string
	Salt      string
	ExpiresAt uint64
}

type CommitmentRecord struct {
	Commitment string
	TTL        uint64
	Height     uint64
}

type NameNew struct {
	Commitment string `json:"commitment"`
	TTL        uint64 `json:"ttl,omitempty"`
}

type NameFirstUpdate struct {
	Domain string `json:"domain"`
	Salt   string `json:"salt"`
	IP     string `json:"ip"`
	TTL    uint64 `json:"ttl,omitempty"`
	TxID   string `json:"txid"`
}

type NameUpdate struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
	TTL    uint64 `json:"ttl,omitempty"`
}

func (b *Block) ComputeHash() []byte {
	headerBytes, _ := json.Marshal(b.Header)
	sum := sha256.Sum256(headerBytes)
	return sum[:]
}

func CloneBytes(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	return append([]byte(nil), src...)
}

func OutpointKey(txID string, index uint32) string {
	return fmt.Sprintf("%s:%d", txID, index)
}
