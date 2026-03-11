package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	RewardCommandName          = "Reward"
	NameNewCommandName         = "NameNew"
	NameFirstUpdateCommandName = "NameFirstUpdate"
	NameUpdateCommandName      = "NameUpdate"
)

type command interface {
	Validate(*State, SignedTransaction) error
	ValidateWithInputs(*State, Transaction) error
	ApplyUTXO(*State, string, Transaction) error
	ApplyState(*State, Transaction) error
}

func resolveCommand(kind string, payload json.RawMessage) (command, error) {
	switch kind {
	case RewardCommandName:
		return rewardCommand{}, nil
	case NameNewCommandName:
		var cmd NameNew
		return cmd, json.Unmarshal(payload, &cmd)
	case NameFirstUpdateCommandName:
		var cmd NameFirstUpdate
		return cmd, json.Unmarshal(payload, &cmd)
	case NameUpdateCommandName:
		var cmd NameUpdate
		return cmd, json.Unmarshal(payload, &cmd)
	default:
		return nil, fmt.Errorf("unknown command: %s", kind)
	}
}

func ApplyTransaction(state *State, txID string, tx Transaction) error {
	cmd, err := resolveCommand(tx.Type, tx.Payload)
	if err != nil {
		return err
	}
	if err := cmd.ValidateWithInputs(state, tx); err != nil {
		return err
	}
	if err := cmd.ApplyUTXO(state, txID, tx); err != nil {
		return err
	}
	return cmd.ApplyState(state, tx)
}

type rewardCommand struct{}

func (rewardCommand) Validate(*State, SignedTransaction) error { return nil }
func (rewardCommand) ValidateWithInputs(*State, Transaction) error { return nil }
func (rewardCommand) ApplyUTXO(state *State, txID string, tx Transaction) error {
	return applyUTXO(state, txID, tx)
}
func (rewardCommand) ApplyState(*State, Transaction) error { return nil }

func (n NameNew) Validate(_ *State, _ SignedTransaction) error {
	if strings.TrimSpace(n.Commitment) == "" {
		return fmt.Errorf("name_new commitment is required")
	}
	return nil
}

func (n NameNew) ValidateWithInputs(_ *State, tx Transaction) error {
	if len(tx.Inputs) == 0 {
		return fmt.Errorf("name_new requires at least one input")
	}
	return nil
}

func (n NameNew) ApplyUTXO(state *State, txID string, tx Transaction) error {
	if err := applyUTXO(state, txID, tx); err != nil {
		return err
	}
	ttl := n.TTL
	if ttl == 0 {
		ttl = DefaultDomainTTLBlocks
	}
	state.SetCommitment(OutpointKey(txID, 0), CommitmentRecord{
		Commitment: n.Commitment,
		TTL:        ttl,
		Height:     state.CurrentHeight(),
	})
	return nil
}

func (n NameNew) ApplyState(*State, Transaction) error { return nil }

func (n NameFirstUpdate) Validate(state *State, tx SignedTransaction) error {
	if strings.TrimSpace(n.Domain) == "" || strings.TrimSpace(n.Salt) == "" || strings.TrimSpace(n.IP) == "" {
		return fmt.Errorf("invalid name_firstupdate payload")
	}
	if record, ok := state.NameLookup(n.Domain); ok && !state.IsExpired(record, state.CurrentHeight()) {
		return fmt.Errorf("domain already exists")
	}
	if tx.From == "" {
		return fmt.Errorf("missing owner address")
	}
	return nil
}

func (n NameFirstUpdate) ValidateWithInputs(state *State, tx Transaction) error {
	record, _, err := n.resolveCommitment(state, tx)
	if err != nil {
		return err
	}
	if state.CurrentHeight() > record.Height && state.CurrentHeight()-record.Height > MaxFirstUpdateDepth {
		return fmt.Errorf("commitment too old for first update")
	}
	if existing, ok := state.NameLookup(n.Domain); ok && !state.IsExpired(existing, state.CurrentHeight()) {
		return fmt.Errorf("domain already exists")
	}
	return nil
}

func (n NameFirstUpdate) ApplyUTXO(state *State, txID string, tx Transaction) error {
	return applyUTXO(state, txID, tx)
}

func (n NameFirstUpdate) ApplyState(state *State, tx Transaction) error {
	record, outpoint, err := n.resolveCommitment(state, tx)
	if err != nil {
		return err
	}
	if state.DomainExists(n.Domain) || state.IsClaimed(n.Domain) {
		return fmt.Errorf("domain already exists")
	}
	ttl := state.EffectiveTTL(firstNonZero(n.TTL, record.TTL))
	state.SetDomain(NameRecord{
		Owner:     tx.From,
		Domain:    n.Domain,
		IP:        n.IP,
		Salt:      n.Salt,
		ExpiresAt: state.CurrentHeight() + ttl,
	})
	state.DeleteCommitment(outpoint)
	return nil
}

func (n NameFirstUpdate) resolveCommitment(state *State, tx Transaction) (CommitmentRecord, string, error) {
	commitment := HashString(fmt.Sprintf("DOMAIN_HASH_v1:%s:%s", n.Domain, n.Salt))
	commitTxID := n.TxID
	commitIndex := uint32(0)
	if commitTxID == "" {
		if len(tx.Inputs) == 0 {
			return CommitmentRecord{}, "", fmt.Errorf("missing name_new reference")
		}
		commitTxID = tx.Inputs[0].TxID
		commitIndex = tx.Inputs[0].Index
	}
	outpoint := OutpointKey(commitTxID, commitIndex)
	record, ok := state.GetCommitment(outpoint)
	if !ok {
		return CommitmentRecord{}, "", fmt.Errorf("unknown commitment outpoint")
	}
	if record.Commitment != commitment {
		return CommitmentRecord{}, "", fmt.Errorf("commitment mismatch")
	}
	return record, outpoint, nil
}

func (n NameUpdate) Validate(state *State, tx SignedTransaction) error {
	record, ok := state.NameLookup(n.Domain)
	if !ok || state.IsExpired(record, state.CurrentHeight()) {
		return fmt.Errorf("cannot update missing or expired domain")
	}
	if record.Owner != tx.From {
		return fmt.Errorf("cannot update domain you do not own")
	}
	return nil
}

func (n NameUpdate) ValidateWithInputs(*State, Transaction) error { return nil }

func (n NameUpdate) ApplyUTXO(state *State, txID string, tx Transaction) error {
	return applyUTXO(state, txID, tx)
}

func (n NameUpdate) ApplyState(state *State, tx Transaction) error {
	record, ok := state.NameLookup(n.Domain)
	if !ok {
		return fmt.Errorf("missing domain")
	}
	if trimmedIP := strings.TrimSpace(n.IP); trimmedIP != "" {
		record.IP = trimmedIP
	}
	record.Owner = tx.From
	record.ExpiresAt = state.CurrentHeight() + state.EffectiveTTL(n.TTL)
	state.SetDomain(record)
	return nil
}

func applyUTXO(state *State, txID string, tx Transaction) error {
	if len(tx.Inputs) > 0 {
		if err := state.BurnUTXOs(tx.From, tx.Inputs); err != nil {
			return err
		}
	}
	for _, output := range tx.Outputs {
		if err := state.AppendUTXO(UTXO{
			TxID:   txID,
			To:     output.To,
			Amount: output.Amount,
		}); err != nil {
			return err
		}
	}
	return nil
}

func firstNonZero(values ...uint64) uint64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
