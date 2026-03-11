# Architecture Overview

This public snapshot keeps the backend slice of the original project that best represents my work: validating signed NameCoin transactions, maintaining state, and handling fork resolution in a proof-of-work chain.

## Components

- `TransactionValidator`: checks address ownership, canonical inputs and outputs, transaction ID, signature, and command-specific payload rules.
- `State`: stores domains, commitments, claimed names, UTXOs, and applied transactions.
- `Chain`: validates block linkage and transaction roots, then applies blocks through a cloned state snapshot.
- `ChainManager`: tracks competing branches, buffers orphan blocks, and promotes a longer fork to canonical status.

## Commit-Reveal Flow

1. `NameNew` anchors a salted commitment without revealing the target domain.
2. `NameFirstUpdate` reveals the domain, salt, IP, and original commitment reference.
3. `NameUpdate` lets the owner refresh the IP binding and TTL.

## Why This Slice Matters

The original private repository contained far more infrastructure than I wanted to publish safely. This slice captures the parts where my contribution was strongest and most implementation-heavy:

- stateful backend wiring
- transaction validation logic
- deterministic spend planning
- longer-chain resolution
- architecture modeling
