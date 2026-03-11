package core

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	blockKeyPrefix = "block:"
	lastBlockKey   = "block:last"
)

type Chain struct {
	store      Store
	state      *State
	headHash   []byte
	headHeight uint64
}

func NewChain(store Store) *Chain {
	if store == nil {
		panic("store is required")
	}
	return &Chain{
		store: store,
		state: NewState(),
	}
}

func (c *Chain) HeadHash() []byte {
	return CloneBytes(c.headHash)
}

func (c *Chain) HeadHeight() uint64 {
	return c.headHeight
}

func (c *Chain) SnapshotDomains() map[string]NameRecord {
	return c.state.SnapshotDomains()
}

func (c *Chain) ApplyBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("nil block")
	}
	working := c.state.Clone()
	if err := validateBlockHeader(c.headHeight, c.headHash, *block); err != nil {
		return err
	}
	if err := working.ApplyBlock(*block); err != nil {
		return err
	}

	data, err := json.Marshal(block)
	if err != nil {
		return err
	}
	c.state.Replace(working)
	c.store.Set(encodeBlockKey(block.Header.Height), data)
	c.store.Set(lastBlockKey, block.ComputeHash())
	c.headHash = block.ComputeHash()
	c.headHeight = block.Header.Height
	return nil
}

func (c *Chain) forkUpToHeight(height uint64, override Store) (*Chain, error) {
	if height > c.headHeight {
		return nil, fmt.Errorf("cannot fork above head")
	}
	store := c.store
	if override != nil {
		store = override
	}
	fork := NewChain(store)
	for current := uint64(0); current <= height; current++ {
		block, err := c.blockAtHeight(current)
		if err != nil {
			return nil, err
		}
		if err := fork.ApplyBlock(block); err != nil {
			return nil, err
		}
	}
	return fork, nil
}

func (c *Chain) blockAtHeight(height uint64) (*Block, error) {
	raw := c.store.Get(encodeBlockKey(height))
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing block at height %d", height)
	}
	var block Block
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, err
	}
	return &block, nil
}

func ComputeTxRoot(transactions []Transaction) ([]byte, error) {
	hashed := make([]byte, 0)
	for _, tx := range transactions {
		data, err := SerializeTransactionForRoot(tx)
		if err != nil {
			return nil, err
		}
		hashed = append(hashed, Hash(data)...)
	}
	return Hash(hashed), nil
}

func validateBlockHeader(currentHeight uint64, currentHash []byte, block Block) error {
	computedRoot, err := ComputeTxRoot(block.Transactions)
	if err != nil {
		return err
	}
	if !bytes.Equal(block.Header.TxRoot, computedRoot) {
		return fmt.Errorf("transaction root mismatch")
	}
	computedHash := block.ComputeHash()
	if len(block.Hash) != 0 && !bytes.Equal(block.Hash, computedHash) {
		return fmt.Errorf("block hash mismatch")
	}

	if currentHash == nil {
		if block.Header.Height != 0 {
			return fmt.Errorf("invalid genesis height")
		}
		if len(block.Header.PrevHash) != 0 {
			return fmt.Errorf("genesis block must not have a parent")
		}
		return nil
	}

	if block.Header.Height != currentHeight+1 {
		return fmt.Errorf("unexpected block height")
	}
	if !bytes.Equal(block.Header.PrevHash, currentHash) {
		return fmt.Errorf("prev hash mismatch")
	}
	return nil
}

func encodeBlockKey(height uint64) string {
	return fmt.Sprintf("%s%020d", blockKeyPrefix, height)
}
