package core

import (
	"bytes"
	"fmt"
)

type ChainManager struct {
	chains         []*Chain
	longestIndex   int
	orphansByPrev  map[string][]*Block
	knownBlockMeta map[string]blockMeta
}

type blockMeta struct {
	height uint64
	chains map[*Chain]struct{}
}

func NewChainManager(chain *Chain) *ChainManager {
	if chain == nil {
		panic("chain is required")
	}
	manager := &ChainManager{
		chains:         []*Chain{chain},
		orphansByPrev:  make(map[string][]*Block),
		knownBlockMeta: make(map[string]blockMeta),
	}
	manager.indexChain(chain)
	return manager
}

func (m *ChainManager) LongestChain() *Chain {
	return m.chains[m.longestIndex]
}

func (m *ChainManager) AppendBlock(block *Block) (bool, error) {
	if block == nil {
		return false, fmt.Errorf("nil block")
	}
	beforeHash := m.LongestChain().HeadHash()
	beforeHeight := m.LongestChain().HeadHeight()
	err := m.processPending(block)
	after := m.LongestChain()
	changed := beforeHeight != after.HeadHeight() || !bytes.Equal(beforeHash, after.HeadHash())
	return changed, err
}

func (m *ChainManager) processPending(initial *Block) error {
	queue := []*Block{initial}
	var firstErr error
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children, err := m.processSingle(current)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		queue = append(queue, children...)
	}
	return firstErr
}

func (m *ChainManager) processSingle(block *Block) ([]*Block, error) {
	key := string(block.ComputeHash())
	if _, ok := m.knownBlockMeta[key]; ok {
		return m.popWaitingChildren(block.Hash), nil
	}

	chain, err := m.chainForParent(block)
	if err != nil {
		if err == errUnknownParent {
			m.storeOrphan(block)
			return nil, nil
		}
		return nil, err
	}
	if err := chain.ApplyBlock(block); err != nil {
		return nil, err
	}
	m.registerBlock(chain, block)
	m.promoteIfLonger(chain)
	return m.popWaitingChildren(block.Hash), nil
}

var errUnknownParent = fmt.Errorf("unknown parent")

func (m *ChainManager) chainForParent(block *Block) (*Chain, error) {
	if block.Header.Height == 0 {
		for _, chain := range m.chains {
			if chain.HeadHash() == nil && chain.HeadHeight() == 0 {
				return chain, nil
			}
		}
		branch := NewChain(newOverlayStore(m.LongestChain().store))
		m.chains = append(m.chains, branch)
		return branch, nil
	}

	parentHash := block.Header.PrevHash
	if len(parentHash) == 0 {
		return nil, errUnknownParent
	}
	for _, chain := range m.chains {
		if bytes.Equal(chain.HeadHash(), parentHash) {
			return chain, nil
		}
	}

	meta, ok := m.knownBlockMeta[string(parentHash)]
	if !ok {
		return nil, errUnknownParent
	}
	var base *Chain
	for chain := range meta.chains {
		if base == nil || chain.HeadHeight() > base.HeadHeight() {
			base = chain
		}
	}
	if base == nil {
		return nil, errUnknownParent
	}

	branch, err := base.forkUpToHeight(meta.height, newOverlayStore(base.store))
	if err != nil {
		return nil, err
	}
	m.chains = append(m.chains, branch)
	m.indexChain(branch)
	return branch, nil
}

func (m *ChainManager) promoteIfLonger(candidate *Chain) {
	current := m.LongestChain()
	if candidate.HeadHeight() < current.HeadHeight() {
		return
	}
	if candidate.HeadHeight() == current.HeadHeight() && bytes.Compare(candidate.HeadHash(), current.HeadHash()) >= 0 {
		return
	}
	if overlay, ok := candidate.store.(*overlayStore); ok {
		overlay.Commit()
		candidate.store = overlay.base
	}
	for index, chain := range m.chains {
		if chain == candidate {
			m.longestIndex = index
			return
		}
	}
}

func (m *ChainManager) storeOrphan(block *Block) {
	if block.Header.Height == 0 || len(block.Header.PrevHash) == 0 {
		return
	}
	parentKey := string(block.Header.PrevHash)
	copyBlock := *block
	copyBlock.Hash = CloneBytes(block.Hash)
	copyBlock.Header.PrevHash = CloneBytes(block.Header.PrevHash)
	m.orphansByPrev[parentKey] = append(m.orphansByPrev[parentKey], &copyBlock)
}

func (m *ChainManager) popWaitingChildren(parentHash []byte) []*Block {
	waiting := m.orphansByPrev[string(parentHash)]
	delete(m.orphansByPrev, string(parentHash))
	return waiting
}

func (m *ChainManager) registerBlock(chain *Chain, block *Block) {
	key := string(block.ComputeHash())
	meta := m.knownBlockMeta[key]
	if meta.chains == nil {
		meta.chains = make(map[*Chain]struct{})
	}
	meta.height = block.Header.Height
	meta.chains[chain] = struct{}{}
	m.knownBlockMeta[key] = meta
}

func (m *ChainManager) indexChain(chain *Chain) {
	for height := uint64(0); height <= chain.HeadHeight(); height++ {
		block, err := chain.blockAtHeight(height)
		if err != nil {
			continue
		}
		m.registerBlock(chain, block)
	}
}
