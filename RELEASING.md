# Releasing

How releases of doc-agent-ai actually work, the moving parts that have to be in place, and what to do when something breaks. Written for whoever cuts the next release (probably future-me).

## What happens when you push a `v*` tag

1. `.github/workflows/release.yml` triggers.
2. The job checks out the tagged commit, sets up Go 1.23, runs `goreleaser release --clean`.
3. goreleaser builds six binaries: linux, darwin, windows × amd64, arm64. Strips them. Names them `doc-agent-ai_<version>_<os>_<arch>.{tar.gz,zip}` (zip for Windows, tar.gz for everything else).
4. Adds Linux `.deb` and `.rpm` via `nfpms`.
5. Writes `checksums.txt`.
6. Publishes a GitHub Release. Tags matching `-rc`, `-beta`, `-alpha` are marked as pre-release automatically (`prerelease: auto` in `.goreleaser.yml`).
7. Pushes an updated `Formula/doc-agent-ai.rb` to [`zeshone/homebrew-tap`](https://github.com/zeshone/homebrew-tap).
8. Pushes an updated `bucket/doc-agent-ai.json` to [`zeshone/scoop-bucket`](https://github.com/zeshone/scoop-bucket).

All of this in roughly 50 seconds.

## Pre-requisites that have to be in place

Set up once. If any of these go missing, releases break.

### `RELEASE_TOKEN` secret

A fine-grained PAT with `Contents: Read and write` on `zeshone/doc-agent-ai`, `zeshone/homebrew-tap`, and `zeshone/scoop-bucket`. Stored as a repository secret named `RELEASE_TOKEN` on the `doc-agent-ai` repo. Used by goreleaser to push to the tap and the bucket.

Rotate every 90 days or so. When you rotate, regenerate the PAT and overwrite the secret value — the workflow file doesn't need to change.

If you ever leak this token, revoke it immediately at `Settings → Developer settings → PATs`. The blast radius is write access to three repositories.

### The tap and the bucket repos exist with a `main` branch

`zeshone/homebrew-tap` needs at least one commit on `main` (an empty `Formula/` directory with a `.gitkeep` is fine — goreleaser writes `Formula/doc-agent-ai.rb` on top of that). Same shape for `zeshone/scoop-bucket` with a `bucket/` directory.

If either repo is truly empty (no commits, no default branch), the first release will fail because goreleaser can't push to a nonexistent ref.

### Branch protection bypass

`zeshone/homebrew-tap` and `zeshone/scoop-bucket` both have a ruleset that blocks direct pushes to `main`, with Admin as the bypass actor. The account behind `RELEASE_TOKEN` needs Admin on the org (or be added explicitly as a bypass actor). Without that, the push fails with `Cannot update this protected ref` even though the bot has write access.

## How to cut a release

1. Make sure `main` has all the changes you want shipped.
2. Make sure the [`CHANGELOG.md`](./CHANGELOG.md) has a section for the version.
3. Tag and push:

    ```sh
    git checkout main
    git pull
    git tag -a v3.4.0 -m "Release notes summary"
    git push origin v3.4.0
    ```

4. Watch the workflow at `Actions → Release` on GitHub. Should complete in under a minute.
5. Verify (in this order):
   - `gh release view v3.4.0 --repo zeshone/doc-agent-ai` — 11 assets and `isPrerelease: false` (or `true` if it was an `-rc` / `-beta`).
   - [zeshone/homebrew-tap → Formula/doc-agent-ai.rb](https://github.com/zeshone/homebrew-tap/blob/main/Formula/doc-agent-ai.rb) — version matches the tag.
   - [zeshone/scoop-bucket → bucket/doc-agent-ai.json](https://github.com/zeshone/scoop-bucket/blob/main/bucket/doc-agent-ai.json) — version matches the tag.
6. Smoke test on at least one platform: `brew install`, `scoop install`, or just `doc-agent-ai --version` after running the new binary.

## Testing the pipeline before the real tag

Always do this if you've touched `.goreleaser.yml` or `release.yml`. Tag a pre-release:

```sh
git tag -a v3.4.0-rc1 -m "pre-release test"
git push origin v3.4.0-rc1
```

`prerelease: auto` will mark it correctly on GitHub. The tap and bucket get updated too (this is intentional — we want to test the full path, not just the build).

If something fails:

- Read the failed run logs: `gh run view <id> --log-failed`.
- Fix on the branch (not on `main` necessarily — most release config lives in the repo so it can also be tested from a feature branch with its own tag).
- Tag `-rc2`, `-rc3`, until clean.

When you're satisfied, cut the real `v3.4.0` tag.

## Cleaning up after pre-release iteration

Once the real release is out, the `-rc*` tags and their releases are noise. Delete:

```sh
# delete the GitHub Release
gh release delete v3.4.0-rc1 --repo zeshone/doc-agent-ai --yes

# delete the tag (remote and local)
git push origin :refs/tags/v3.4.0-rc1
git tag -d v3.4.0-rc1
```

Tag deletion goes through the tag ruleset's deletion protection. Admins bypass it automatically.

## Things that have broken before

- **Workflow not registered on default branch.** Tag pushes only trigger workflows that exist on the default branch (or in the tagged commit). If you tag from a feature branch that has `release.yml` but `main` doesn't, `gh run list --workflow=release.yml` returns 404 even though the workflow ran. The run is still findable with plain `gh run list`.

- **Permission denied on a shell script.** Files committed from Windows Git Bash with `core.fileMode=false` lose their executable bit. Fix with `git update-index --chmod=+x <path>` before committing. The Linux runner will refuse to fork/exec otherwise.

- **`brews` failed with "one tap can handle only one archive of an OS/Arch combination".** The `archives:` block produced both a `.tar.gz` and a `.zip` for every target. Homebrew needs exactly one. The fix is `format_overrides: [{goos: windows, formats: [zip]}]` so tar.gz is the default and Windows gets zip — one archive per target.

- **Homebrew push failed silently with `branch=` empty.** goreleaser's `brews.repository` block needs `branch: main` explicit. Without it the push log shows `branch=` blank and the formula never lands.

- **GitHub Release published as stable instead of pre-release.** goreleaser v2 default is stable. `release.prerelease: auto` parses the tag suffix (`-rc1`, `-beta.1`, etc.) and flags the release correctly.

- **Custom Scoop publisher script ran 11 times.** A `publishers:` block runs once per artifact, not once per release. First call succeeded, the rest hit a stale repo and got rejected with `cannot lock ref: is at NEW but expected OLD`. The fix was switching to goreleaser's native `scoops:` block, which runs once.

- **`Cannot update this protected ref` even though the bypass message appeared.** The PAT user's role on the target repo wasn't Admin, and the ruleset bypass is `RepositoryRole: 5` (Admin). Either elevate the user, add a user-specific bypass, or move to a GitHub App that's wired as a bypass actor by Integration ID.
