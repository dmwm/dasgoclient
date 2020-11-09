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
	var aggregate bool
	flag.BoolVar(&aggregate, "aggregate", false, "aggregate results across all data-services")
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "Verbose level, support 0,1,2")
	var examples bool
	flag.BoolVar(&examples, "examples", false, "Show examples of supported DAS queries")
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	var daskeys bool
	flag.BoolVar(&daskeys, "daskeys", false, "Show supported DAS keys")
	var exitCodes bool
	flag.BoolVar(&exitCodes, "exitCodes", false, "Show DAS error codes")
	var unique bool
	flag.BoolVar(&unique, "unique", false, "Sort results and return unique list")
	var timeout int
	flag.IntVar(&timeout, "timeout", 0, "Timeout for url call")
	var urlRetry int
	flag.IntVar(&urlRetry, "urlRetry", 3, "urlRetry for url call")
	var funcProfile string
	flag.StringVar(&funcProfile, "funcProfile", "", "Specify location of function profile file")
	flag.Usage = func() {
		fmt.Println("Usage: dasgoclient [options]")
		flag.PrintDefaults()
		fmt.Println("Examples:")
		fmt.Println("\t# get results")
		fmt.Println("\tdasgoclient -query=\"dataset=/ZMM*/*/*\"")
		fmt.Println("\t# get results in JSON data-format")
		fmt.Println("\tdasgoclient -query=\"dataset=/ZMM*/*/*\" -json")
		fmt.Println("\t# get results from specific CMS data-service, e.g. rucio")
		fmt.Println("\tdasgoclient -query=\"file dataset=/ZMM/Summer11-DESIGN42_V11_428_SLHC1-v1/GEN-SIM system=rucio\" -json")
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
	utils.TLSCertsRenewInterval = 600 * time.Second
	if funcProfile != "" {
		utils.InitFunctionProfiler(funcProfile)
	}
	checkX509()
	if verbose > 0 {
		fmt.Println("DBSUrl: ", services.DBSUrl("prod"))
		fmt.Println("SitedbUrl: ", services.SitedbUrl())
		fmt.Println("CricUrl w/ site API: ", services.CricUrl("site"))
		fmt.Println("RucioUrl: ", services.RucioUrl())
		fmt.Println("RucioAuthUrl: ", utils.RucioAuth.Url())
	}
	if examples {
		showExamples()
	} else if version {
		fmt.Println(info())
	} else if daskeys {
		showDASKeys()
	} else if exitCodes {
		showDASExitCodes()
	} else {
		if strings.Contains(query, "|") && !strings.Contains(query, "detail") {
			// for filters and aggregators we need to use detail=true flag
			query = strings.Replace(query, "|", " detail=true |", 1)
		}
		process(query, jsonout, sep, unique, format, host, idx, limit, aggregate)
	}
}

// helper function to check DAS records and return first error code, otherwise 0
func checkDASrecords(dasrecords []mongo.DASRecord) (int, string) {
	for _, r := range dasrecords {
		das := r["das"].(mongo.DASRecord)
		if das["error"] != nil {
			ecode := das["code"]
			if ecode != nil {
				return ecode.(int), das["error"].(string)
			}
			return utils.DASServerError, utils.DASServerErrorName
		}
		key := das["primary_key"].(string)
		pkey := strings.Split(key, ".")[0]
		rec := r[pkey]
		var records []mongo.DASRecord
		switch v := rec.(type) {
		case []mongo.DASRecord:
			records = v
		case []interface{}:
			for _, r := range v {
				switch a := r.(type) {
				case map[string]interface{}:
					record := make(mongo.DASRecord)
					for k, v := range a {
						record[k] = v
					}
					records = append(records, record)
				}
			}
		}
		for _, v := range records {
			e := v["error"]
			if e != nil {
				ecode := v["code"]
				if ecode != nil {
					return ecode.(int), e.(string)
				}
				return utils.DASServerError, utils.DASServerErrorName
			}
		}
	}
	return 0, ""
}

func showDASExitCodes() {
	fmt.Println("DAS exit codes:")
	fmt.Printf("%v %s\n", utils.DASServerError, utils.DASServerErrorName)
	fmt.Printf("%v %s\n", utils.DBSError, utils.DBSErrorName)
	fmt.Printf("%v %s\n", utils.PhedexError, utils.PhedexErrorName)
	fmt.Printf("%v %s\n", utils.RucioError, utils.RucioErrorName)
	fmt.Printf("%v %s\n", utils.DynamoError, utils.DynamoErrorName)
	fmt.Printf("%v %s\n", utils.ReqMgrError, utils.ReqMgrErrorName)
	fmt.Printf("%v %s\n", utils.RunRegistryError, utils.RunRegistryErrorName)
	fmt.Printf("%v %s\n", utils.McMError, utils.McMErrorName)
	fmt.Printf("%v %s\n", utils.DashboardError, utils.DashboardErrorName)
	fmt.Printf("%v %s\n", utils.SiteDBError, utils.SiteDBErrorName)
	fmt.Printf("%v %s\n", utils.CRICError, utils.CRICErrorName)
	fmt.Printf("%v %s\n", utils.CondDBError, utils.CondDBErrorName)
	fmt.Printf("%v %s\n", utils.CombinedError, utils.CombinedErrorName)
	fmt.Printf("%v %s\n", utils.MongoDBError, utils.MongoDBErrorName)
	fmt.Printf("%v %s\n", utils.DASProxyError, utils.DASProxyErrorName)
	fmt.Printf("%v %s\n", utils.DASQueryError, utils.DASQueryErrorName)
	fmt.Printf("%v %s\n", utils.DASParserError, utils.DASParserErrorName)
	fmt.Printf("%v %s\n", utils.DASValidationError, utils.DASValidationErrorName)
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
		os.Exit(utils.DASProxyError)
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
		"site":    []string{"dbs", "combined", "rucio"},
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
			if !utils.FindInList(system, srvs) {
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
func process(query string, jsonout bool, sep string, unique bool, format, host string, rdx, limit int, aggregate bool) {

	// defer function profiler
	defer utils.MeasureTime("dasgoclient/process")

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
	dasquery, err, posLine := dasql.Parse(query, "", dmaps.DASKeys())
	if utils.VERBOSE > 0 {
		fmt.Println(err)
		fmt.Println(query)
		fmt.Println(posLine)
	}
	if err != "" {
		fmt.Println("ERROR: das parser error:", err)
		os.Exit(utils.DASParserError)
	}
	if e := dasql.ValidateDASQuerySpecs(dasquery); e != nil {
		fmt.Println(e)
		os.Exit(utils.DASValidationError)
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
	var mapServices []string
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
	if len(selectedServices) == 0 || aggregate {
		selectedServices = mapServices
	}

	// if we're not aggregating results
	// use only primary data-service for all requests except site queries
	if !aggregate {
		if len(dasquery.Fields) == 1 && dasquery.Fields[0] != "site" {
			selectedServices = []string{selectedServices[0]}
		}
	}

	// get list of services, pkeys, urls and localApis we need to process
	srvs, pkeys, urls, localApis := das.ProcessLogic(dasquery, maps, selectedServices)

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
		if utils.VERBOSE > 0 {
			fmt.Println("#### processURLs", len(dasrecords))
		}
	}
	if len(localApis) > 0 {
		for _, r := range processLocalApis(dasquery, localApis, pkeys) {
			dasrecords = append(dasrecords, r)
		}
		if utils.VERBOSE > 0 {
			fmt.Println("#### processLocalApis", len(dasrecords))
		}
	}
	if utils.VERBOSE > 0 {
		fmt.Println("Received", len(dasrecords), "records")
	}

	// perform post-processing of DAS records
	//     dasrecords = das.PostProcessing(dasquery, dasrecords)

	// check if site query returns nothing and then look-up data in DBS3
	if len(dasrecords) == 0 && utils.InList("site", dasquery.Fields) {
		if !jsonout {
			fmt.Println("WARNING: No site records found in Rucio, will look-up original sites in DBS")
		}
		if utils.VERBOSE > 0 {
			fmt.Println("### site query returns nothing, will look-up data in DBS")
		}
		dasquery.System = "dbs3"
		selectedServices = []string{"dbs3"}
		args := ""
		furl := ""
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

	ecode, dasError := checkDASrecords(dasrecords)

	// apply aggregation
	aggrs := dasquery.Aggregators
	var out []mongo.DASRecord
	if len(aggrs) > 0 {
		for _, agg := range aggrs {
			fagg := agg[0]
			fval := agg[1]
			if len(dasrecords) > 0 {
				rec := das.Aggregate(dasrecords, fagg, fval)
				out = append(out, rec)
			}
		}
		dasrecords = out
	}

	// if user provides format option we'll add extra fields to be compatible with das_client
	if strings.ToLower(format) == "json" {
		ctime := time.Now().Unix() - time0
		// add status wrapper to be compatible with das_client.py
		fmt.Printf("{\"status\":\"ok\", \"ecode\":\"%s\", \"mongo_query\":%s, \"nresults\":%d, \"timestamp\":%d, \"ctime\":%d, \"data\":", dasError, dasquery.Marshall(), len(dasrecords), time.Now().Unix(), ctime)
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
				fmt.Println("ERROR: DAS record", rec, "fail to marshal it to JSON stream")
				os.Exit(utils.DASServerError)
			}
		}
		fmt.Println("]") // end of data output
		if strings.ToLower(format) == "json" {
			fmt.Println("}") // end of status wrapper
		}
		if ecode != 0 {
			fmt.Println("ERROR: das exit with code:", ecode, ", error:", dasError)
		}
		os.Exit(ecode)
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
	} else if len(dasquery.Aggregators) > 0 {
		if strings.ToLower(format) == "json" {
			records = getRecords(dasrecords, selectKeys, selectSubKeys, sep, jsonout)
		} else {
			records = getAggregatedRecords(dasquery, dasrecords, sep)
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
	if ecode != 0 {
		fmt.Println("ERROR: das exit with code:", ecode, ", error:", dasError)
	}
	os.Exit(ecode)
}

// helper function to extract filtered fields from DAS records
func getFilteredRecords(dasquery dasql.DASQuery, dasrecords []mongo.DASRecord, sep string) []string {

	// defer function profiler
	defer utils.MeasureTime("dasgoclient/getFilteredRecords")

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
					fmt.Printf("Fail to parse DAS record=%+v, error=%v\n", rec, err)
				} else {
					val, _, _, err := jsonparser.Get(rbytes, filters...)
					if err != nil {
						if utils.VERBOSE > 0 {
							fmt.Printf("Unable to extract filters=%v, error=%v\n", filters, err)
						}
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

// helper function to extract aggregated fields from DAS records
func getAggregatedRecords(dasquery dasql.DASQuery, dasrecords []mongo.DASRecord, sep string) []string {

	// defer function profiler
	defer utils.MeasureTime("dasgoclient/getAggregatedRecords")

	var records []string
	for _, rec := range dasrecords {
		var out []string
		for _, agg := range dasquery.Aggregators {
			fagg := agg[0]
			if fagg != rec["function"] {
				continue
			}
			fval := agg[1]
			var aval string
			switch res := rec["result"].(type) {
			case map[string]interface{}:
				aval = fmt.Sprintf("%v", res["value"])
			case mongo.DASRecord:
				aval = fmt.Sprintf("%v", res["value"])
			}
			sval := fmt.Sprintf("%s(%s): %v", fagg, fval, aval)
			out = append(out, sval)
		}
		if len(out) > 0 {
			records = append(records, strings.Join(out, sep))
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
		fmt.Println("ERROR: Unable to parse DAS query, no select keys are found", dasquery)
		os.Exit(utils.DASQueryError)
	}
	return selectKeys, selectSubKeys
}

// helper function to print DAS records on stdout
func getRecords(dasrecords []mongo.DASRecord, selectKeys, selectSubKeys [][]string, sep string, jsonout bool) []string {

	// defer function profiler
	defer utils.MeasureTime("dasgoclient/getRecords")

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
				if r == nil {
					continue
				}
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
			fmt.Printf("Fail to parse DAS record=%v, selKeys=%+v, error=%v\n", rec, selectKeys, err)
		} else {
			if jsonout {
				out, err := json.Marshal(rec)
				if err != nil {
					fmt.Printf("Fail to marshal DAS record=%v, error=%v\n", rec, err)
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
					if utils.VERBOSE > 0 {
						fmt.Printf("Fail to parse DAS record=%+v, keys=%v, error=%v\n", rec, keys, err)
					}
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
						fmt.Printf("Fail to parse DAS record=%+v, keys=%v, error=%v\n", rec, keys, err)
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

	// defer function profiler
	defer utils.MeasureTime("dasgoclient/response2Records")

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

	// defer function profiler
	defer utils.MeasureTime("dasgoclient/processURLs")

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

	// defer function profiler
	defer utils.MeasureTime("dasgoclient/processLocalApis")

	var dasrecords []mongo.DASRecord
	localApiMap := services.LocalAPIMap()
	for _, dmap := range dmaps {
		urn := dasmaps.GetString(dmap, "urn")
		system := dasmaps.GetString(dmap, "system")
		expire := dasmaps.GetInt(dmap, "expire")
		api := fmt.Sprintf("%s_%s", system, urn)
		apiFunc := localApiMap[api]
		if utils.VERBOSE > 0 {
			fmt.Println("DAS local API", api)
		}
		// we use reflection to look-up api from our services/localapis.go functions
		// for details on reflection see
		// http://stackoverflow.com/questions/12127585/go-lookup-function-by-name
		t := reflect.ValueOf(services.LocalAPIs{})         // type of LocalAPIs struct
		m := t.MethodByName(apiFunc)                       // associative function name for given api
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
