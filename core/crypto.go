package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	canonicaljson "github.com/gibson042/canonicaljson-go"
)

func Hash(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func HashHex(data []byte) string {
	return hex.EncodeToString(Hash(data))
}

func HashString(value string) string {
	return HashHex([]byte(value))
}

func DecodeHex(value string) ([]byte, error) {
	return hex.DecodeString(value)
}

func VerifySignature(publicKey, unsignedBytes []byte, signatureHex string) error {
	signature, err := DecodeHex(signatureHex)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if !ed25519.Verify(publicKey, Hash(unsignedBytes), signature) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func SerializeSignedTransaction(tx SignedTransaction) ([]byte, error) {
	return canonicaljson.Marshal(map[string]any{
		"type":    tx.Type,
		"from":    tx.From,
		"amount":  tx.Amount,
		"payload": tx.Payload,
		"inputs":  tx.Inputs,
		"outputs": tx.Outputs,
	})
}

func SerializeTransaction(tx Transaction) ([]byte, error) {
	return canonicaljson.Marshal(map[string]any{
		"type":    tx.Type,
		"from":    tx.From,
		"amount":  tx.Amount,
		"payload": tx.Payload,
		"inputs":  tx.Inputs,
		"outputs": tx.Outputs,
	})
}

func SerializeTransactionForRoot(tx Transaction) ([]byte, error) {
	return canonicaljson.Marshal(map[string]any{
		"type":      tx.Type,
		"from":      tx.From,
		"amount":    tx.Amount,
		"payload":   tx.Payload,
		"inputs":    tx.Inputs,
		"outputs":   tx.Outputs,
		"pk":        tx.PublicKey,
		"txid":      tx.TxID,
		"signature": tx.Signature,
	})
}

func BuildTransactionID(tx Transaction) (string, error) {
	data, err := SerializeTransaction(tx)
	if err != nil {
		return "", err
	}
	return HashHex(data), nil
}

func EqualInputs(a, b []TxInput) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func EqualOutputs(a, b []TxOutput) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
