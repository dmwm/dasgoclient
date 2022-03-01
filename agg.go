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
	Number int64 `json:"run_number"`
}

// Lumi structure represents lumi number
type Lumi struct {
	Number int64 `json:"number"`
}

// Event structure represents event number
type Event struct {
	Number int64 `json:"number"`
}

// FileLumi rerpesents file lumi record
type FileLumi struct {
	File   []File  `json:"file"`
	Runs   []Run   `json:"run"`
	Lumis  []Lumi  `json:"lumi"`
	Events []Event `json:"events"`
	Das    DAS     `json:"das"`
}

// InList helper function to check item in a list
func InList(a int64, list []int64) bool {
	check := 0
	for _, b := range list {
		if b == a {
			check += 1
		}
	}
	if check != 0 {
		return true
	}
	return false
}

// helper function to aggregation file lumis
func aggregateFileLumis(records []mongo.DASRecord) []mongo.DASRecord {
	// {"das":{"expire":1645799095,"instance":"prod/global","primary_key":"file.name","record":1,"services":["dbs3:file_lumi4dataset"]},"file":[{"name":"/store/data/Commissioning2021/HLTPhysics/RAW/v1/000/346/512/00000/0797d739-0677-432e-91b9-7a6d8a0e5601.root"}],"lumi":[{"number":620}],"qhash":"78a0e0de6934303d25d459b5e06b9dad"
	// {"event_count":1831,"logical_file_name":"/store/data/Run2018A/DoubleMuon/RAW/v1/000/316/469/00000/ACEDE0D3-2A5B-E811-BC13-FA163EFFF7A4.root","lumi_section_num":160,"run_num":316469}
	// {"das":{"expire":1646070229,"instance":"prod/global","primary_key":"file.name","record":1,"services":["dbs3:file_run_lumi_evts4dataset"]},"events":[{"number":514}],"file":[{"name":"/store/data/Run2018A/DoubleMuon/RAW/v1/000/315/257/00000/E422999B-9F49-E811-B098-FA163E94BBA0.root"}],"lumi":[{"number":39}],"qhash":"b4864bb1da23f6d2214a4f442bc00a02","run":[{"run_number":315257}]}
	amap := make(map[string][]int64)
	emap := make(map[string][]int64)
	rmap := make(map[string][]int64)
	var das mongo.DASRecord
	for _, r := range records {
		das = r["das"].(mongo.DASRecord)
		var fileLumi FileLumi
		data, err := json.Marshal(r)
		if err != nil {
			log.Fatal(err.Error())
		}
		//         fmt.Println("#### record", string(data))
		err = json.Unmarshal(data, &fileLumi)
		if err != nil {
			log.Fatal(err.Error())
		}
		if len(fileLumi.File) != len(fileLumi.Lumis) || len(fileLumi.File) != 1 {
			log.Fatal("wrong fileLumi record")
		}
		file := fileLumi.File[0].Name
		var run, evt, lumi int64
		if len(fileLumi.Lumis) > 0 {
			lumi = fileLumi.Lumis[0].Number
		}
		if len(fileLumi.Runs) > 0 {
			run = fileLumi.Runs[0].Number
		}
		if len(fileLumi.Events) > 0 {
			evt = fileLumi.Events[0].Number
		}
		if lumi > 0 {
			if v, ok := amap[file]; ok {
				if !InList(lumi, v) {
					v = append(v, lumi)
					amap[file] = v
				}
			} else {
				arr := []int64{lumi}
				amap[file] = arr
			}
		}
		if run > 0 {
			if v, ok := rmap[file]; ok {
				if !InList(run, v) {
					v = append(v, run)
					rmap[file] = v
				}
			} else {
				arr := []int64{run}
				rmap[file] = arr
			}
		}
		if evt > 0 {
			if v, ok := emap[file]; ok {
				if !InList(evt, v) {
					v = append(v, evt)
					emap[file] = v
				}
			} else {
				arr := []int64{evt}
				emap[file] = arr
			}
		}
	}

	var out []mongo.DASRecord
	for file, lumis := range amap {
		rec := make(mongo.DASRecord)
		rec["das"] = das
		frec := make(mongo.DASRecord)
		frec["name"] = file
		rec["file"] = []mongo.DASRecord{frec}
		lrec := make(mongo.DASRecord)
		lrec["number"] = lumis
		rec["lumi"] = []mongo.DASRecord{lrec}
		if v, ok := rmap[file]; ok {
			runs := make(mongo.DASRecord)
			if len(v) == 1 {
				runs["run_number"] = v[0]
			} else {
				runs["run_number"] = v
			}
			rec["run"] = []mongo.DASRecord{runs}
		}
		if v, ok := emap[file]; ok {
			evts := make(mongo.DASRecord)
			evts["number"] = v
			rec["events"] = []mongo.DASRecord{evts}
		}
		out = append(out, rec)
	}
	return out
}

// helper function to aggregation file lumis
func aggregateRuns(records []mongo.DASRecord) []mongo.DASRecord {
	return records
}
