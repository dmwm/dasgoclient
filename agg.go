package main

import (
	"encoding/json"
	"log"

	"github.com/dmwm/das2go/mongo"
)

// DAS structure represent DAS porition of DAS record
type DAS struct {
	Expire     int64    `json:"expire"`
	Instance   string   `json:"instance"`
	PrimaryKey string   `json:"primary_key"`
	Record     int      `json:"record"`
	Services   []string `json:"services"`
}

// File structure represents file name
type File struct {
	Name string `json:"name"`
}

// Run structure represents run number
type Run struct {
	Number int64 `json:"number"`
}

// Lumi structure represents lumi number
type Lumi struct {
	Number int64 `json:"number"`
}

// FileLumi rerpesents file lumi record
type FileLumi struct {
	File []File `json:"file"`
	Lumi []Lumi `json:"lumi"`
	Das  DAS    `json:"das"`
}

// helper function to aggregation file lumis
func aggregateFileLumis(records []mongo.DASRecord) []mongo.DASRecord {
	// {"das":{"expire":1645799095,"instance":"prod/global","primary_key":"file.name","record":1,"services":["dbs3:file_lumi4dataset"]},"file":[{"name":"/store/data/Commissioning2021/HLTPhysics/RAW/v1/000/346/512/00000/0797d739-0677-432e-91b9-7a6d8a0e5601.root"}],"lumi":[{"number":620}],"qhash":"78a0e0de6934303d25d459b5e06b9dad"
	amap := make(map[string][]int64)
	var das mongo.DASRecord
	for _, r := range records {
		das = r["das"].(mongo.DASRecord)
		var fileLumi FileLumi
		data, err := json.Marshal(r)
		if err != nil {
			log.Fatal(err.Error())
		}
		err = json.Unmarshal(data, &fileLumi)
		if err != nil {
			log.Fatal(err.Error())
		}
		if len(fileLumi.File) != len(fileLumi.Lumi) || len(fileLumi.File) != 1 {
			log.Fatal("wrong fileLumi record")
		}
		file := fileLumi.File[0].Name
		lumi := fileLumi.Lumi[0].Number
		if v, ok := amap[file]; ok {
			v = append(v, lumi)
			amap[file] = v
		} else {
			arr := []int64{lumi}
			amap[file] = arr
		}
	}

	var out []mongo.DASRecord
	for file, lumis := range amap {
		//         frec := File{Name: file}
		rec := make(mongo.DASRecord)
		rec["das"] = das
		frec := make(mongo.DASRecord)
		frec["name"] = file
		rec["file"] = []mongo.DASRecord{frec}
		//         rec["file"] = frec
		lrec := make(mongo.DASRecord)
		lrec["number"] = lumis
		rec["lumi"] = []mongo.DASRecord{lrec}
		out = append(out, rec)
	}
	return out
}

// helper function to aggregation file lumis
func aggregateRuns(records []mongo.DASRecord) []mongo.DASRecord {
	return records
}
