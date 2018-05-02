package main

import (
	"io"
	"os"
	"fmt"
	"flag"
	"bufio"
	"strings"
	"net/url"
	"strconv"
	"encoding/csv"
	"encoding/json"
	"github.com/cloudfoundry-community/go-cfclient"
)

var Cfendpoints []cfEndpoint
var Foundations []foundation
var csvFileName string
var labelsToQuery string

type cfEndpoint struct {
	ApiAddress		string			`json:"apiaddress"`
	Username		string			`json:"username"`
	Password		string			`json:"password"`
	SkipSslValidation	bool			`json:"skipSslValidation"`
}
type foundation struct {
	ApiEndpoint		string			`json:"apiEndpoint"`
	Services		[]service		`json:"services"`
}
type service struct {
	Name			string			`'json:"serviceName"`
	ServicePlan		[]servicePlan		`'json:"sericePLan"`
}
type servicePlan struct {
	Name			string			`json:servicePlanName"`
	ServiceInstances	[]serviceInstance	`json:"serviceInstances,omitempty"`
}
type serviceInstance struct {
	Name			string			`json:serviceInstanceName"`
	Credentials		credentials		`json:credentials"`
	App			[]application		`json:app,omitempty"`
}
type credentials struct {
	Host			string			`json:"host,omitempty"`
	User			string			`json:"user,omitempty"`
	Password		string			`json:"password,omitempty"`
	Port			float64			`json:"port,omitempty"`
}
type application struct {
	Name			string			`json:"app"`
	Space			string			`json:"space"`
	Org			string			`json:"org"`
}

func ParseConfigFile() {
	csvFile, _ := os.Open(csvFileName)
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
				ApiAddress:		line[0],
				Username:		line[1],
				Password:		line[2],
				SkipSslValidation:	b,
			})
		}
	}
	//cfendpointsJson, _ := json.Marshal(cfendpoints)
	//fmt.Println(string(cfendpointsJson))
	//fmt.Printf("%+v",cfendpoints)
}
func QueryFoundation(apiaddress string, username string, password string, skipsslvalidation bool) {
	c := &cfclient.Config {
		ApiAddress:		apiaddress,
		Username:		username,
		Password:		password,
		SkipSslValidation:	skipsslvalidation,
	}
	client, err := cfclient.NewClient(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	myFoundation := foundation{
		ApiEndpoint: apiaddress,
	}
	v := url.Values{}
	if labelsToQuery != "" {
		v.Set("q", "label IN " + labelsToQuery)
	}
	//fmt.Println(v.Encode())
//first get the service
	services, _ := client.ListServicesByQuery(v)
	for _, h := range services {
		myService := service{
			Name: h.Label,
		}
		v = url.Values{}
		v.Set("q", "service_guid:" + h.Guid)
//next get the service plans
		servicePlans, _ := client.ListServicePlansByQuery(v)
		for _, i := range servicePlans {
			myServicePlan := servicePlan{}
			myServicePlan.Name = i.Name
			v = url.Values{}
			v.Set("q", "service_plan_guid:" + i.Guid)
//next get service instances
			serviceInstances, _ := client.ListServiceInstancesByQuery(v)
			for _, j := range serviceInstances {
				myServiceInstance := serviceInstance{}
				myServiceInstance.Name = j.Name
//next get service bindings
				v = url.Values{}
				v.Set("q", "service_instance_guid:" + j.Guid)
				serviceBinding, _ := client.ListServiceBindingsByQuery(v)
				//situation where a service broker exists without bindings
				if len(serviceBinding) > 0 {
//finally we print the binding details and iterate the apps
					credsMap := serviceBinding[0].Credentials.(map[string]interface{})
					//fmt.Printf("%+v\n\n", reflect.TypeOf(credsMap["port"]))
					myCredentials := credentials {}
					if str, ok := credsMap["host"].(string); ok {
						myCredentials.Host = str
					}
					if str, ok := credsMap["name"].(string); ok {
						myCredentials.User = str
					}
					if str, ok := credsMap["password"].(string); ok {
						myCredentials.Password = str
					}
					if str, ok := credsMap["port"].(float64); ok {
						myCredentials.Port = str
					}
					for _, k := range serviceBinding {
						app, _ := client.GetAppByGuid(k.AppGuid)
						space, _ := client.GetSpaceByGuid(app.SpaceGuid)
						org, _ := client.GetOrgByGuid(space.OrganizationGuid)
						myApp := application {
							Name: app.Name,
							Space: space.Name,
							Org: org.Name,
						}
						myServiceInstance.App = append(myServiceInstance.App, myApp)
					}
					myServiceInstance.Credentials = myCredentials
				}
				myServicePlan.ServiceInstances = append(myServicePlan.ServiceInstances, myServiceInstance)
			}
			myService.ServicePlan = append(myService.ServicePlan, myServicePlan)
		}
		myFoundation.Services = append(myFoundation.Services, myService)
	}
	Foundations = append(Foundations, myFoundation)
}
func main() {
	noheaderPtr := flag.Bool("noheader", false, "Disable printing column headings")
	jsonPtr := flag.Bool("json", false, "Output is in json")
	csvFileName = *flag.String("cfendpoints", "cfendpoints.csv", "csv file that contains: ApiEndpoint, Username, Password, skip-ssl-validation")
	labelsToQuery = *flag.String("labels", "redislabs,redislabs-enterprise-cluster", "Only query given service labels. Use comma separated. Do not use comma space separated.")
	flag.Parse()
	ParseConfigFile()
	//thread this next loop eventually
	for _, i := range Cfendpoints {
		QueryFoundation(i.ApiAddress, i.Username, i.Password, i.SkipSslValidation)
		//os.Exit(0)
	}
	if *jsonPtr {
		FoundationsJson, _ := json.Marshal(Foundations)
		fmt.Println(string(FoundationsJson))
	} else {
		if !*noheaderPtr {
			fmt.Println("ApiEndpoint, Service, Service Plan, Service Instance, Host, Username, Password, Port, App Name, Space, Org")
		}
		for _, i := range Foundations {
			for _, j := range i.Services {
				for _, k := range j.ServicePlan {
					for _, l := range k.ServiceInstances {
						for _, m := range  l.App {
							fmt.Printf("\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\"\n", i.ApiEndpoint, j.Name, k.Name,  l.Name, l.Credentials.Host, l.Credentials.User, l.Credentials.Password, l.Credentials.Port, m.Name, m.Space, m.Org)
						}
					}
				}
			}
		}
	}
}
