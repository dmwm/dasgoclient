# dasgoclient

[![Build Status](https://travis-ci.org/vkuznet/dasgoclient.svg?branch=master)](https://travis-ci.org/vkuznet/dasgoclient)
[![Go Report Card](https://goreportcard.com/badge/github.com/vkuznet/dasgoclient)](https://goreportcard.com/report/github.com/vkuznet/dasgoclient)
[![GoDoc](https://godoc.org/github.com/vkuznet/dasgoclient?status.svg)](https://godoc.org/github.com/vkuznet/dasgoclient)
[![DOI](https://zenodo.org/badge/78777726.svg)](https://zenodo.org/badge/78777726.svg)

Go implementation of DAS (Data Aggregation System) client for CMS data-services

### Installation & Usage

To compile the client you need a Go compiler, then perform the following:

```
# one time operation, setup your GOPATH and download the following
go get github.com/vkuznet/cmsauth
go get github.com/vkuznet/das2go
go get github.com/vkuznet/x509proxy
go get github.com/buger/jsonparser
go get github.com/pkg/profile

# to build DAS Go client 
make build_all
```

The build will produce ```dasgoclient``` static executable which you can use as following:
```
./dasgoclient -help
Usage of ./dasgoclient:
  -inst string
        DBS instance to use (default "prod/global")
  -json
        Return results from DAS CLI in json form
  -query string
        DAS query to run
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
