package main

import (
	"io"
	"os"
	"fmt"
	"bufio"
	"net/url"
	"strconv"
	"encoding/csv"
	"encoding/json"
	"github.com/cloudfoundry-community/go-cfclient"
)

var Cfendpoints []cfEndpoint
var Foundations []foundation

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
}
type application struct {
	Name			string			`json:"app"`
	Space			string			`json:"space"`
	Org			string			`json:"org"`
}

func ParseConfigFile() {
	csvFile, _ := os.Open("cfendpoints.csv")
	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			fmt.Fprintln(os.Stderr, error)
		}
		b, _ := strconv.ParseBool(line[3])
		Cfendpoints = append(Cfendpoints, cfEndpoint{
			ApiAddress:		line[0],
			Username:		line[1],
			Password:		line[2],
			SkipSslValidation:	b,
		})
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
	//this label is suspect, different versions of redis have different labels
	//v.Set("q", "label:redislabs")
	//v.Set("q", "label:redislabs-enterprise-cluster")
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
	ParseConfigFile()
	//thread this next loop
	for _, i := range Cfendpoints {
		QueryFoundation(i.ApiAddress, i.Username, i.Password, i.SkipSslValidation)
		//os.Exit(0)
	}
	FoundationsJson, _ := json.Marshal(Foundations)
	fmt.Println(string(FoundationsJson))
}
