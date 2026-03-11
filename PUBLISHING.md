# Publishing Instructions

## Before You Push

Review the snapshot one more time with these questions:

- Does the README accurately describe this as a sanitized portfolio snapshot rather than the full original class repo?
- Are you comfortable claiming the code in `core/` as your contribution-oriented rewrite?
- Is there any teammate-specific or teacher-specific material you still want to remove?

## Suggested Public Repo Setup

Create a brand new repository on GitHub, then from the snapshot folder run:

```bash
git init
git add .
git commit -m "Initial public portfolio snapshot"
git branch -M main
git remote add origin <YOUR_NEW_PUBLIC_REPO_URL>
git push -u origin main
```

## Recommended Repo Description

`Sanitized backend snapshot of a NameCoin-style decentralized DNS project with transaction validation and fork-aware chain resolution.`
