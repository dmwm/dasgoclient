package main

// dasgoclient - Go implementation of Data Aggregation System (DAS) client for CMS
//
// Copyright (c) 2017 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/dmwm/das2go/das"
	"github.com/dmwm/das2go/dasmaps"
	"github.com/dmwm/das2go/dasql"
	"github.com/dmwm/das2go/mongo"
	"github.com/dmwm/das2go/services"
	"github.com/dmwm/das2go/utils"
	"github.com/pkg/profile"
)

func main() {
	var query string
	flag.StringVar(&query, "query", "", "DAS query to run")
	var jsonout bool
	flag.BoolVar(&jsonout, "json", false, "Return results in JSON data-format")
	var format string
	flag.StringVar(&format, "format", "", "Compatibility option with python das_client, use json to get das_client behavior")
	var limit int
	flag.IntVar(&limit, "limit", 0, "Compatibility option with python das_client")
	var idx int
	flag.IntVar(&idx, "idx", 0, "Compatibility option with python das_client")
	var host string
	flag.StringVar(&host, "host", "https://cmsweb.cern.ch", "Specify hostname to talk to")
	var threshold int
	flag.IntVar(&threshold, "threshold", 0, "Compatibility option with python das_client, has no effect")
	var sep string
	flag.StringVar(&sep, "sep", " ", "Separator to use")
	var dasmaps string
	flag.StringVar(&dasmaps, "dasmaps", "", "Specify location of dasmaps")
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "Verbose level, support 0,1,2")
	var examples bool
	flag.BoolVar(&examples, "examples", false, "Show examples of supported DAS queries")
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	var daskeys bool
	flag.BoolVar(&daskeys, "daskeys", false, "Show supported DAS keys")
	var unique bool
	flag.BoolVar(&unique, "unique", false, "Sort results and return unique list")
	var timeout int
	flag.IntVar(&timeout, "timeout", 0, "Timeout for url call")
	var urlRetry int
	flag.IntVar(&urlRetry, "urlRetry", 3, "urlRetry for url call")
	flag.Usage = func() {
		fmt.Println("Usage: dasgoclient [options]")
		flag.PrintDefaults()
		fmt.Println("Examples:")
		fmt.Println("\t# get results")
		fmt.Println("\tdasgoclient -query=\"dataset=/ZMM*/*/*\"")
		fmt.Println("\t# get results in JSON data-format")
		fmt.Println("\tdasgoclient -query=\"dataset=/ZMM*/*/*\" -json")
		fmt.Println("\t# get results from specific CMS data-service, e.g. phedex")
		fmt.Println("\tdasgoclient -query=\"file dataset=/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM system=phedex\" -json")
	}
	mode := flag.String("profileMode", "", "enable profiling mode, one of [cpu, mem, block]")
	flag.Parse()
	switch *mode {
	case "cpu":
		defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()
	case "mem":
		defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	case "block":
		defer profile.Start(profile.BlockProfile, profile.ProfilePath(".")).Stop()
	default:
		// do nothing
	}
	utils.DASMAPS = dasmaps
	utils.VERBOSE = verbose
	utils.UrlQueueLimit = 1000
	utils.UrlRetry = urlRetry
	utils.WEBSERVER = 0
	utils.TIMEOUT = timeout
	utils.CLIENT_VERSION = "{{VERSION}}"
	checkX509()
	if examples {
		showExamples()
	} else if version {
		fmt.Println(info())
	} else if daskeys {
		showDASKeys()
	} else {
		process(query, jsonout, sep, unique, format, host, idx, limit)
	}
}

func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now()
	return fmt.Sprintf("Build: git={{VERSION}} go=%s date=%s", goVersion, tstamp)
}

// helper function to check X509 settings
func checkX509() {
	uproxy := os.Getenv("X509_USER_PROXY")
	uckey := os.Getenv("X509_USER_KEY")
	ucert := os.Getenv("X509_USER_CERT")
	var check int
	if uproxy == "" {
		// check if /tmp/x509up_u$UID exists
		u, err := user.Current()
		if err == nil {
			fname := fmt.Sprintf("/tmp/x509up_u%s", u.Uid)
			if _, err := os.Stat(fname); err != nil {
				check += 1
			}
		}
	}
	if uckey == "" && ucert == "" {
		check += 1
	}
	if check > 1 {
		fmt.Println("Neither X509_USER_PROXY or X509_USER_KEY/X509_USER_CERT are set")
		fmt.Println("In order to run please obtain valid proxy via \"voms-proxy-init -voms cms -rfc\"")
		fmt.Println("and setup X509_USER_PROXY or setup X509_USER_KEY/X509_USER_CERT in your environment")
		os.Exit(1)
	}
}

// helper function to show examples of DAS queries
func showExamples() {
	examples := []string{"block_queries.txt", "file_queries.txt", "lumi_queries.txt", "mcm_queries.txt", "run_queries.txt", "dataset_queries.txt", "jobsummary_queries.txt", "misc_queries.txt", "site_queries.txt"}
	var home string
	for _, item := range os.Environ() {
		value := strings.Split(item, "=")
		if value[0] == "HOME" {
			home = value[1]
			break
		}
	}
	for _, fname := range examples {
		arr := strings.Split(fname, "_")
		msg := fmt.Sprintf("### %s queries:\n", arr[0])
		fmt.Println(strings.ToTitle(msg))
		fmt.Println(utils.LoadExamples(fname, home))
	}
}

// global keymap for DAS keys and associate CMS data-service
func DASKeyMap() map[string][]string {
	keyMap := map[string][]string{
		"site":    []string{"combined"},
		"dataset": []string{"dbs3"},
		"block":   []string{"dbs3"},
		"file":    []string{"dbs3"},
		"run":     []string{"runregistry", "dbs3"},
		"config":  []string{"reqmgr2"},
	}
	return keyMap
}

// helper function to extracvt services from DAS map
func dasServices(rec mongo.DASRecord) []string {
	var out []string
	switch s := rec["services"].(type) {
	case string:
		out = append(out, rec["system"].(string))
	case map[string]interface{}:
		for k, _ := range s {
			out = append(out, k)
		}
	}
	return out
}

// helper function to show supported DAS keys
func showDASKeys() {
	var dmaps dasmaps.DASMaps
	dmaps.LoadMapsFromFile()
	keys := make(map[string][]string)
	for _, rec := range dmaps.Maps() {
		if rec["lookup"] != nil {
			lookup := rec["lookup"].(string)
			if lookup == "city" || lookup == "zip" || lookup == "ip" || lookup == "monitor" {
				continue
			}
			if s, ok := keys[lookup]; ok {
				for _, v := range dasServices(rec) {
					s = append(s, v)
				}
				keys[lookup] = s
			} else {
				keys[lookup] = dasServices(rec)
			}
		}
	}
	keyMap := DASKeyMap()
	fmt.Println("DAS keys and associated CMS data-service info")
	fmt.Println("---------------------------------------------")
	var sKeys []string
	for k, _ := range keys {
		sKeys = append(sKeys, k)
	}
	sort.Sort(utils.StringList(sKeys))
	for _, k := range sKeys {
		msg := fmt.Sprintf("%s comes from %v services", k, utils.List2Set(keys[k]))
		if val, ok := keyMap[k]; ok {
			msg = fmt.Sprintf("%s, default is %s system", msg, val)
		}
		fmt.Println(msg)
	}
}

// helper function to make a choice which CMS data-service will be used for DAS query
// in other words it let us skip unnecessary system
func skipSystem(dasquery dasql.DASQuery, system string) bool {
	if len(dasquery.Fields) > 1 { // multiple keys
		return false
	}
	fields := dasquery.Fields
	specKeys := utils.MapKeys(dasquery.Spec)
	if len(fields) == 1 && len(specKeys) == 1 && fields[0] == specKeys[0] { // e.g. site=T3_*
		return false
	}
	keyMap := DASKeyMap()
	if dasquery.System == "" {
		for _, key := range dasquery.Fields {
			srvs := keyMap[key]
			if !utils.InList(system, srvs) {
				return true
			}
		}
	} else {
		if dasquery.System != system {
			return true
		}
	}
	return false
}

// Process function process' given query and return back results
func process(query string, jsonout bool, sep string, unique bool, format, host string, rdx, limit int) {
	time0 := time.Now().Unix()
	if strings.ToLower(format) == "json" {
		jsonout = true
	}
	var dmaps dasmaps.DASMaps
	dmaps.LoadMapsFromFile()
	if !strings.Contains(host, "cmsweb.cern.ch") {
		dmaps.ChangeUrl("https://cmsweb.cern.ch", host)
		dmaps.ChangeUrl("prod/global", "int/global")
	}
	dasquery, err := dasql.Parse(query, "", dmaps.DASKeys())
	if utils.VERBOSE > 0 {
		fmt.Println(dasquery, err)
	}
	// check dasquery and overwrite unique filter for everything except file
	if !unique && !utils.InList("file", dasquery.Fields) && !dasquery.Detail {
		unique = true
	}
	if utils.VERBOSE > 0 {
		fmt.Println("### unique", unique)
	}

	// find out list of APIs/CMS services which can process this query request
	maps := dmaps.FindServices(dasquery)
	var srvs, pkeys, mapServices []string
	urls := make(map[string]string)
	var localApis []mongo.DASRecord
	var furl string
	for _, dmap := range maps {
		if v, ok := dmap["system"]; ok {
			srv := v.(string)
			if !utils.InList(srv, mapServices) {
				mapServices = append(mapServices, srv)
			}
		}
	}
	// loop over services and select which one(s) we'll use
	var selectedServices []string
	for _, dmap := range maps {
		system, _ := dmap["system"].(string)
		if skipSystem(dasquery, system) && len(mapServices) > 1 {
			continue
		}
		selectedServices = append(selectedServices, system)
	}
	// if nothing is selected use original from the map
	if len(selectedServices) == 0 {
		selectedServices = mapServices
	}
	// loop over services and fetch data
	for _, dmap := range maps {
		args := ""
		system, _ := dmap["system"].(string)
		if !utils.InList(system, selectedServices) {
			continue
		}
		if system == "runregistry" {
			switch v := dasquery.Spec["run"].(type) {
			case string:
				args = fmt.Sprintf("{\"filter\": {\"number\": \">= %s and <= %s\"}}", v, v)
			case []string:
				cond := fmt.Sprintf("= %s", v[0])
				for i, vvv := range v {
					if i > 0 {
						cond = fmt.Sprintf("%s or = %s", cond, vvv)
					}
				}
				args = fmt.Sprintf("{\"filter\": {\"number\": \"%s\"}}", cond)
				//                 args = fmt.Sprintf("{\"filter\": {\"number\": \">= %s and <= %s\"}}", v[0], v[len(v)-1])
			}
			furl, _ = dmap["url"].(string)
			// Adjust url to use custom columns
			columns := "number%2CstartTime%2CstopTime%2Ctriggers%2CrunClassName%2CrunStopReason%2Cbfield%2CgtKey%2Cl1Menu%2ChltKeyDescription%2ClhcFill%2ClhcEnergy%2CrunCreated%2Cmodified%2ClsCount%2ClsRanges"
			if furl[len(furl)-1:] == "/" { // look-up last slash
				furl = fmt.Sprintf("%sapi/GLOBAL/runsummary/json/%s/none/data", furl, columns)
			} else {
				furl = fmt.Sprintf("%s/api/GLOBAL/runsummary/json/%s/none/data", furl, columns)
			}
		} else if system == "reqmgr" || system == "mcm" {
			furl = das.FormRESTUrl(dasquery, dmap)
		} else {
			furl = das.FormUrlCall(dasquery, dmap)
		}
		if furl == "local_api" && !dasmaps.MapInList(dmap, localApis) {
			localApis = append(localApis, dmap)
		} else if furl != "" {
			// adjust conddb URL, remove Runs= empty parater since it leads to an error
			if strings.Contains(furl, "Runs=&") {
				furl = strings.Replace(furl, "Runs=&", "", -1)
			}
			if _, ok := urls[furl]; !ok {
				urls[furl] = args
			}
		}

		srv := fmt.Sprintf("%s:%s", dmap["system"], dmap["urn"])
		srvs = append(srvs, srv)
		lkeys := strings.Split(dmap["lookup"].(string), ",")
		for _, pkey := range lkeys {
			for _, item := range dmap["das_map"].([]interface{}) {
				rec := mongo.Convert2DASRecord(item)
				daskey := rec["das_key"].(string)
				reckey := rec["rec_key"].(string)
				if daskey == pkey {
					pkeys = append(pkeys, reckey)
					break
				}
			}
		}
	}
	if utils.VERBOSE > 0 {
		fmt.Println("### selected services", srvs, pkeys)
		fmt.Println("### selected urls", urls)
		fmt.Println("### selected localApis", localApis)
	}
	// extract selected keys from dasquery and primary keys
	selectKeys, selectSubKeys := selectedKeys(dasquery, pkeys)

	var dasrecords []mongo.DASRecord
	if len(urls) > 0 {
		for _, r := range processURLs(dasquery, urls, maps, &dmaps, pkeys) {
			dasrecords = append(dasrecords, r)
		}
	}
	if len(localApis) > 0 {
		for _, r := range processLocalApis(dasquery, localApis, pkeys) {
			dasrecords = append(dasrecords, r)
		}
	}
	if utils.VERBOSE > 0 {
		fmt.Println("Received", len(dasrecords), "records")
	}

	// check if site query returns nothing and then look-up data in DBS3
	if len(dasrecords) == 0 && utils.InList("site", dasquery.Fields) {
		if !jsonout {
			fmt.Println("WARNING: No site records found in PhEDEx, will look-up original sites in DBS")
		}
		dasquery.System = "dbs3"
		selectedServices = []string{"dbs3"}
		args := ""
		for _, dmap := range maps {
			system, _ := dmap["system"].(string)
			if !utils.InList(system, selectedServices) {
				continue
			}
			furl = das.FormUrlCall(dasquery, dmap)
		}
		if furl != "" {
			if _, ok := urls[furl]; !ok {
				urls[furl] = args
			}
		}
		for _, r := range processURLs(dasquery, urls, maps, &dmaps, pkeys) {
			dasrecords = append(dasrecords, r)
		}
	}

	// if user provides format option we'll add extra fields to be compatible with das_client
	if strings.ToLower(format) == "json" {
		ctime := time.Now().Unix() - time0
		// add status wrapper to be compatible with das_client.py
		fmt.Printf("{\"status\":\"ok\", \"mongo_query\":%s, \"nresults\":%d, \"timestamp\":%d, \"ctime\":%d, \"data\":", dasquery.Marshall(), len(dasrecords), time.Now().Unix(), ctime)
	}

	// if we use detail=True option in json format we'll dump entire dasrecords
	if dasquery.Detail && jsonout {
		fmt.Println("[") // data output goes here
		for idx, rec := range dasrecords {
			if idx < rdx {
				continue
			}
			out, err := json.Marshal(rec)
			if err == nil {
				if idx < len(dasrecords)-1 {
					if limit > 0 && limit == idx+1 {
						fmt.Println(string(out))
						break
					}
					fmt.Println(string(out), ",")
				} else {
					fmt.Println(string(out))
				}
			} else {
				fmt.Println("DAS record", rec, "fail to mashal it to JSON stream")
				os.Exit(1)
			}
		}
		fmt.Println("]") // end of data output
		if strings.ToLower(format) == "json" {
			fmt.Println("}") // end of status wrapper
		}
		os.Exit(0)
	}

	// for non-detailed output, we first get records, then convert them to a set and sort them
	var records []string
	if len(dasquery.Filters) > 0 {
		if strings.ToLower(format) == "json" {
			// TODO: to simplify so far I'll ignore filters since records should be returned as dict
			// but later I need to adjust code to return only filter's attribute
			records = getRecords(dasrecords, selectKeys, selectSubKeys, sep, jsonout)
		} else {
			records = getFilteredRecords(dasquery, dasrecords, sep)
		}
	} else {
		records = getRecords(dasrecords, selectKeys, selectSubKeys, sep, jsonout)
	}
	if unique {
		records = utils.List2Set(records)
		sort.Sort(utils.StringList(records))
	}
	if jsonout {
		fmt.Println("[")
	}
	for idx, rec := range records {
		if idx < rdx {
			continue
		}
		if jsonout {
			if idx < len(records)-1 {
				if limit > 0 && limit == idx+1 {
					fmt.Println(rec)
					break
				}
				fmt.Println(rec, ",")
			} else {
				fmt.Println(rec)
			}
		} else {
			if limit > 0 && limit == idx {
				break
			}
			fmt.Println(rec)
		}
	}
	if jsonout {
		fmt.Println("]")
	}

	if strings.ToLower(format) == "json" {
		fmt.Printf("}")
	}
}

// helper function to extract filtered fields from DAS records
func getFilteredRecords(dasquery dasql.DASQuery, dasrecords []mongo.DASRecord, sep string) []string {
	var records []string
	if dasfilters, ok := dasquery.Filters["grep"]; ok {
		var filterEntries [][]string
		for _, filter := range dasfilters {
			var entries []string
			for idx, val := range strings.Split(filter, ".") {
				entries = append(entries, val)
				if idx == 0 {
					entries = append(entries, "[0]")
				}
			}
			filterEntries = append(filterEntries, entries)
		}
		for _, rec := range dasrecords {
			var out []string
			for _, filters := range filterEntries {
				rbytes, err := mongo.GetBytesFromDASRecord(rec)
				if err != nil {
					fmt.Errorf("Fail to parse DAS record=%v, error=%v\n", rec, err)
				} else {
					val, _, _, err := jsonparser.Get(rbytes, filters...)
					if err != nil {
						fmt.Errorf("Unable to extract filters=%v, error=%v\n", filters, err)
					} else {
						out = append(out, string(val))
					}
				}
				out = append(out, sep)
			}
			if len(out) > 0 {
				records = append(records, strings.Join(out, sep))
			}
		}
	}
	return records
}

// helper function to extract selected keys of DAS queryes from primary keys
func selectedKeys(dasquery dasql.DASQuery, pkeys []string) ([][]string, [][]string) {
	// extract list of select keys we'll need to display on stdout
	var selectKeys, selectSubKeys [][]string
	for _, pkey := range pkeys {
		var skeys []string
		for _, kkk := range strings.Split(pkey, ".") {
			if !utils.InList(kkk, skeys) {
				skeys = append(skeys, kkk)
			}
		}
		selectKeys = append(selectKeys, skeys) // hold [ key attribute ]
		var keys []string
		for _, kkk := range strings.Split(pkey, ".") {
			if !utils.InList(kkk, keys) {
				keys = append(keys, kkk)
				if len(keys) == 1 {
					keys = append(keys, "[0]") // to hadle DAS records lists
				}
			}
		}
		selectSubKeys = append(selectSubKeys, keys) // hold  [key [0] attribute]
	}
	if len(selectKeys) == 0 {
		fmt.Println("Unable to parse DAS query, no select keys are found", dasquery)
		os.Exit(1)
	}
	return selectKeys, selectSubKeys
}

// helper function to print DAS records on stdout
func getRecords(dasrecords []mongo.DASRecord, selectKeys, selectSubKeys [][]string, sep string, jsonout bool) []string {
	var records []string
	for _, rec := range dasrecords {
		das := rec["das"].(mongo.DASRecord)
		pkey := das["primary_key"].(string)
		lkey := strings.Split(pkey, ".")[0]
		skip := false
		switch rrr := rec[lkey].(type) {
		case []mongo.DASRecord:
			for _, r := range rrr {
				recErr := r["error"]
				if recErr != nil {
					skip = true
					if utils.VERBOSE > 0 {
						fmt.Println(recErr)
					}
				}
			}
		case []interface{}:
			for _, r := range rrr {
				v := r.(map[string]interface{})
				recErr := v["error"]
				if recErr != nil {
					skip = true
					if utils.VERBOSE > 0 {
						fmt.Println(recErr)
					}
				}
			}
		}
		if !jsonout && skip {
			continue
		}
		rbytes, err := mongo.GetBytesFromDASRecord(rec)
		if err != nil {
			fmt.Errorf("Fail to parse DAS record=%v, selKeys=%v, error=%v\n", rec, selectKeys, err)
		} else {
			if jsonout {
				out, err := json.Marshal(rec)
				if err != nil {
					fmt.Errorf("Fail to marshal DAS record=%v, error=%v\n", rec, err)
				}
				records = append(records, string(out))
				continue
			}
			var out []string
			for _, keys := range selectKeys {
				val, _, _, err := jsonparser.Get(rbytes, keys...)
				if err == nil {
					sval := string(val)
					if !utils.InList(sval, out) {
						out = append(out, sval)
					}
				} else {
					fmt.Errorf("Fail to parse DAS record=%v, keys=%v, error=%v\n", rec, keys, err)
				}
			}
			if len(out) > 0 {
				records = append(records, strings.Join(out, sep))
			} else { // try out [key [0] attribute]
				for _, keys := range selectSubKeys {
					val, _, _, err := jsonparser.Get(rbytes, keys...)
					if err == nil {
						sval := string(val)
						if !utils.InList(sval, out) {
							out = append(out, sval)
						}
					} else {
						fmt.Errorf("Fail to parse DAS record=%v, keys=%v, error=%v\n", rec, keys, err)
					}
				}
				if len(out) > 0 {
					records = append(records, strings.Join(out, sep))
				}
			}
		}
	}
	return records
}

// helper function to get DAS records out of url response
func response2Records(r *utils.ResponseType, dasquery dasql.DASQuery, maps []mongo.DASRecord, dmaps *dasmaps.DASMaps, pkeys []string) []mongo.DASRecord {
	var dasrecords []mongo.DASRecord
	system := ""
	expire := 0
	urn := ""
	for _, dmap := range maps {
		surl := dasmaps.GetString(dmap, "url")
		// TMP fix, until we fix Phedex data to use JSON
		if strings.Contains(surl, "phedex") {
			surl = strings.Replace(surl, "xml", "json", -1)
		}
		// here we check that request Url match DAS map one either by splitting
		// base from parameters or making a match for REST based urls
		stm := dasmaps.GetString(dmap, "system")
		inst := dasquery.Instance
		if inst != "prod/global" && stm == "dbs3" {
			surl = strings.Replace(surl, "prod/global", inst, -1)
		}
		if strings.Split(r.Url, "?")[0] == surl || strings.HasPrefix(r.Url, surl) || r.Url == surl {
			urn = dasmaps.GetString(dmap, "urn")
			system = dasmaps.GetString(dmap, "system")
			expire = dasmaps.GetInt(dmap, "expire")
		}
	}
	// process data records
	notations := dmaps.FindNotations(system)
	records := services.Unmarshal(dasquery, system, urn, *r, notations, pkeys)
	records = services.AdjustRecords(dasquery, system, urn, records, expire, pkeys)

	// add records
	for _, rec := range records {
		dasrecords = append(dasrecords, rec)
	}
	return dasrecords
}

// helper function to process given set of URLs associted with dasquery
func processURLs(dasquery dasql.DASQuery, urls map[string]string, maps []mongo.DASRecord, dmaps *dasmaps.DASMaps, pkeys []string) []mongo.DASRecord {
	// defer function will propagate panic message to higher level
	//     defer utils.ErrPropagate("processUrls")

	var dasrecords []mongo.DASRecord
	if len(urls) == 1 {
		for furl, args := range urls {
			if dasquery.Detail && strings.Contains(furl, "detail=False") {
				furl = strings.Replace(furl, "detail=False", "detail=True", -1)
			}
			//             if !dasquery.Detail && strings.Contains(furl, "detail=True") {
			//                 furl = strings.Replace(furl, "detail=True", "detail=False", -1)
			//             }
			resp := utils.FetchResponse(furl, args)
			return response2Records(&resp, dasquery, maps, dmaps, pkeys)
		}
	}
	out := make(chan utils.ResponseType)
	defer close(out)
	umap := map[string]int{}
	for furl, args := range urls {
		if dasquery.Detail && strings.Contains(furl, "detail=False") {
			furl = strings.Replace(furl, "detail=False", "detail=True", -1)
		}
		//         if !dasquery.Detail && strings.Contains(furl, "detail=True") {
		//             furl = strings.Replace(furl, "detail=True", "detail=False", -1)
		//         }
		umap[furl] = 1 // keep track of processed urls below
		go utils.Fetch(furl, args, out)
	}

	// collect all results from out channel
	exit := false
	for {
		select {
		case r := <-out:
			for _, rec := range response2Records(&r, dasquery, maps, dmaps, pkeys) {
				dasrecords = append(dasrecords, rec)
			}
			// remove from umap, indicate that we processed it
			delete(umap, r.Url) // remove Url from map
		default:
			if len(umap) == 0 { // no more requests, merge data records
				exit = true
			}
			time.Sleep(time.Duration(10) * time.Millisecond) // wait for response
		}
		if exit {
			break
		}
	}
	return dasrecords
}

// helper function to process given set of URLs associted with dasquery
func processLocalApis(dasquery dasql.DASQuery, dmaps []mongo.DASRecord, pkeys []string) []mongo.DASRecord {
	var dasrecords []mongo.DASRecord
	for _, dmap := range dmaps {
		urn := dasmaps.GetString(dmap, "urn")
		system := dasmaps.GetString(dmap, "system")
		expire := dasmaps.GetInt(dmap, "expire")
		api := fmt.Sprintf("L_%s_%s", system, urn)
		if utils.VERBOSE > 0 {
			fmt.Println("DAS local API", api)
		}
		// we use reflection to look-up api from our services/localapis.go functions
		// for details on reflection see
		// http://stackoverflow.com/questions/12127585/go-lookup-function-by-name
		t := reflect.ValueOf(services.LocalAPIs{})         // type of LocalAPIs struct
		m := t.MethodByName(api)                           // associative function name for given api
		args := []reflect.Value{reflect.ValueOf(dasquery)} // list of function arguments
		vals := m.Call(args)[0]                            // return value
		records := vals.Interface().([]mongo.DASRecord)    // cast reflect value to its type
		if utils.VERBOSE > 0 {
			fmt.Println("### LOCAL APIS", urn, system, expire, dmap, api, m, len(records))
		}

		records = services.AdjustRecords(dasquery, system, urn, records, expire, pkeys)
		// add records
		for _, rec := range records {
			dasrecords = append(dasrecords, rec)
		}
	}
	return dasrecords
}
