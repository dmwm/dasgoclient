# re-generate go.mod/go.sum
rm go.mod go.sum && go mod init && go mod tidy

# login to lxplus
cd workspace/builds
# login on vocms0181
cd /wma/vk/das_builds
export SCRAM_ARCH=slc7_amd64_gcc820
cd cmsdist
# either create new git branch

# OPTIONAL
# checkout branch I need from upstream (OPTIONAL if it does no exists)
git checkout master
git fetch upstream; git rebase upstream/master
git checkout -b IB/CMSSW_14_0_X/master upstream/IB/CMSSW_14_0_X/master
# push this branch into my cmsdist (OPTIONAL depends on previous step)
git push -u origin IB/CMSSW_14_0_X/master
# END  OPTIONAL

# if I have this branch I need to sync first
git checkout IB/CMSSW_14_0_X/master
git fetch upstream; git rebase upstream/IB/CMSSW_14_0_X/master
git push
git checkout -b dasgoclient-v02.04.51
git branch -l

# change specs
vim dasgoclient*.spec

# now we can build dasgoclient wrapper since it look-up dasgoclient RPMs
./build.sh dasgoclient

# test new client
cp w820/slc7_amd64_gcc820/cms/dasgoclient/v02.04.51*/bin/dasgoclient .
voms-proxy-init -voms cms -rfc
./dasgoclient -help

# commit changes
cd cmsdist
git commit -m "New dasgoclient version" dasgoclient*.spec

# push changes
git push -u origin dasgoclient-v02.04.51

# finally make pull request

# delete local branch (named newfeature)
git branch -d dasgoclient-v02.04.51
# delete remote branch (named newfeature)
git push origin :dasgoclient-v02.04.51
