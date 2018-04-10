#!/usr/bin/python3

import subprocess
import json
import sys

#ret dictionary
ret = {}

#foundation collection
foundation = {}
filename = "foundation_collection"
#exec(compile(open(filename, "rb").read(), filename, 'exec'), globals, locals)
exec(open(filename).read())

for myfoundations in foundation:
	#print(myfoundations)
	argsList = list() 
	argsList.append("cf")
	argsList.append("login")
	argsList.append("--skip-ssl-validation")
	argsList.append("-o")
	argsList.append("MichaelIsMetal")
	argsList.append("-u")
	argsList.append("admin")
	argsList.append("-a")
	argsList.append(myfoundations)
	argsList.append("-p")
	argsList.append(foundation[myfoundations])
	#The way the cf cli works is that it uses dot files in the OS to store login state.... your OS has to iterate through cf enpoints - not just the subshell
	subprocess.call(argsList, stdin=None, stdout=None, stderr=None, shell=False)

	#initialize dictionaries
	serviceDictionary = {}
	appServiceDict = {}
	serviceAppDict = {}

	#populate serviceDictionary
	#assumes less than 100 services....
	serviceReturn = json.loads(subprocess.check_output(["cf", "curl", "/v2/services?results-per-page=100"]))
	for i in serviceReturn['resources']:
		serviceDictionary[i["metadata"]["guid"]] = i["entity"]["label"]
	print(serviceDictionary)
	sys.exit(0)

	#populate appServiceDict
	urltocfcurl = "/v2/apps?results-per-page=100"
	while urltocfcurl:
		appReturn = json.loads(subprocess.check_output(["cf", "curl", urltocfcurl]))
		urltocfcurl =  appReturn['next_url']
		for i in appReturn['resources']:
			localList = list()
			appServiceBinding = json.loads(subprocess.check_output(["cf", "curl", i["entity"]["service_bindings_url"]]))
			if appServiceBinding['total_results'] != 0:
				for j in appServiceBinding['resources']:
					myurl = j["entity"]["service_instance_url"]
					#print (myurl)
					myServiceInstance = json.loads(subprocess.check_output(["cf", "curl", myurl]))["entity"]
					if myServiceInstance["type"] == "managed_service_instance":
						localList.append(serviceDictionary[myServiceInstance["service_guid"]])
			appServiceDict[i["entity"]["name"] + "_in_space_" + i["entity"]["space_guid"]] = localList
	#print(json.dumps(appServiceDict))

	#populate serviceAppDict
	for i in appServiceDict:
		#print(appServiceDict[i])
		for j in appServiceDict[i]:
			if j not in serviceAppDict:
				serviceAppDict[j] = list()
			serviceAppDict[j].append(i)
	#print(json.dumps(serviceAppDict))
	ret[myfoundations]=serviceAppDict

#finished! print it
print(json.dumps(ret))
