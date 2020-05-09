# login to lxplus
cd workspace/builds
# login on vocms0181
cd /wma/vk/das_builds
export SCRAM_ARCH=slc7_amd64_gcc700
export SCRAM_ARCH=slc7_amd64_gcc820
cd cmsdist
# either create new git branch

# OPTIONAL
# checkout branch I need from upstream (OPTIONAL if it does no exists)
git checkout master
git fetch upstream; git rebase upstream/master
git checkout -b IB/CMSSW_11_1_X/master upstream/IB/CMSSW_11_1_X/master
# push this branch into my cmsdist (OPTIONAL depends on previous step)
git push -u origin IB/CMSSW_11_1_X/master
# END  OPTIONAL

# if I have this branch I need to sync first
git checkout IB/CMSSW_11_1_X/master
git fetch upstream; git rebase upstream/IB/CMSSW_11_1_X/master
git push
git checkout -b dasgoclient-v02.02.08
git branch -l

# change specs
vim dasgoclient*.spec

# build RPMS
cd .. # cd ~/workspace/builds
./build.sh dasgoclient-binary

# locate RPM
ls -al w700/RPMS/slc6_amd64_gcc700/cms+dasgoclient-binary+v02.02.08-1-1.slc6_amd64_gcc700.rpm
ls -al w820/RPMS/slc7_amd64_gcc820/cms+dasgoclient-binary+v02.02.08-1-1.slc7_amd64_gcc820.rpm

# copy RPM to EOS area
cp w700/RPMS/slc6_amd64_gcc700/cms+dasgoclient-binary+v02.02.08-1-1.slc6_amd64_gcc700.rpm /eos/user/v/valya/www/dasgoclient/
cp w820/RPMS/slc7_amd64_gcc820/cms+dasgoclient-binary+v02.02.08-1-1.slc7_amd64_gcc820.rpm /eos/user/v/valya/www/dasgoclient/
# copy RPM on vocms0181
scp w700/RPMS/slc6_amd64_gcc700/cms+dasgoclient-binary+v02.02.08-1-1.slc6_amd64_gcc700.rpm valya@lxplus.cern.ch:/eos/user/v/valya/www/dasgoclient/
scp w820/RPMS/slc7_amd64_gcc820/cms+dasgoclient-binary+v02.02.08-1-1.slc7_amd64_gcc820.rpm valya@lxplus.cern.ch:/eos/user/v/valya/www/dasgoclient/


# now we can build dasgoclient wrapper since it look-up dasgoclient RPMs
./build.sh dasgoclient

# test new client
cp w700/slc6_amd64_gcc700/cms/dasgoclient-binary/v02.02.08/bin/dasgoclient_linux ./dasgoclient
cp w820/slc7_amd64_gcc820/cms/dasgoclient-binary/v02.02.08/bin/dasgoclient_linux ./dasgoclient
voms-proxy-init -voms cms -rfc
./dasgoclient -help

# commit changes
cd cmsdist
git commit -m "New dasgoclient version" dasgoclient*.spec

# push changes
git push -u origin dasgoclient-v02.02.08

# finally make pull request

# delete local branch (named newfeature)
git branch -d dasgoclient-v02.02.08
# delete remote branch (named newfeature)
git push origin :dasgoclient-v02.02.08
