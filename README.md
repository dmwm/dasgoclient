# dasgoclient

[![Build Status](https://travis-ci.org/dmwm/dasgoclient.svg?branch=master)](https://travis-ci.org/dmwm/dasgoclient)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmwm/dasgoclient)](https://goreportcard.com/report/github.com/dmwm/dasgoclient)
[![GoDoc](https://godoc.org/github.com/dmwm/dasgoclient?status.svg)](https://godoc.org/github.com/dmwm/dasgoclient)
[![DOI](https://zenodo.org/badge/78777726.svg)](https://zenodo.org/badge/78777726.svg)

Go implementation of DAS (Data Aggregation System) client for CMS data-services

### Installation & Usage

To compile the client you need a Go compiler, then perform the following:

```
# one time operation, setup your GOPATH and download the following
go get github.com/dmwm/cmsauth
go get github.com/dmwm/das2go
go get github.com/vkuznet/x509proxy
go get github.com/buger/jsonparser
go get github.com/pkg/profile

# to build DAS Go client 
make build_all
```

The build will produce ```dasgoclient``` static executable which you can use as following:
```
dasgoclient -help
Usage: dasgoclient [options]
  -daskeys
        Show supported DAS keys
  -examples
        Show examples of supported DAS queries
  -json
        Return results in JSON data-format
  -profileMode string
        enable profiling mode, one of [cpu, mem, block]
  -query string
        DAS query to run
  -sep string
        Separator to use (default " ")
  -unique
        Sort results and return unique list
  -verbose int
        Verbose level, support 0,1,2
Examples:
        # get results
        dasgoclient -query="dataset=/ZMM*/*/*"
        # get results in JSON data-format
        dasgoclient -query="dataset=/ZMM*/*/*" -json
        # get results from specific CMS data-service, e.g. phedex
        dasgoclient -query="file dataset=/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM system=phedex" -json
```

### DAS queries example
You may use any DAS queries using dasgoclient, e.g. find files for a given dataset
```
./dasgoclient -query="file dataset=/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM"/store/mc/Summer11/ZMM/GEN-SIM/DESIGN42_V11_428_SLHC1-v1/0003/02ACAA1A-9F32-E111-BB31-0002C90B743A.root
/store/mc/Summer11/ZMM/GEN-SIM/DESIGN42_V11_428_SLHC1-v1/0003/02ACAA1A-9F32-E111-BB31-0002C90B743A.root
...
```
to see full records, use `-json` option, e.g.
```
./dasgoclient -query="file dataset=/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM" -json
{"file":[{"adler32":"eb5450c4","auto_cross_section":null,"block.name":"/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM#8420a3e7-96cf-48b1-a6a8-f9abcd3de8ef","block_id":5.959551e+06,"block_name":"/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM#8420a3e7-96cf-48b1-a6a8-f9abcd3de8ef","branch_hash_id":null,"check_sum":"2684046953","create_by":"cmsprod@cmsprod01.hep.wisc.edu","created_by":"cmsprod@cmsprod01.hep.wisc.edu","creation_date":1.325221208e+09,"creation_time":1.325221208e+09,"dataset":"/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM","dataset_id":4.043703e+06,"file_id":3.8629887e+07,"file_type_id":1,"is_file_valid":1,"last_modification_date":1.325267812e+09,"last_modified_by":"/DC=org/DC=doegrids/OU=People/CN=Ajit Kumar Mohapatra 867118","md5":"NOTSET","modification_time":1.325267812e+09,"modified_by":"/DC=org/DC=doegrids/OU=People/CN=Ajit Kumar Mohapatra 867118","name":"/store/mc/Summer11/ZMM/GEN-SIM/DESIGN42_V11_428_SLHC1-v1/0003/02ACAA1A-9F32-E111-BB31-0002C90B743A.root","nevents":null,"size":2.063019163e+09,"type":"EDM"}]}
```

### Profile dasgoclient
It is possible to invoke a profiler for dasgoclient tool. To do so please run
it with your favorite query, e.g.

```
dasgoclient -query="file dataset=/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM" -profileMode=cpu
```

then invoke go profiler as following:

```
go tool pprof ./dasgoclient cpu.pprof
Entering interactive mode (type "help" for commands)
(pprof) top10
20ms of 20ms total (  100%)
Showing top 10 nodes out of 20 (cum >= 10ms)
      flat  flat%   sum%        cum   cum%
      10ms 50.00% 50.00%       10ms 50.00%  github.com/dmwm/das2go/services.Unmarshal
      10ms 50.00%   100%       10ms 50.00%  runtime.memmove
         0     0%   100%       10ms 50.00%  crypto/rsa.(*PrivateKey).Sign
         0     0%   100%       10ms 50.00%  crypto/rsa.SignPKCS1v15
         0     0%   100%       10ms 50.00%  crypto/rsa.decrypt
         0     0%   100%       10ms 50.00%  crypto/rsa.decryptAndCheck
         0     0%   100%       10ms 50.00%  crypto/tls.(*Conn).Handshake
         0     0%   100%       10ms 50.00%  crypto/tls.(*Conn).clientHandshake
         0     0%   100%       10ms 50.00%  crypto/tls.(*clientHandshakeState).doFullHandshake
         0     0%   100%       10ms 50.00%  main.main
```
