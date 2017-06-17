package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/dmwm/das2go/mongo"
	"github.com/dmwm/das2go/utils"
	"github.com/stretchr/testify/assert"
)

var exe string

func init() {
	flag.StringVar(&exe, "exe", "", "dasgoclient executable to run")
	flag.Parse()
	if exe == "" {
		panic("Unable to find dasgoclient executable, please provide as -exe=/path/dasgoclient")
	}
}

// helper function to perform DAS query and capture its output
func runCommand(query string) ([]byte, error) {
	//     path := "/Users/vk/Work/Languages/Go/gopath/src/github.com/dmwm/dasgoclient"
	//     cmd := fmt.Sprintf("%s/dasgoclient", path)
	out, err := exec.Command(exe, "-query", query, "-format", "json").Output()
	return out, err
}

// helper function to return type of given value
func typeof(v interface{}) string {
	return reflect.TypeOf(v).String()
}

func testMsg(msg, query string) string {
	return fmt.Sprintf("%s, %s", msg, query)
}

// Check output of DAS queries
func TestStatus(t *testing.T) {
	assert := assert.New(t)

	examples := []string{"block_queries.txt", "file_queries.txt", "lumi_queries.txt", "mcm_queries.txt", "run_queries.txt", "dataset_queries.txt", "jobsummary_queries.txt", "misc_queries.txt", "site_queries.txt"}
	dasKeys := []string{"expire", "instance", "primary_key", "record", "services"}
	sort.Sort(utils.StringList(dasKeys))
	recKeys := []string{"status", "mongo_query", "nresults", "timestamp", "ctime", "data"}
	sort.Sort(utils.StringList(recKeys))
	var rec mongo.DASRecord
	for _, fname := range examples {
		for _, query := range strings.Split(utils.LoadExamples(fname), "\n") {
			if len(query) > 0 && string(query[0]) != "#" {

				// process DAS query
				fmt.Println("query:", query)
				data, err := runCommand(query)
				assert.NoError(err, testMsg("runCommand", query))

				// get DAS record
				err = json.Unmarshal(data, &rec)
				assert.NoError(err, testMsg("json.Unmarshal", query))

				// test DAS records keys
				keys := utils.MapKeys(rec)
				sort.Sort(utils.StringList(keys))
				assert.Equal(recKeys, keys, testMsg("DAS record keys", query))

				// extract data part of DAS record
				for _, r := range rec["data"].([]interface{}) {
					switch rec := r.(type) {
					case map[string]interface{}: // das map
						for k, v := range rec {
							switch k {
							case "qhash":
								assert.Equal(typeof(v), "string", testMsg("qhash data-type", query))
							case "das":
								val := v.(map[string]interface{})
								dkeys := utils.MapKeys(val)
								sort.Sort(utils.StringList(dkeys))
								assert.Equal(typeof(v), "map[string]interface {}", testMsg("das metadata data-type", query))
								assert.Equal(dkeys, dasKeys, "Test das metadata keys")
							default:
								assert.Equal(typeof(v), "[]interface {}", testMsg("record data-type", query))
							}
						}
					case []interface{}: // empty results
						continue
					default:
						assert.Equal(r, "", testMsg("DAS record data-type", query))
					}
				}
			}
		}
	}
}
