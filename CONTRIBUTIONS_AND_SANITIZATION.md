# Contributions And Sanitization Notes

## Verified Contribution Areas

Using the private repository history for `npozharskyi <cactyr@gmail.com>`, the clearest personally authored areas were:

- Transaction service and wallet/balance logic introduced on November 25, 2025
- Backend integration and code-structure cleanup on December 1-2, 2025
- Longer-chain resolution and its tests on December 12-13, 2025
- Architecture diagrams added on December 18, 2025
- Deterministic integration-test hardening on December 19, 2025

These commits heavily touched:

- `peer/impl/transaction_service.go`
- `peer/impl/token_balance_manager.go`
- `peer/impl/namecoin_chain_service.go`
- `peer/impl/namecoin_chain.go`
- `peer/impl/namecoin_state*.go`
- `peer/tests/unit/namecoin_chain_service_test.go`
- `Diagrams/*.mmd`

## Sanitization Strategy

I did not publish the original repository as-is. Instead, I created a new local snapshot that:

- preserves the implementation ideas and substantial logic from the contribution areas above
- rewrites the code into a smaller standalone package layout
- removes course-specific naming at the repository level
- strips out assets that were clearly coursework infrastructure rather than portfolio material

## What Was Kept

- The NameCoin commit-reveal transaction model
- Deterministic spend-plan logic
- Signature and transaction-ID verification
- Stateful domain and UTXO application
- Longest-chain and orphan-handling logic
- Mermaid architecture diagrams and rewritten design notes

## What Was Removed

- Teacher-provided skeleton commits and generic homework framework surface
- `HW0`-`HW3` unit/integration/perf tests
- GUI and HTTP proxy code
- GitHub workflows and grading-oriented automation
- PDFs, screenshots, generated diagram exports, and untracked coursework artifacts
- Frontend application code and teammate-owned integration layers

## What Was Rewritten

- The public code under `core/` is a contribution-oriented rewrite, not a blind copy of the private tree.
- Internal APIs were renamed and compressed so the public repo highlights the interesting backend logic instead of the whole course scaffold.
- Documentation was fully rewritten for portfolio readers instead of course staff or teammates.

## Defensive Programming Refactor

One inherited pattern in the private repo was repeated method-level validation of mandatory internal object state, especially on the large `node` type where many methods called a shared `validateNode(...)` helper before doing normal work. That style is reasonable for a defensive coursework codebase, but noisy in a public portfolio snapshot.

In this sanitized version, the core services use constructor-enforced invariants instead:

- `NewState`
- `NewBalanceManager`
- `NewTransactionValidator`
- `NewChain`
- `NewChainManager`

Those constructors panic on invalid setup, so internal methods can focus on domain logic rather than re-checking required collaborators on every call.

## Remaining Risk To Review Before Publishing

- The public snapshot still reflects a team project domain and protocol, so you should avoid wording that suggests sole authorship of the full original system.
- The Mermaid diagrams are derived from your own commits, but you should still verify that no teammate-only detail feels too specific to unpublished coursework deliverables.
- If you want a very conservative public repo, you can remove `docs/diagrams/` and keep only the prose architecture note.
