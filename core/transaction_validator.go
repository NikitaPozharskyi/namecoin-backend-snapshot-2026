package core

import "fmt"

type BalanceManager struct {
	state *State
}

func NewBalanceManager(state *State) *BalanceManager {
	if state == nil {
		panic("state is required")
	}
	return &BalanceManager{state: state}
}

func (m *BalanceManager) VerifyOwnership(owner string, publicKey []byte) error {
	if HashHex(publicKey) != owner {
		return fmt.Errorf("public key does not match sender address")
	}
	return nil
}

func (m *BalanceManager) SpendPlan(owner string, amount uint64) ([]TxInput, []TxOutput, error) {
	return m.state.DeterministicSpendPlan(owner, amount)
}

type TransactionValidator struct {
	state   *State
	balance *BalanceManager
}

func NewTransactionValidator(state *State) *TransactionValidator {
	if state == nil {
		panic("state is required")
	}
	return &TransactionValidator{
		state:   state,
		balance: NewBalanceManager(state),
	}
}

func (v *TransactionValidator) ValidateSigned(tx SignedTransaction) error {
	publicKey, err := DecodeHex(tx.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}
	if err := v.balance.VerifyOwnership(tx.From, publicKey); err != nil {
		return err
	}

	expectedInputs, expectedOutputs, err := v.balance.SpendPlan(tx.From, tx.Amount)
	if err != nil {
		return err
	}
	if !EqualInputs(tx.Inputs, expectedInputs) {
		return fmt.Errorf("inputs do not match deterministic spend plan")
	}
	if !EqualOutputs(tx.Outputs, expectedOutputs) {
		return fmt.Errorf("outputs do not match deterministic spend plan")
	}

	unsignedBytes, err := SerializeSignedTransaction(tx)
	if err != nil {
		return err
	}
	if HashHex(unsignedBytes) != tx.TxID {
		return fmt.Errorf("transaction id mismatch")
	}
	if err := VerifySignature(publicKey, unsignedBytes, tx.Signature); err != nil {
		return err
	}

	cmd, err := resolveCommand(tx.Type, tx.Payload)
	if err != nil {
		return err
	}
	return cmd.Validate(v.state, tx)
}

func (v *TransactionValidator) Materialize(tx SignedTransaction) (Transaction, error) {
	if err := v.ValidateSigned(tx); err != nil {
		return Transaction{}, err
	}
	onChain := Transaction{
		From:      tx.From,
		Type:      tx.Type,
		Inputs:    tx.Inputs,
		Outputs:   tx.Outputs,
		Amount:    tx.Amount,
		Payload:   tx.Payload,
		PublicKey: tx.PublicKey,
		TxID:      tx.TxID,
		Signature: tx.Signature,
	}

	cmd, err := resolveCommand(onChain.Type, onChain.Payload)
	if err != nil {
		return Transaction{}, err
	}
	if err := cmd.ValidateWithInputs(v.state, onChain); err != nil {
		return Transaction{}, err
	}
	return onChain, nil
}
