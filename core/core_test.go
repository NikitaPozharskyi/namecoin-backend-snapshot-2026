package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestTransactionValidatorRejectsSignatureMismatch(t *testing.T) {
	state := NewState()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	owner := HashHex(publicKey)
	state.EnsureAccount(owner)
	if err := state.AppendUTXO(UTXO{TxID: "funds-1", To: owner, Amount: 2}); err != nil {
		t.Fatal(err)
	}

	validator := NewTransactionValidator(state)
	tx := mustSignedTransaction(t, state, owner, publicKey, privateKey, NameNewCommandName, 2, NameNew{Commitment: "commitment-1"})
	tx.Signature = "deadbeef"

	if err := validator.ValidateSigned(tx); err == nil {
		t.Fatal("expected signature verification to fail")
	}
}

func TestTransactionValidatorChecksRevealCommitmentAgainstMaterializedInputs(t *testing.T) {
	state := NewState()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	owner := HashHex(publicKey)
	state.EnsureAccount(owner)
	if err := state.AppendUTXO(UTXO{TxID: "commit-funds", To: owner, Amount: 3}); err != nil {
		t.Fatal(err)
	}

	reveal := NameFirstUpdate{Domain: "portfolio.test", Salt: "salt-1", IP: "192.0.2.10", TxID: "seed-commit"}
	state.SetCommitment(OutpointKey("seed-commit", 0), CommitmentRecord{Commitment: HashString("DOMAIN_HASH_v1:portfolio.test:salt-1"), TTL: 10, Height: 0})

	validator := NewTransactionValidator(state)
	tx := mustSignedTransaction(t, state, owner, publicKey, privateKey, NameFirstUpdateCommandName, 3, reveal)
	if _, err := validator.Materialize(tx); err != nil {
		t.Fatalf("expected reveal to materialize, got %v", err)
	}

	state.SetCommitment(OutpointKey("seed-commit", 0), CommitmentRecord{Commitment: HashString("DOMAIN_HASH_v1:portfolio.test:wrong"), TTL: 10, Height: 0})
	if _, err := validator.Materialize(tx); err == nil {
		t.Fatal("expected reveal validation to fail after commitment mismatch")
	}
}

func TestDeterministicSpendPlanUsesStableOrderAndReturnsChange(t *testing.T) {
	state := NewState()
	state.EnsureAccount("alice")
	_ = state.AppendUTXO(UTXO{TxID: "coin-2", To: "alice", Amount: 30})
	_ = state.AppendUTXO(UTXO{TxID: "coin-1", To: "alice", Amount: 20})

	inputs, outputs, err := state.DeterministicSpendPlan("alice", 35)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 || inputs[0].TxID != "coin-2" || inputs[1].TxID != "coin-1" {
		t.Fatalf("unexpected inputs: %+v", inputs)
	}
	if len(outputs) != 1 || outputs[0].Amount != 15 || outputs[0].To != "alice" {
		t.Fatalf("unexpected outputs: %+v", outputs)
	}
}

func TestChainManagerPromotesLongerForkAndDrainsOrphans(t *testing.T) {
	store := NewMemoryStore()
	chain := NewChain(store)

	genesis := mustBlock(t, 0, nil, nil)
	if err := chain.ApplyBlock(genesis); err != nil {
		t.Fatal(err)
	}
	mainBlock := mustBlock(t, 1, genesis.Hash, nil)
	if err := chain.ApplyBlock(mainBlock); err != nil {
		t.Fatal(err)
	}

	manager := NewChainManager(chain)

	forkOne := mustBlock(t, 1, genesis.Hash, nil)
	forkTwo := mustBlock(t, 2, forkOne.Hash, nil)

	changed, err := manager.AppendBlock(forkTwo)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("child without parent should remain orphaned")
	}

	changed, err = manager.AppendBlock(forkOne)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected longer fork to become canonical after parent arrives")
	}
	if manager.LongestChain().HeadHeight() != 2 {
		t.Fatalf("expected height 2, got %d", manager.LongestChain().HeadHeight())
	}
}

func mustSignedTransaction(t *testing.T, state *State, owner string, publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, txType string, amount uint64, payload any) SignedTransaction {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	inputs, outputs, err := state.DeterministicSpendPlan(owner, amount)
	if err != nil {
		t.Fatal(err)
	}

	tx := SignedTransaction{Type: txType, From: owner, Amount: amount, Payload: payloadBytes, Inputs: inputs, Outputs: outputs, PublicKey: hex.EncodeToString(publicKey)}
	unsignedBytes, err := SerializeSignedTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}
	tx.TxID = HashHex(unsignedBytes)
	tx.Signature = hex.EncodeToString(ed25519.Sign(privateKey, Hash(unsignedBytes)))
	return tx
}

func mustBlock(t *testing.T, height uint64, prevHash []byte, txs []Transaction) *Block {
	t.Helper()
	root, err := ComputeTxRoot(txs)
	if err != nil {
		t.Fatal(err)
	}
	block := &Block{Header: BlockHeader{Height: height, PrevHash: CloneBytes(prevHash), TxRoot: root}, Transactions: txs}
	block.Hash = block.ComputeHash()
	return block
}
