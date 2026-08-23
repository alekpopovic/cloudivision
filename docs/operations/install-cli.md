# Install the cloudivision CLI

Release binaries are statically linked and published for Linux, macOS, and Windows on amd64 and arm64. Verify downloaded archives against `SHA256SUMS` from the same GitHub release before installing them.

## Linux and macOS

The installer detects the operating system and architecture, verifies the checksum, and installs to `/usr/local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/alekpopovic/cloudivision/v0.2.0/scripts/install-cli.sh | \
  CLOUDIVISION_CLI_VERSION=0.2.0 bash
cloudivision version
```

Pin both the installer source and `CLOUDIVISION_CLI_VERSION` to the same release.
Using the `main` installer with `latest` is convenient for evaluation but is not a
reproducible installation.

Use a pinned release and a user-owned destination when desired:

```sh
curl -fsSLO https://raw.githubusercontent.com/alekpopovic/cloudivision/v0.2.0/scripts/install-cli.sh
CLOUDIVISION_CLI_VERSION=0.2.0 \
CLOUDIVISION_CLI_INSTALL_DIR="$HOME/.local/bin" \
  bash install-cli.sh
```

For a manual install, download `cloudivision-cli_VERSION_OS_ARCH.tar.gz`, verify it with `sha256sum -c SHA256SUMS`, extract it, and move `cloudivision` to a directory on `PATH`.

## Windows

Download `cloudivision-cli_VERSION_windows_amd64.zip` (or `windows_arm64.zip`) and `SHA256SUMS` from the GitHub release. In PowerShell, verify and install it:

```powershell
(Get-FileHash .\cloudivision-cli_0.2.0_windows_amd64.zip -Algorithm SHA256).Hash
Expand-Archive .\cloudivision-cli_0.2.0_windows_amd64.zip
$env:Path += ";$PWD\cloudivision-cli_0.2.0_windows_amd64"
cloudivision.exe version
```

Compare the hash with `SHA256SUMS` before extraction. Add the extracted directory to the persistent user `PATH` for later shells.

## Source and local installs

`make build-cli` writes a versioned binary to `bin/cloudivision`. `make install-cli-local` installs it to `GOBIN`, or to `GOPATH/bin` when `GOBIN` is unset.

## Shell completion

Generate all supported completion files with `make cli-completions`, or load one directly:

```sh
source <(cloudivision completion bash)
# zsh: cloudivision completion zsh > "${fpath[1]}/_cloudivision"
# fish: cloudivision completion fish > ~/.config/fish/completions/cloudivision.fish
```

PowerShell users can add the output of `cloudivision completion powershell` to their profile.

## Release packaging

Tagged releases run the release workflow and attach CLI archives for every supported platform alongside checksums. `.goreleaser.yaml` mirrors the CLI build matrix for maintainers who use GoReleaser. The repository release builder also includes these archives in `dist/release/vVERSION`.

After installation, run `cloudivision doctor` with the API URL and credentials to validate connectivity. Tokens should come from `CLOU_DIVISION_TOKEN` or the mode-0600 CLI config, not shell history.

The complete platform assets, upgrade impact, and known issues are recorded in
the [v0.2.0 release notes](../releases/v0.2.0.md).
