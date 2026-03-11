package core

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"sync"
)

const (
	DefaultDomainTTLBlocks uint64 = 36_000
	MaxDomainTTLBlocks     uint64 = 5_256_000
	MaxFirstUpdateDepth    uint64 = 25
)

type State struct {
	mu             sync.RWMutex
	domains        map[string]NameRecord
	commitments    map[string]CommitmentRecord
	claimedDomains map[string]struct{}
	expiresAt      map[uint64][]string
	utxosByOwner   map[string]map[string]UTXO
	appliedTxs     map[string]struct{}
	currentHeight  uint64
	defaultTTL     uint64
	nextUTXOOrder  uint64
}

func NewState() *State {
	return &State{
		domains:        make(map[string]NameRecord),
		commitments:    make(map[string]CommitmentRecord),
		claimedDomains: make(map[string]struct{}),
		expiresAt:      make(map[uint64][]string),
		utxosByOwner:   make(map[string]map[string]UTXO),
		appliedTxs:     make(map[string]struct{}),
		defaultTTL:     clampTTL(DefaultDomainTTLBlocks),
	}
}

func (s *State) Clone() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &State{
		domains:        maps.Clone(s.domains),
		commitments:    maps.Clone(s.commitments),
		claimedDomains: maps.Clone(s.claimedDomains),
		expiresAt:      make(map[uint64][]string, len(s.expiresAt)),
		utxosByOwner:   make(map[string]map[string]UTXO, len(s.utxosByOwner)),
		appliedTxs:     maps.Clone(s.appliedTxs),
		currentHeight:  s.currentHeight,
		defaultTTL:     s.defaultTTL,
		nextUTXOOrder:  s.nextUTXOOrder,
	}

	for height, names := range s.expiresAt {
		clone.expiresAt[height] = slices.Clone(names)
	}
	for owner, utxos := range s.utxosByOwner {
		clone.utxosByOwner[owner] = maps.Clone(utxos)
	}
	return clone
}

func (s *State) Replace(snapshot *State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.domains = snapshot.domains
	s.commitments = snapshot.commitments
	s.claimedDomains = snapshot.claimedDomains
	s.expiresAt = snapshot.expiresAt
	s.utxosByOwner = snapshot.utxosByOwner
	s.appliedTxs = snapshot.appliedTxs
	s.currentHeight = snapshot.currentHeight
	s.defaultTTL = snapshot.defaultTTL
	s.nextUTXOOrder = snapshot.nextUTXOOrder
}

func (s *State) CurrentHeight() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentHeight
}

func (s *State) SetHeight(height uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentHeight = height
}

func (s *State) SnapshotDomains() map[string]NameRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.domains)
}

func (s *State) NameLookup(domain string) (NameRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.domains[domain]
	return record, ok
}

func (s *State) DomainExists(domain string) bool {
	_, ok := s.NameLookup(domain)
	return ok
}

func (s *State) IsClaimed(domain string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.claimedDomains[domain]
	return ok
}

func (s *State) IsExpired(record NameRecord, height uint64) bool {
	return record.ExpiresAt != 0 && record.ExpiresAt <= height
}

func (s *State) SetDomain(record NameRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.domains[record.Domain]; ok && current.ExpiresAt != 0 {
		s.removeExpiryLocked(record.Domain, current.ExpiresAt)
	}
	s.domains[record.Domain] = record
	s.claimedDomains[record.Domain] = struct{}{}
	if record.ExpiresAt != 0 {
		s.expiresAt[record.ExpiresAt] = append(s.expiresAt[record.ExpiresAt], record.Domain)
	}
}

func (s *State) SetCommitment(outpoint string, record CommitmentRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commitments[outpoint] = record
}

func (s *State) GetCommitment(outpoint string) (CommitmentRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.commitments[outpoint]
	return record, ok
}

func (s *State) DeleteCommitment(outpoint string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.commitments, outpoint)
}

func (s *State) MarkApplied(txID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appliedTxs[txID] = struct{}{}
}

func (s *State) IsApplied(txID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.appliedTxs[txID]
	return ok
}

func (s *State) EnsureAccount(owner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.utxosByOwner[owner]; !ok {
		s.utxosByOwner[owner] = make(map[string]UTXO)
	}
}

func (s *State) AppendUTXO(utxo UTXO) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.utxosByOwner[utxo.To]; !ok {
		s.utxosByOwner[utxo.To] = make(map[string]UTXO)
	}
	if _, exists := s.utxosByOwner[utxo.To][utxo.TxID]; exists {
		return fmt.Errorf("utxo already exists: %s", utxo.TxID)
	}
	utxo.Order = s.nextUTXOOrder
	s.nextUTXOOrder++
	s.utxosByOwner[utxo.To][utxo.TxID] = utxo
	return nil
}

func (s *State) BurnUTXOs(owner string, inputs []TxInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userUTXOs, ok := s.utxosByOwner[owner]
	if !ok {
		return fmt.Errorf("no utxos for %s", owner)
	}
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if _, dup := seen[input.TxID]; dup {
			return fmt.Errorf("duplicate utxo input: %s", input.TxID)
		}
		if _, exists := userUTXOs[input.TxID]; !exists {
			return fmt.Errorf("missing utxo: %s", input.TxID)
		}
		seen[input.TxID] = struct{}{}
	}
	for txID := range seen {
		delete(userUTXOs, txID)
	}
	return nil
}

func (s *State) DeterministicSpendPlan(owner string, amount uint64) ([]TxInput, []TxOutput, error) {
	if amount == 0 {
		return nil, nil, nil
	}

	s.mu.RLock()
	userUTXOs, ok := s.utxosByOwner[owner]
	if !ok {
		s.mu.RUnlock()
		return nil, nil, fmt.Errorf("no utxos for sender")
	}
	type entry struct {
		txID   string
		amount uint64
		order  uint64
	}
	entries := make([]entry, 0, len(userUTXOs))
	for txID, utxo := range userUTXOs {
		entries = append(entries, entry{txID: txID, amount: utxo.Amount, order: utxo.Order})
	}
	s.mu.RUnlock()

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].order == entries[j].order {
			return entries[i].txID < entries[j].txID
		}
		return entries[i].order < entries[j].order
	})

	var (
		inputs []TxInput
		total  uint64
	)
	for _, entry := range entries {
		inputs = append(inputs, TxInput{TxID: entry.txID, Index: 0})
		total += entry.amount
		if total >= amount {
			break
		}
	}
	if total < amount {
		return nil, nil, fmt.Errorf("insufficient funds")
	}

	var outputs []TxOutput
	if change := total - amount; change > 0 {
		outputs = append(outputs, TxOutput{To: owner, Amount: change})
	}
	return inputs, outputs, nil
}

func (s *State) ApplyBlock(block Block) error {
	working := s.Clone()
	working.SetHeight(block.Header.Height)
	working.PruneExpired(block.Header.Height)
	for i := range block.Transactions {
		tx := block.Transactions[i]
		txID, err := BuildTransactionID(tx)
		if err != nil {
			return err
		}
		if working.IsApplied(txID) {
			continue
		}
		if err := ApplyTransaction(working, txID, tx); err != nil {
			return fmt.Errorf("apply tx %d (%s): %w", i, txID, err)
		}
		working.MarkApplied(txID)
	}
	s.Replace(working)
	return nil
}

func (s *State) PruneExpired(height uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for expiryHeight, names := range s.expiresAt {
		if expiryHeight > height {
			continue
		}
		for _, name := range names {
			record, ok := s.domains[name]
			if ok && record.ExpiresAt == expiryHeight {
				delete(s.domains, name)
				delete(s.claimedDomains, name)
			}
		}
		delete(s.expiresAt, expiryHeight)
	}
}

func (s *State) EffectiveTTL(ttl uint64) uint64 {
	if ttl == 0 {
		return clampTTL(s.defaultTTL)
	}
	return clampTTL(ttl)
}

func (s *State) removeExpiryLocked(domain string, height uint64) {
	if height == 0 {
		return
	}
	names := s.expiresAt[height]
	if len(names) == 0 {
		return
	}
	filtered := names[:0]
	for _, name := range names {
		if name != domain {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		delete(s.expiresAt, height)
		return
	}
	s.expiresAt[height] = filtered
}

func clampTTL(ttl uint64) uint64 {
	if ttl == 0 {
		return 0
	}
	if ttl > MaxDomainTTLBlocks {
		return MaxDomainTTLBlocks
	}
	return ttl
}
