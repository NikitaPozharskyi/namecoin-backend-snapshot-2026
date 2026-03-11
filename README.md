# NameCoin Portfolio Snapshot

This repository is a sanitized, public-facing snapshot of my contribution to an EPFL decentralized systems group project. In addition to implementing major backend pieces of the NameCoin subsystem, I was responsible for shaping the project architecture and its layered design, helping structure responsibilities across transaction validation, state management, chain application, and fork resolution.

The original private coursework repository mixed together course skeleton code, homework solutions, teammate-owned modules, frontend experiments, grading assets, and the final team project. For portfolio publication, I rebuilt the most representative backend slice into a smaller standalone Go package under `core/` and kept the architecture notes and Mermaid diagrams under `docs/`.

## What This Snapshot Shows

- The project architecture and layered backend design I helped define for the group project
- A commit-reveal domain registration flow inspired by NameCoin
- Canonical transaction hashing and signature verification
- Deterministic UTXO selection and change generation
- Stateful application of `NameNew`, `NameFirstUpdate`, and `NameUpdate`
- Fork-aware chain management with orphan buffering and longest-chain promotion

## My Contribution Focus

- Defined and documented the backend architecture and layering of the project
- Implemented key backend logic for transaction validation and state transitions
- Worked on deterministic spend planning and UTXO-based transaction materialization
- Built and refined longer-chain resolution and fork-handling logic
- Produced architecture diagrams to explain the system design and data flow

## What It Intentionally Omits

- Course skeleton networking, transport, registry, gossip, and Paxos layers
- Homework `HW0`-`HW3` tests and teacher-provided scaffolding
- GUI, HTTP proxy, grading workflows, generated binaries, PDFs, and screenshots not needed for portfolio context
- Teammate-authored modules that I could not safely claim as primarily mine

## Design Choices In This Public Version

- Constructor-enforced invariants replace the inherited pattern of repeatedly nil-checking internal service state at runtime.
- The exported code is intentionally smaller than the private repo. It is meant to be readable and attributable, not to mirror every private integration detail.
- Historical NameCoin terms such as `NameNew` and `NameFirstUpdate` are preserved because they are part of the protocol semantics, while the package and repository structure were renamed to feel like a standalone project rather than a coursework dump.

## Run The Tests

```bash
go test ./...
```

## Files Worth Reading First

- `core/transaction_validator.go`
- `core/state.go`
- `core/chain.go`
- `core/chain_manager.go`
- `docs/architecture.md`
- `CONTRIBUTIONS_AND_SANITIZATION.md`
