# DASgoclient DevOps Procedures

This document describes the procedures for:

1. building `dasgoclient` against a local `das2go` source tree;
2. ramping the `das2go` dependency used by `dasgoclient`;
3. publishing a new `dasgoclient` release through `cmsdist` and CMS CVMFS.

## Repositories and external references

* DASgoclient: https://github.com/dmwm/dasgoclient
* DAS2Go: https://github.com/dmwm/das2go
* CMSDIST: https://github.com/cms-sw/cmsdist
* CMS build infrastructure: https://github.com/cms-sw/cms-bot
* CMSSW release schedule: https://twiki.cern.ch/twiki/bin/view/CMS/ReleaseSchedule

---

# 1. Build DASgoclient against a local DAS2Go tree

## Requirements

Use this procedure when `das2go` changes need to be tested through `dasgoclient` before they are merged and released upstream.

The local repositories are expected to have the following layout:

```text
workspace/
├── dasgoclient/
└── das2go/
```

The `das2go` checkout must contain the code to be tested.

The local replacement is a development-only configuration and must not be committed as part of a DASgoclient release.

## Steps

From the `dasgoclient` directory:

```bash
go mod edit -replace=github.com/dmwm/das2go=../das2go
go mod tidy
```

Verify that the local source tree is selected:

```bash
go list -m -f '{{.Path}} => {{.Replace}}' github.com/dmwm/das2go
```

Build the client:

```bash
make build
```

Run the required functional/regression tests.

After testing, remove the local replacement:

```bash
go mod edit -dropreplace=github.com/dmwm/das2go
go mod tidy
```

Verify that no `replace` directive remains:

```bash
grep -n '^replace.*github.com/dmwm/das2go' go.mod
```

The command must return no result.

---

# 2. Ramp DASgoclient to a released DAS2Go version

## Requirements

A production DASgoclient build must depend on the upstream module:

```text
https://github.com/dmwm/das2go
```

Temporary references to forks or local source trees must not be present.

DAS2Go release tags such as:

```text
v04.07.43rc1
```

are project release tags. They cannot be used directly as Go semantic module versions because of the leading zero in `v04`.

The DASgoclient dependency must therefore be updated using the **commit associated with the DAS2Go release**. The Go toolchain then generates the corresponding pseudo-version.

Before starting, determine:

```text
<DAS2GO_RELEASE>
<DAS2GO_RELEASE_COMMIT>
```

from:

https://github.com/dmwm/das2go/releases

## Steps

Start from the current DASgoclient master branch:

```bash
git checkout master
git pull
git checkout -b ramp-das2go-<DAS2GO_RELEASE>
```

Remove any development replacement if present:

```bash
go mod edit -dropreplace=github.com/dmwm/das2go
```

Update the dependency using the exact DAS2Go release commit:

```bash
go get github.com/dmwm/das2go@<DAS2GO_RELEASE_COMMIT>
go mod tidy
```

Verify the resolved dependency:

```bash
go list -m -json github.com/dmwm/das2go
```

Build DASgoclient:

```bash
make build
```

Verify the resulting client:

```bash
./dasgoclient -version
```

Run the regression tests relevant to the changes contained in the new DAS2Go release.

Review the repository changes:

```bash
git diff -- go.mod go.sum
```

For a normal dependency ramp, the expected modified files are:

```text
go.mod
go.sum
```

Commit the change:

```bash
git add go.mod go.sum
git commit -m "Ramping das2go version to <DAS2GO_RELEASE>"
```

---

# 3. Publish a DASgoclient release through CMSDIST

## Requirements

A new DASgoclient version must already be tagged and released in:

https://github.com/dmwm/dasgoclient/releases

The corresponding CMSDIST update is performed in:

https://github.com/cms-sw/cmsdist

The target CMSSW IB series must be selected before creating the CMSDIST branch.

The CMSDIST PR must target an active branch of the form:

```text
IB/<CMSSW_SERIES>/master
```

The `dasgoclient.spec` update for a new DASgoclient release changes:

```text
%define dasgoclient_tag
```

Do not increment `version_suffix` solely because a new upstream DASgoclient tag is being used.

---

## 3.1 Select the CMSSW IB series

### Requirements

Do not assume that the numerically newest CMSSW series is automatically the appropriate target.

Use both:

* the active IB definitions in
  https://github.com/cms-sw/cms-bot/blob/master/releases.map
* the CMSSW release schedule at
  https://twiki.cern.ch/twiki/bin/view/CMS/ReleaseSchedule

The schedule matters because different CMSSW series may have pre-releases planned at different dates.

### Steps

List the active development IB definitions:

```bash
curl -s https://raw.githubusercontent.com/cms-sw/cms-bot/master/releases.map \
  | grep 'type=Development;state=IB' \
  | grep 'prodarch=1'
```

Identify the appropriate:

```text
label=<CMSSW_SERIES>
```

using the active IB information together with the CMSSW release schedule.

Verify that the corresponding CMSDIST branch exists:

```bash
git ls-remote --heads https://github.com/cms-sw/cmsdist.git \
  "refs/heads/IB/<CMSSW_SERIES>/master"
```

The command must return the matching branch.

Use:

```text
IB/<CMSSW_SERIES>/master
```

as the base branch for the CMSDIST PR.

---

## 3.2 Create the CMSDIST PR

### Requirements

Determine:

```text
<DASGOCLIENT_TAG>
<CMSSW_SERIES>
<GITHUB_USER>
```

before creating the branch.

A fork of:

https://github.com/cms-sw/cmsdist

must be available under `<GITHUB_USER>`.

### Steps

Clone the fork and configure the upstream repository:

```bash
git clone git@github.com:<GITHUB_USER>/cmsdist.git
cd cmsdist

git remote add upstream https://github.com/cms-sw/cmsdist.git
git fetch upstream
```

Create a branch from the selected CMSSW IB:

```bash
git checkout -b IB/<CMSSW_SERIES>/master_dasgoclient-<DASGOCLIENT_TAG> \
  upstream/IB/<CMSSW_SERIES>/master
```

Edit:

```text
dasgoclient.spec
```

and update only the DASgoclient tag:

```diff
-%define dasgoclient_tag <OLD_DASGOCLIENT_TAG>
+%define dasgoclient_tag <DASGOCLIENT_TAG>
```

Review the change:

```bash
git diff -- dasgoclient.spec
```

Commit and push:

```bash
git add dasgoclient.spec

git commit -m "Update dasgoclient version to <DASGOCLIENT_TAG>"

git push origin \
  IB/<CMSSW_SERIES>/master_dasgoclient-<DASGOCLIENT_TAG>
```

Create a pull request with:

```text
base:
cms-sw/cmsdist:IB/<CMSSW_SERIES>/master

head:
<GITHUB_USER>:IB/<CMSSW_SERIES>/master_dasgoclient-<DASGOCLIENT_TAG>
```

Use a title of the form:

```text
Update dasgoclient version to <DASGOCLIENT_TAG>
```

The PR description should identify the DASgoclient release and link the relevant DAS2Go release, fixes, or DASMaps changes that require the new client version.

---

## 3.3 Validate CMSDIST and CVMFS publication


## Requirements

The CMS build infrastructure runs the integration build after the CMSDIST PR is submitted.
A successful integration build must be validated before relying on the new client.

After the CMSDIST PR is submitted, the CMS build infrastructure builds the updated package.
The new client must be validated first from the integration environment and, after publication, from production CVMFS.


### Steps

Check the CMS bot results attached to the CMSDIST PR and, when provided, test the temporary CMS CI build.

After the change reaches the target IB, verify:

```bash
/cvmfs/cms-ib.cern.ch/sw/x86_64/latest/common/dasgoclient -version
```

Run the required DAS regression queries against this binary.

After production publication, verify:

```bash
/cvmfs/cms.cern.ch/common/dasgoclient -version
```

The procedure is complete when the expected version is available and validated from the production location.

---

# Example: previous CMSDIST ramp

A previous DASgoclient update is available as a complete CMSDIST PR example:

https://github.com/cms-sw/cmsdist/pull/10677

It can be used to inspect the expected `dasgoclient.spec` change, CMS bot processing, and PR structure.

The example is provided for reference only. The CMSSW series, DASgoclient version, branch names, and CI paths must always be determined for the release being prepared.
