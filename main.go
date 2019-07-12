package main

import (
	"bufio"
	"encoding/csv"
	//"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/cloudfoundry-community/go-cfclient"
)

var Cfendpoints []cfEndpoint
var Foundations []foundation
var csvFileName *string
var labelsToQuery *string

// define the foundation and creds
type cfEndpoint struct {
	ApiAddress        string `json:"apiaddress"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	SkipSslValidation bool   `json:"skipSslValidation"`
}

// get services for the foundation
type foundation struct {
	ApiEndpoint      string    `json:"apiEndpoint"`
	ServiceInstance []serviceInstance `json:"serviceInstance"`
}

type serviceInstance struct {
	ServiceName         string `json:"serviceName"`
	ServicePlan         string `json:"servicePlan"`
	ServiceInstanceName string `json:"serviceInstanceName"`
	Org                 string `json:"service"`
	Space               string `json:"service"`
	NumberOfBindings    int    `json:"numberOfBindings"`
}
func ParseConfigFile() {
	csvFile, _ := os.Open(*csvFileName)
	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Fprintln(os.Stderr, error)
		}
		//check for comments (#)
		if 35 != strings.TrimSpace(line[0])[0] {
			b, _ := strconv.ParseBool(line[3])
			Cfendpoints = append(Cfendpoints, cfEndpoint{
				ApiAddress:        line[0],
				Username:          line[1],
				Password:          line[2],
				SkipSslValidation: b,
			})
		}
	}
}

// create the client to do curl commands with
func serviceLabels(apiaddress string, username string, password string, skipsslvalidation bool) {
	fmt.Printf("im here\n")
	c := &cfclient.Config{
		ApiAddress:        apiaddress,
		Username:          username,
		Password:          password,
		SkipSslValidation: skipsslvalidation,
	}
	client, err := cfclient.NewClient(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	services, _ := client.ListServices()
	fmt.Println(apiaddress)
	for _, i := range services {
		fmt.Println("\t", i.Label)
	}
}

func QueryFoundation(apiaddress string, username string, password string, skipsslvalidation bool, labels string) {

	c := &cfclient.Config{
		ApiAddress:        apiaddress,
		Username:          username,
		Password:          password,
		SkipSslValidation: skipsslvalidation,
	}
	client, err := cfclient.NewClient(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	serviceInstances, _ := client.ListServiceInstances()
	for _, i := range serviceInstances {
		service, _ := client.GetServiceByGuid(i.ServiceGuid)
		if "" == labels || strings.LastIndex(labels, service.Label) > -1 {
			v := url.Values{}
			v.Set("q", "service_instance_guid:"+i.Guid)
			serviceBindings, _ := client.ListServiceBindingsByQuery(v)
			servicePlan, _ := client.GetServicePlanByGUID(i.ServicePlanGuid)
			space, _ := client.GetSpaceByGuid(i.SpaceGuid)
			org, _ := client.GetOrgByGuid(space.OrganizationGuid)
			hostname := ""
			username := ""
			port := 0.0
			if service.Label == "redislabs" {
				if len(serviceBindings) > 0 {
					credsMap := serviceBindings[0].Credentials.(map[string]interface{})
					if str, ok := credsMap["host"].(string); ok {
						hostname = str
					}
					if str, ok := credsMap["name"].(string); ok {
						username = str
					}
					if myfloat, ok := credsMap["port"].(float64); ok {
						port = myfloat
					}
				} else {
					//if you get a delete failed, you'll be happy i left these
					//fmt.Printf("else: %v, ", i.ServiceKeysUrl)
					//fmt.Printf("%+v,%v,%v,%v,%v,%v,%v\n", apiaddress, service.Label, servicePlan.Name, i.Name, org.Name, space.Name, len(serviceBindings))
					//fmt.Printf("keysUrl: %v\n", i.ServiceKeysUrl)
					serviceKeysGuid := i.ServiceKeysUrl[22:58]
					//fmt.Printf("else: %v\n", serviceKeysGuid)
					serviceKeys, _ := client.GetServiceKeysByInstanceGuid(serviceKeysGuid)
					if len(serviceKeys) > 0 {
						credsMap := serviceKeys[0].Credentials.(map[string]interface{})
						if str, ok := credsMap["host"].(string); ok {
							hostname = str
						}
						if str, ok := credsMap["name"].(string); ok {
							username = str
						}
						if myfloat, ok := credsMap["port"].(float64); ok {
							port = myfloat
						}
					} else {
						hostname = "ERROR"
						username = "ERROR"
						port = 0.0
					}
				}
			fmt.Printf("%+v,%v,%v,%v,%v,%v,%v,%v,%v,%v\n", apiaddress, service.Label, servicePlan.Name, i.Name, org.Name, space.Name, len(serviceBindings), hostname, username, port)
			} else {
                fmt.Printf("%+v,%v,%v,%v,%v,%v,%v,%v,%v,%v\n", apiaddress, service.Label, servicePlan.Name, i.Name, org.Name, space.Name, len(serviceBindings), hostname, username, port)
            }
        }
	}
}

func main() {
	noheaderPtr := flag.Bool("noheader", false, "Disable printing column headings")
	printServiceLabels := flag.Bool("printServiceLabels", false, "Print deployments and exit")
	csvFileName = flag.String("cfendpoints", "cfendpoints.csv", "csv file that contains: ApiEndpoint, Username, Password, skip-ssl-validation")
	labelsToQuery = flag.String("labels", "redislabs,redislabs-enterprise-cluster", "Only print given service labels. Use comma separated. Do not use comma space separated.")
	flag.Parse()
	ParseConfigFile()
	if *printServiceLabels {
		for _, i := range Cfendpoints {
	fmt.Printf("im here\n")
			serviceLabels(i.ApiAddress, i.Username, i.Password, i.SkipSslValidation)
		}
			os.Exit(0)
	}
	if !*noheaderPtr {
		fmt.Printf("ApiAddress,Service_Label,Serice_Plan,Service_Instance,Org,Space,Number_of_Service_Bindings,Host(redislabs_only),username,port\n")
	}
	//thread this next loop , if you just put go in front of it, it doesn't print, you need to add sleeps or something
	for _, i := range Cfendpoints {
		QueryFoundation(i.ApiAddress, i.Username, i.Password, i.SkipSslValidation, *labelsToQuery)
		//os.Exit(0)
	}
}
