/*
# Author: Pascal Hubacher
# History:
# 2017-08-04 - initial program
#
#
# NAME
# HiCHStorageInfo
# Hitachi Confoederatio Helvetica Storage Information
#
# LICENSE
# Hitachi, Ltd. All rights reserved
# Hitachi Data Systems products and services can be ordered only under the terms and conditions
# of the applicable Hitachi Data Systems agreements.
#
#
# SYNOPSIS
# see HelpOutput function below
#
# DESCRIPTION
#
# OPTIONS
# see "HelpOutput" function below.
#
# NOTES
#
# AUTHOR
#   Pascal Hubacher (pascal.hubacher@hds.com) - August,2017
#
#   xx.0.xx for alpha (status)
#   xx.1.xx for beta (status)
#   xx.2.xx for release candidate
#   xx.3.xx for (final) release

# MODIFICATIONS
#   2017-08-04 - v01.0.02      - initial script
#   2017-08-15 - v01.0.03      - help improved
#   2017-08-22 - v01.0.04      - csv output added, date format(excel friendly)
#   2017-08-22 - v01.0.05      - bug fixes
#   2017-11-01 - v01.0.06      - hcs configuration manager register new storage added
#   2017-11-01 - v01.0.07      - error handling added
#   2018-01-30 - v01.0.08      - physical values added (usedPhysicalCapacityRate,totalPhysicalCapacity,usedPhysicalCapacity,freePhysicalCapacity)
#							   order changed of overall savings. virtual/mapped gb free removed
#   2018-02-18 - v01.0.09      - all functions changed to a parameter struct to easy parameter handling. JSONUnmarshal function introduced to write cleaner code.
#   2018-02-23 - v01.0.10      - applied new rules from maik ernst
#								•	used physical capacity =
#								o	RAIDCOM:   ACT_TP(MB) - ACT_AV(MB)
#								o	REST API:
#									Falls FMC vorhanden = usedPhysicalCapacity (ansonsten gibt es die Variable garnicht)
#									Falls FMC nicht vorhanden = totalPoolCapacity - availableVolumeCapacity  (Achtung!!! – Falls FMC vorhanden ist die Wert falsch und zeigt den virtuellen freien Speicher an)
#								•	free physical capacity =
#								o	RAIDCOM: ACT_AV(MB)
#								o	REST API:
#									Falls FMC vorhanden = availablePhysicalVolumeCapacity (ansonsten gibt es die Variable garnicht)
#									Falls FMC nicht vorhanden = availableVolumeCapacity (Achtung!!! – Falls FMC vorhanden ist die Wert falsch und zeigt den virtuellen freien Speicher an)
#								•	total physical capacity   =
#								o	RAIDCOM: ACT_TP(MB)
#								o	REST API:
#									Falls FMC vorhanden = totalPhysicalCapacity (ansonsten gibt es die Variable garnicht)
#									Falls FMC nicht vorhanden = totalPoolCapacity (Achtung!!! – Falls FMC vorhanden ist die Wert falsch und zeigt den virtuellen freien Speicher an)
#								•	compression ratio FMC =
#								o	RAIDCOM: FMC_LOG_USED(BLK) / FMC_ACT_USED(BLK)
#								o	REST API:
#									Falls FMC vorhanden =  usedFMCPoolVolumesCapacity / usedPhysicalFMCPoolVolumesCapacity (ansonsten gibt es die Variablen garnicht)
#								•	compression ratio TOTAL =
#								o	RAIDCOM: ( TP_CAP(MB) - AV_CAP(MB) ) / ( ACT_TP(MB) - ACT_AV(MB) )
#								o	REST API:
#									Falls FMC vorhanden =   (totalPoolCapacity  - availablePhysicalVolumeCapacity ) / usedPhysicalCapacity (ansonsten gibt es die Variable “usedPhysicalCapacity” garnicht)
#   2018-03-15 - v01.0.11      - applied FMC Compression Ratio requested by maik ernst
#   2018-05-18 - v01.0.12      - added trace option (-trace) to get all output (pascal hubacher)
#   2018-05-25 - v01.0.13      - BUG: Check if 'availablePhysicalVolumeCapacity' exists. If not take 'availableVolumeCapacity' (pascal hubacher)
#   2018-06-01 - v01.0.14      - BUG: Line 746 : Value was divided by mistake by Mb2Gb (1024). The Mb2Gb was removed. (pascal hubacher)
#   2018-06-15 - v01.0.15      - BUG: Version control was wrong. (pascal hubacher)
#								 Change: Ordering of the output changed to show the 'Compression ratio FMC' closer to the physical values.
#								         And the 'Compression ratio total' closer to the Effective total GB free. (roman siegenthaler)
#								 Change: Name changed from 'Effective GB free [GB]' to 'Effective total GB free [GB]' (roman siegenthaler)
#
*/

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

var (
	//output
	output string
	//Token acts as a security token
	Token string
	//StorageDeviceID of the Storage System
	StorageDeviceID string
	//SessionID of the security token
	SessionID float64

	//Parameters this type contains all parameters needed
	Parameters Params

	//State boolean status to see if an error happened in a function
	//false -> OK, true -> NOS
	State bool

	//Verbose Logger
	Verbose *log.Logger
	//Debug Logger
	Debug *log.Logger
	//Info Logger
	Info *log.Logger
	//Warning Logger
	Warning *log.Logger
	//Error Logger
	Error *log.Logger
)

//Params type is used for all request related parameters
type Params struct {
	Protocol        string
	Port            string
	Host            string
	RequestType     string
	URL             string
	RequestBody     string
	Username        string
	Password        string
	Token           string
	StorageDeviceID string
	SessionID       float64

	OutputStyle        string
	OutputType         string
	ElementStringStart string
	ElementStringEnd   string
	RoundPrecision     int
	MaxElementCount    int64
	RestVersion        string
	APIVersionElement  string
	DataElement        string
	CSVString          string
}

//PoolInfo type is used for all Pool Element actions
type PoolInfo struct {
	PoolID                          string
	PoolType                        string
	PoolName                        string
	usedPhysicalCapacityRate        string
	totalPhysicalCapacity           string
	usedPhysicalCapacity            string
	availablePhysicalVolumeCapacity string
	availableVolumeCapacity         string
	OverallSavings                  string
	FMCCompressionRatio             string
	PhysFMCPoolVolCapTotal          string
	PhysFMCPoolVolCapFree           string
	PhysFMCPoolVolCapUsed           string
	EffectiveGBFree                 string
	CompressionRatioTotal           string
}

//Init is used to initialize the logging
func Init(
	debugHandle io.Writer,
	verboseHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Verbose = log.New(verboseHandle,
		"VERBOSE: ",
		log.Ldate|log.Ltime)

	Debug = log.New(debugHandle,
		"DEBUG  : ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO   : ",
		log.Ldate|log.Ltime)

	Warning = log.New(warningHandle,
		"WARNING : ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR  : ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {

	//defaults
	//Version of the script
	const Version string = "01.00.16"

	//output styles
	const OutputTypeStdout string = "stdout"
	const OutputTypeCsv string = "csv"
	// Minimum Version to be able to run the script
	const VersionMinimum string = "1.5.0"

	//only for testing purpose
	//activate debug mode
	//true -> debug mode, false -> no debug mode
	const DebugMode = false
	//const DebugMode = true

	//initial state
	State = false

	//command line options
	HostPtr := flag.String("host", "localhost", "host to send request to. (Optional)")
	//ProtocolPtr := flag.String("protocol", "https", "protocol to use to send RestAPI requests. (Optional)")
	PortPtr := flag.String("port", "443", "Port to be used to contact the host. The storage RestAPI uses 443 (https). The HCS Rest API uses 23451. (Optional)")
	UserPtr := flag.String("user", "", "User you want to use to contact. (Required)")
	PasswordPtr := flag.String("password", "", "Password you want to use to contact. (Required)")
	OutputPtr := flag.String("output", "stdout", "Specify the way you want to send the output to. Options are 'stdout' or 'csv'. 'stdout' sends the output to the command line. 'csv' create a file containing comma separated data. (Optional)")
	TypePtr := flag.String("type", "pool", "Sets the type of output you want. 'pool' get all pool data. 'reserve' gets you all LUNs/LDEVs that have a reserve. (Optional)")
	VerbosePtr := flag.Bool("verbose", false, "Sets the output mode to verbose. (Optional)")
	HelpPtr := flag.Bool("h", false, "Shows the help. (Optional)")
	TracePtr := flag.Bool("trace", false, "Shows all output for tracing.")
	flag.Parse()

	//--- As the input arguments are needed to specify what to be output this has to be done here.
	//Initialize the Logging start
	if DebugMode { //show trace and debug logging in standard out
		Init(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stdout)
	} else {
		if *VerbosePtr { //show trace logging in standard out
			Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stdout, os.Stdout)
		} else {
			//discard all standard out logging if csv is set. show only data
			if *OutputPtr == "csv" {
				Init(ioutil.Discard, ioutil.Discard, ioutil.Discard, os.Stderr, os.Stderr)
			} else {
				if *TracePtr {
					//discard no messages in standard out
					Init(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stderr)
				} else {
					//discard only debug and trace messages in standard out
					Init(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
				}
			}
		}
	}
	//Initialize the Logging end

	// overwrite standard command line usage output to custom help output
	flag.Usage = func() {
		HelpOutput(Version)
		//os.Exit(0)
	}

	//no flag set
	if flag.NFlag() == 0 {
		//Dispaly the help output
		Debug.Println("No argument specified.")
		flag.Usage()
		os.Exit(0)
	}

	//help flag set -h --h
	if *HelpPtr {
		//Dispaly the help output
		//Debug.Println("-h or --h argument specified.")
		flag.Usage()
		os.Exit(0)
	}

	if *UserPtr == "" {
		//Message what to do
		fmt.Println()
		fmt.Println("You must specify a user for your request")
		fmt.Println()

		//Dispaly the help output
		flag.Usage()
		os.Exit(1)
	}

	if *PasswordPtr == "" {
		//Message what to do
		fmt.Println()
		fmt.Println("You must specify a password for your request")
		fmt.Println()

		//Dispaly the help output
		flag.Usage()
		os.Exit(1)
	}

	//check the type values if they are correct
	if (*OutputPtr != "stdout") && (*OutputPtr != "csv") {
		//throw an error an strop the program
		Warning.Println("The output type you specified is not valid. Please specify 'stdout' or 'csv'. No action will take place.")
		os.Exit(1)
	}

	//check the type values if they are correct
	if (*TypePtr != "reserve") && (*TypePtr != "pool") {
		//throw an error an strop the program
		Warning.Println("The type you specified is not valid. Please specify 'pool' or 'reserve'. No action will take place.")
		os.Exit(1)
	}

	///////////////////////////
	//Specify the paramters
	///////////////////////////
	//version
	Parameters.RestVersion = ""
	//All values are output with 2 decimal digits after the dot
	Parameters.RoundPrecision = 2
	//This is the maximum number of elements that the REST API can return
	Parameters.MaxElementCount = 16348

	//These are constants to specify the start and end of a row and the table to easy th output creation as table and csv
	Parameters.ElementStringStart = "Lacsap-Hitachi-Start"
	Parameters.ElementStringEnd = "Lacsap-Hitachi-End"

	//CSV separator string
	Parameters.CSVString = ","

	//some data output are ind key -> "data" value -> "data output slice"
	//to check if you have such a response the data pattern is needed
	Parameters.DataElement = "data"

	//Web request info
	Parameters.Protocol = "https"
	Parameters.Username = *UserPtr
	Parameters.Password = *PasswordPtr
	Parameters.Host = *HostPtr
	Parameters.Port = *PortPtr
	Parameters.URL = ""
	Parameters.RequestType = ""
	Parameters.RequestBody = ""
	Parameters.OutputStyle = *OutputPtr
	Parameters.OutputType = *TypePtr
	Parameters.Token = ""
	Parameters.StorageDeviceID = ""
	Parameters.SessionID = 0.0

	/*
		//hcs rest api
		Protocol = "http"
		*PortPtr = "23450"
		*UserPtr = "maintenance"
		*PasswordPtr = "password"
		*HostPtr = "10.1.1.1"
	*/

	/*
		G600
		//storage api
		//Protocol = "https"
		//*PortPtr = "443"
		*UserPtr = "maintenance"
		*PasswordPtr = "password"
		*HostPtr = "10.1.1.1"
	*/

	/*
		G1000
		*PortPtr = "443"
		*UserPtr = "maintenance"
		*PasswordPtr = "password"
		*HostPtr = "10.1.1.1"
	*/
	//*VerbosePtr = "y"

	// Output of command line
	Info.Println("HiCHPoolInfo version: " + Version)

	//---------------------------
	//Start execute commands

	//pool type is default
	if *TypePtr == "pool" {
		//Get pool information

		//get the RestAPI version
		Parameters.RestVersion, State = StorageRestAPIVersionGet(Parameters)

		//check if version is ok
		State = CheckVersion(Parameters.RestVersion, VersionMinimum)

		//Get the StorageDeviceID
		//Verbose.Println("Get the StorageDeviceID")
		Parameters.StorageDeviceID, State = StorageDeviceIDGet(Parameters)

		//Create a sesseion
		//Verbose.Println("Create a Session")
		Parameters.Token, Parameters.SessionID, State = TokenGet(Parameters)

		//Verbose.Println("Get Pool information")
		output, State = PoolsGet(Parameters)

		//Delete the session
		//Verbose.Println("Delete the Session")
		output, State = TokenDelete(Parameters)
	}

	//reserve type
	if *TypePtr == "reserve" {
		//Get LUN reservation information

		//Get the StorageDeviceID
		//Verbose.Println("Get the StorageDeviceID")
		Parameters.StorageDeviceID, State = StorageDeviceIDGet(Parameters)

		//Create a sesseion
		//Verbose.Println("Create a Session")
		Parameters.Token, Parameters.SessionID, State = TokenGet(Parameters)

		output, State = LunsGetReserve(Parameters)

		//Delete the session
		//Verbose.Println("Delete the Session")
		output, State = TokenDelete(Parameters)

	}

	//Stop execute commands
	//---------------------------

}

//LunsGetReserve shows all LUNs/LDEVs that have a reserve
//return value (string) is empty. if an error happened the state is true. Otherwise false.
func LunsGetReserve(p Params) (string, bool) {
	Debug.Println("Function 'LunsGetReserve' started.")
	//start timer
	TimeStart := time.Now()

	//initial state is true that means NOK
	State := true

	var Out string

	//GET base-URL/v1/objects/storages/storage-device-ID/luns
	var ConfigurationManagerString string
	ConfigurationManagerString = "/ConfigurationManager/v1/objects/storages/"
	var PostString string
	PostString = "/host-groups?count=" + strconv.FormatInt(p.MaxElementCount, 10)

	Verbose.Println("Get general information of all HostGroups")

	/*
	   http://10.70.4.145/ConfigurationManager/v1/objects/storages/800000058068/host-groups?count=16384
	   {
	       "data": [{
	           "hostGroupId": "CL1-A,0",
	           "portId": "CL1-A",
	           "hostGroupNumber": 0,
	           "hostGroupName": "1A-G00",
	           "hostMode": "LINUX/IRIX"
	       }, {

	*/

	p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + ConfigurationManagerString + p.StorageDeviceID + PostString
	Debug.Println(p.URL)
	p.RequestType = "GET"
	Out = HTTPRequest(p)

	//is the string "data" in the output
	if CheckIsInString(Out, p.DataElement) {
		Error.Println(Out)
	}

	var JSONUnmarshalOut map[string]interface{}
	JSONUnmarshalOut, State = JSONUnmarshal(Out)
	Debug.Println("JSON Unmarshal:", JSONUnmarshalOut)

	// all responses from hitachi rest api calls answer with only one element called data "{ "data": [{"
	if len(JSONUnmarshalOut) != 1 {
		//this should never happens
		//at hitachi the response always starts with "{ "data": [{"
		Error.Println("JSON parsing error (Return Format is not correct).")
		os.Exit(41)
	}

	Debug.Println("Number of HostGroups", len(JSONUnmarshalOut["data"].([]interface{})))
	for key1, hostgroup := range JSONUnmarshalOut["data"].([]interface{}) {
		//Verbose.Println("Key: "+strconv.Itoa(key1), "Value: ", hostgroup)
		ParsedHostGroupMap := hostgroup.(map[string]interface{})

		// HostGroup Element
		Debug.Println("HostGroup Element: ", key1)
		Info.Println("Get the HostGroup Information: " + ParsedHostGroupMap["portId"].(string) + " " + ParsedHostGroupMap["hostGroupName"].(string) + "(" + strconv.FormatFloat(ParsedHostGroupMap["hostGroupNumber"].(float64), 'f', 0, 64) + ")")

		//------------------------------------
		//get lun information
		p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + ConfigurationManagerString + p.StorageDeviceID + "/luns?portId=" + ParsedHostGroupMap["portId"].(string) + "&hostGroupNumber=" + strconv.FormatFloat(ParsedHostGroupMap["hostGroupNumber"].(float64), 'f', 0, 64)
		Debug.Println(p.URL)
		p.RequestType = "GET"
		Out = HTTPRequest(p)

		//is the string "data" in the output
		if CheckIsInString(Out, p.DataElement) {
			Error.Println(Out)
		}

		var JSONUnmarshalOut1 map[string]interface{}
		JSONUnmarshalOut1, State = JSONUnmarshal(Out)
		Debug.Println("JSON Unmarshal:", JSONUnmarshalOut1)

		// all responses from hitachi rest api calls answer with only one element called data "{ "data": [{"
		if len(JSONUnmarshalOut1) != 1 {
			//this should never happens
			//at hitachi the response always starts with "{ "data": [{"
			Error.Println("JSON parsing error (Return Format is not correct).")
			os.Exit(41)
		}

		Debug.Println("Number of LUNs", len(JSONUnmarshalOut1["data"].([]interface{})))
		for _, luns := range JSONUnmarshalOut1["data"].([]interface{}) {
			//Verbose.Println("Key: "+strconv.Itoa(key1), "Value: ", value1)
			ParsedLunsMap := luns.(map[string]interface{})

			/*
			   http://10.70.4.145/ConfigurationManager/v1/objects/storages/800000058068/luns?portId=CL1-B&hostGroupNumber=1

			   				"data": [{
			   			"lunId": "CL1-B,1,1",
			   			"portId": "CL1-B",
			   			"hostGroupNumber": 1,
			   			"hostMode": "WIN_EX",
			   			"lun": 1,
			   			"ldevId": 13312,
			   			"isCommandDevice": false,
			   			"luHostReserve": {
			   				"openSystem": false,
			   				"persistent": false,
			   				"pgrKey": false,
			   				"mainframe": false,
			   				"acaReserve": false
			   			},
			   			"hostModeOptions": [40, 73]
			*/

			// LUN Element
			//Verbose.Println("LUN Element: ", key2)

			// Loop over the "luHostReserve" map.
			ReserveString := ""
			ReserveSet := false

			//map[persistent:false pgrKey:false mainframe:false acaReserve:false openSystem:false]
			for key3, reserve := range ParsedLunsMap["luHostReserve"].(map[string]interface{}) {
				Debug.Println("luHostReserve Element: ", key3)
				if reserve.(bool) {
					ReserveSet = true
					if ReserveString == "" {
						ReserveString = "(" + key3 + "=" + strconv.FormatBool(reserve.(bool))
					} else {
						ReserveString = ReserveString + "; " + key3 + "=" + strconv.FormatBool(reserve.(bool))
					}
				}
			}
			ReserveString = ReserveString + ")"

			var LdevString string
			//format float64 to string in hex
			LdevString = strconv.FormatInt(int64(ParsedLunsMap["ldevId"].(float64)), 16)
			//add leading zeros until it is 4 digits long
			if len(LdevString) < 4 {
				Len := 4 - len(LdevString)
				for i := 0; i < Len; i++ {
					LdevString = "0" + LdevString
				}
			}

			// format the hex string to xx:xx
			LdevString = LdevString[:len(LdevString)-2] + ":" + LdevString[len(LdevString)-2:]

			if ReserveSet {
				//reservations set
				Info.Printf("LUN: %04.f LDEV: %s reservations: %s", ParsedLunsMap["lun"].(float64), LdevString, ReserveString)
			} else {
				//No reservations set
				Info.Printf("LUN: %04.f LDEV: %s reservations: none", ParsedLunsMap["lun"].(float64), LdevString)
			}
		}

	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'LunsGetReserve' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'LunsGetReserve' return values State:", State)
	Debug.Println("Function 'LunsGetReserve' end")

	Verbose.Println("Get the LUNs Information end")

	//state to OK
	State = false
	//return the choosen "storageDeviceId" and its state
	return "", State
}

//PoolsGet is used to get all pool information
//return value (string) is the version used and the status of the request. if an error happened the state is true. Otherwise false.
//The function stops with exit status 40 ("JSON parsing error (Unmarshal function threw an error).")
//The function stops with exit status 41 ("JSON parsing error (Return Format is not correct).")
//example: StorageRestAPIVersionGet("https", "443", "10.0.0.1", "c0492b4c-165d-4052-87e2-27053023e29f", "834000470018")
func PoolsGet(p Params) (string, bool) {
	Debug.Println("Function 'PoolsGet' start.")
	//start timer
	TimeStart := time.Now()

	//initial state is true that means NOK
	State := true

	var Out string
	Out = ""

	var Mb2Gb float64
	Mb2Gb = 1024.0

	//PoolInfo this type contains all parameters needed
	var PoolElement PoolInfo

	//Array to get the mapped and the used capacity of all ldevs of a pool
	//var LdevMappedUsedArray [2]float64

	var ConfigurationManagerString string
	ConfigurationManagerString = "/ConfigurationManager/v1/objects/storages/"
	var PoolsString string
	PoolsString = "/pools?detailInfoType=FMC"

	Info.Println("Get Pool information start")
	Verbose.Println("Get general information of all Pools start")

	p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + ConfigurationManagerString + p.StorageDeviceID + PoolsString
	Debug.Println(p.URL)
	p.RequestType = "GET"
	Out = HTTPRequest(p)

	//Testdata
	//Out = `{    "data": [{        "poolId": 0,        "poolStatus": "POLN",        "usedCapacityRate": 76,        "poolName": "FMD_Pool",        "availableVolumeCapacity": 2808204,        "totalPoolCapacity": 11739672,        "numOfLdevs": 8,        "firstLdevId": 3840,        "warningThreshold": 80,        "depletionThreshold": 90,        "virtualVolumeCapacityRate": -1,        "isMainframe": false,        "isShrinking": false,        "locatedVolumeCount": 20,        "totalLocatedCapacity": 12391712,        "blockingMode": "NB",        "totalReservedCapacity": 0,        "reservedVolumeCount": 0,        "poolType": "HDP",        "duplicationNumber": 0,        "dataReductionAccelerateCompCapacity": 0,        "dataReductionCapacity": 0,        "dataReductionBeforeCapacity": 0,        "dataReductionAccelerateCompRate": 0,        "duplicationRate": 0,        "compressionRate": 0,        "dataReductionRate": 0    }, {        "poolId": 1,        "poolStatus": "POLN",        "usedCapacityRate": 0,        "poolName": "Test_Comp",        "availableVolumeCapacity": 6287232,        "totalPoolCapacity": 6287232,        "numOfLdevs": 4,        "firstLdevId": 512,        "warningThreshold": 70,        "depletionThreshold": 80,        "virtualVolumeCapacityRate": -1,        "isMainframe": false,        "isShrinking": false,        "locatedVolumeCount": 1,        "totalLocatedCapacity": 102568,        "blockingMode": "NB",        "totalReservedCapacity": 0,        "reservedVolumeCount": 0,        "poolType": "HDP",        "duplicationNumber": 0,        "dataReductionAccelerateCompCapacity": 0,        "dataReductionCapacity": 0,        "dataReductionBeforeCapacity": 0,        "dataReductionAccelerateCompRate": 0,        "duplicationRate": 0,        "compressionRate": 0,        "dataReductionRate": 0    }, {        "poolId": 20,        "poolStatus": "POLN",        "usedCapacityRate": 12,        "usedPhysicalCapacityRate": 6,        "poolName": "FMC_HDP",        "availableVolumeCapacity": 8838690,        "availablePhysicalVolumeCapacity": 4593750,        "usedPhysicalCapacity": 316512,        "totalPoolCapacity": 10062024,        "totalPhysicalCapacity": 4910262,        "numOfLdevs": 8,        "firstLdevId": 2560,        "warningThreshold": 99,        "depletionThreshold": 99,        "virtualVolumeCapacityRate": -1,        "isMainframe": false,        "isShrinking": false,        "locatedVolumeCount": 6,        "totalLocatedCapacity": 6292464,        "blockingMode": "NB",        "totalReservedCapacity": 0,        "reservedVolumeCount": 0,        "poolType": "HDP",        "duplicationNumber": 0,        "dataReductionAccelerateCompCapacity": 1857198150,        "dataReductionCapacity": 0,        "dataReductionBeforeCapacity": 0,        "dataReductionAccelerateCompRate": 74,        "duplicationRate": 0,        "compressionRate": 74,        "dataReductionRate": 0,        "availablePhysicalFMCPoolVolumesCapacity": 4910262,        "usedPhysicalFMCPoolVolumesCapacity": 316498,        "availableFMCPoolVolumesCapacity": 10062024,        "usedFMCPoolVolumesCapacity": 1223334,        "fmcPoolVolumesCapacitySaving": 906835,        "fmcPoolVolumesCapacitySavingRate": 74,        "fmcPoolVolumesCapacityExpansionRate": 204    }, {        "poolId": 22,        "poolStatus": "POLN",        "usedCapacityRate": 3,        "usedPhysicalCapacityRate": 1,        "poolName": "FMC_HDT",        "availableVolumeCapacity": 12181134,        "availablePhysicalVolumeCapacity": 7324422,        "usedPhysicalCapacity": 80136,        "totalPoolCapacity": 12560520,        "totalPhysicalCapacity": 7404558,        "numOfLdevs": 10,        "firstLdevId": 1792,        "warningThreshold": 99,        "depletionThreshold": 99,        "virtualVolumeCapacityRate": -1,        "isMainframe": false,        "isShrinking": false,        "locatedVolumeCount": 6,        "totalLocatedCapacity": 6292464,        "blockingMode": "NB",        "totalReservedCapacity": 0,        "reservedVolumeCount": 0,        "poolActionMode": "AUT",        "tierOperationStatus": "MON",        "dat": "VAL",        "poolType": "RT",        "monitoringMode": "CM",        "tiers": [{            "tierNumber": 1,            "tierLevelRange": "00000000",            "tierDeltaRange": "00000000",            "tierUsedCapacity": 379386,            "tierTotalCapacity": 10066224,            "tablespaceRate": 0,            "performanceRate": 0,            "progressOfReplacing": 100,            "bufferRate": 2        }, {            "tierNumber": 2,            "tierLevelRange": "00000000",            "tierDeltaRange": "00000000",            "tierUsedCapacity": 0,            "tierTotalCapacity": 2494296,            "tablespaceRate": 8,            "performanceRate": 0,            "progressOfReplacing": 100,            "bufferRate": 2        }],        "duplicationNumber": 0,        "dataReductionAccelerateCompCapacity": 612709578,        "dataReductionCapacity": 0,        "dataReductionBeforeCapacity": 0,        "dataReductionAccelerateCompRate": 78,        "duplicationRate": 0,        "compressionRate": 78,        "dataReductionRate": 0,        "availablePhysicalFMCPoolVolumesCapacity": 4910262,        "usedPhysicalFMCPoolVolumesCapacity": 80118,        "availableFMCPoolVolumesCapacity": 10066224,        "usedFMCPoolVolumesCapacity": 379293,        "fmcPoolVolumesCapacitySaving": 299174,        "fmcPoolVolumesCapacitySavingRate": 78,        "fmcPoolVolumesCapacityExpansionRate": 205    }]}`
	//Verbose.Println("RAW data: " + Out)

	//is the key string "data" in the output
	if CheckIsInString(Out, p.DataElement) {
		Error.Println(Out)
	}

	var JSONUnmarshalOut map[string]interface{}
	JSONUnmarshalOut, State = JSONUnmarshal(Out)
	Debug.Println("JSON Unmarshal:", JSONUnmarshalOut)

	// all responses from hitachi rest api calls answer with only one element called data "{ "data": [{"
	if len(JSONUnmarshalOut) != 1 {
		//this should never happens
		//at hitachi the response always starts with "{ "data": [{"
		Error.Println("JSON parsing error (Return Format is not correct).")
		os.Exit(41)
	}

	Debug.Println("Number of Pools", len(JSONUnmarshalOut["data"].([]interface{})))
	Verbose.Println("Get general information of all Pools end")

	//add empty string of strings to collect all pool data to output
	OutData := [][]string{}

	for Key1, Value1 := range JSONUnmarshalOut["data"].([]interface{}) {
		//Verbose.Println("Key: "+strconv.Itoa(Key1), "Value: ", Value1)
		ParsedMap := Value1.(map[string]interface{})

		// Pool element
		Debug.Println("Pool Element: ", Key1)
		Debug.Println("Get the Pool Information of Pool: " + strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64) + " (" + ParsedMap["poolName"].(string) + ")")

		// get the mapped and used capactiy
		//LdevMappedUsedArray[0] -> mapped capacity
		//LdevMappedUsedArray[1] -> used capacity
		//Verbose.Println("Get mapped LUN Information of Pool: " + strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64) + " (" + ParsedMap["poolName"].(string) + ")")
		//LdevMappedUsedArray, State = LdevCapSumGet(p, ParsedMap["poolId"].(float64))
		//Verbose.Println("Get mapped LUN Information of Pool: " + strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64) + " (" + ParsedMap["poolName"].(string) + ") completed")
		//if State {
		//	Verbose.Println(LdevMappedUsedArray)
		//}

		//check if the element "availablePhysicalVolumeCapacity" is existent. Then use it or use the element "availableVolumeCapacity"
		var availablePhysicalVolumeCapacity float64
		availablePhysicalVolumeCapacity = 0
		if ParsedMap["availablePhysicalVolumeCapacity"] != nil {
			availablePhysicalVolumeCapacity = ParsedMap["availablePhysicalVolumeCapacity"].(float64)
			Debug.Println("Element: 'availablePhysicalVolumeCapacity' (" + strconv.FormatFloat(availablePhysicalVolumeCapacity/Mb2Gb, 'f', p.RoundPrecision, 64) + "MiB) exists.")
		} else {
			availablePhysicalVolumeCapacity = ParsedMap["availableVolumeCapacity"].(float64)
			Debug.Println("Element: 'availablePhysicalVolumeCapacity' does not exist. Took 'availableVolumeCapacity' (" + strconv.FormatFloat(availablePhysicalVolumeCapacity/Mb2Gb, 'f', p.RoundPrecision, 64) + "MiB) instead.")
		}

		//is it a pool containing FMC?
		if ParsedMap["usedFMCPoolVolumesCapacity"] != nil {

			//-------------------------------------
			// FMC containing disk pool
			//-------------------------------------

			//is it FMC only?
			//"totalPhysicalCapacity" =  "availablePhysicalFMCPoolVolumesCapacity" then it is an all flash pool
			if ParsedMap["totalPhysicalCapacity"].(float64) == ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64) {

				//-------------------------------------
				// FMC Only disk pool (All flash pool)
				//-------------------------------------

				Verbose.Println("All FMC Pool (All flash Pool)")

				// Pool ID
				PoolElement.PoolID = strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64)

				//PoolElement.PoolType = ParsedMap["poolType"].(string)

				//Pool Name
				PoolElement.PoolName = ParsedMap["poolName"].(string)

				//Physical Capacity
				//Used(%)
				//PoolElement.usedPhysicalCapacityRate = strconv.FormatFloat(ParsedMap["usedPhysicalCapacityRate"].(float64), 'f', p.RoundPrecision, 64)
				//Total
				PoolElement.totalPhysicalCapacity = strconv.FormatFloat(ParsedMap["totalPhysicalCapacity"].(float64)/Mb2Gb, 'f', p.RoundPrecision, 64)
				//Used
				PoolElement.usedPhysicalCapacity = strconv.FormatFloat(ParsedMap["usedPhysicalCapacity"].(float64)/Mb2Gb, 'f', p.RoundPrecision, 64)
				//Free
				PoolElement.availablePhysicalVolumeCapacity = strconv.FormatFloat(availablePhysicalVolumeCapacity/Mb2Gb, 'f', p.RoundPrecision, 64)

				//Compression Ratio Total =   (totalPoolCapacity  - availablePhysicalVolumeCapacity ) / usedPhysicalCapacity
				var CompressionRatioTotal float64
				CompressionRatioTotal = (ParsedMap["totalPoolCapacity"].(float64) - availablePhysicalVolumeCapacity) / ParsedMap["usedPhysicalCapacity"].(float64)
				PoolElement.CompressionRatioTotal = strconv.FormatFloat(CompressionRatioTotal, 'f', p.RoundPrecision, 64)

				//PoolElement.OverallSavings = strconv.FormatFloat(((1 - (ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64) / LdevMappedUsedArray[0])) * 100), 'f', p.RoundPrecision, 64)

				//FMC Only Values
				//FMC Compression Ratio =  usedFMCPoolVolumesCapacity / usedPhysicalFMCPoolVolumesCapacity
				PoolElement.FMCCompressionRatio = strconv.FormatFloat(ParsedMap["usedFMCPoolVolumesCapacity"].(float64)/ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64), 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapTotal = strconv.FormatFloat(ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapFree = strconv.FormatFloat(((ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64) - ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)) / 1024), 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapUsed = strconv.FormatFloat(ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)

				//Free physical capacity [GB]:  * compression ratio total
				CompressionRatioTotal = RoundFloat64(CompressionRatioTotal, p.RoundPrecision)
				PoolElement.EffectiveGBFree = strconv.FormatFloat(((availablePhysicalVolumeCapacity / Mb2Gb) * CompressionRatioTotal), 'f', p.RoundPrecision, 64)

				//select the output type
				// at the beginning it is checked that only these two values pass the script
				switch p.OutputStyle {
				case "stdout":
					OutData, State = PoolInfoFormatTable(PoolElement, p)
					if OutputStandardFormat(OutData, p) {
						Warning.Println("The function 'OutputStandardFormat' returned an Error.")
					}
				case "csv":
					OutData, State = PoolInfoFormatCSV(OutData, PoolElement, p)
					//As all Pools have to be listed in one Table the output function is called at the end of the function
				}

			} else {
				//-------------------------------------
				//FMC containing disk pool
				//-------------------------------------

				Verbose.Println("FMC containing Pool")

				// Pool ID
				PoolElement.PoolID = strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64)

				//PoolElement.PoolType = ParsedMap["poolType"].(string)

				//Pool Name
				PoolElement.PoolName = ParsedMap["poolName"].(string)

				//Physical Capacity
				//Used(%)
				//PoolElement.usedPhysicalCapacityRate = strconv.FormatFloat(ParsedMap["usedPhysicalCapacityRate"].(float64), 'f', p.RoundPrecision, 64)
				//Total
				PoolElement.totalPhysicalCapacity = strconv.FormatFloat(ParsedMap["totalPhysicalCapacity"].(float64)/Mb2Gb, 'f', p.RoundPrecision, 64)
				//Used
				PoolElement.usedPhysicalCapacity = strconv.FormatFloat(ParsedMap["usedPhysicalCapacity"].(float64)/Mb2Gb, 'f', p.RoundPrecision, 64)
				//Free
				PoolElement.availablePhysicalVolumeCapacity = strconv.FormatFloat(availablePhysicalVolumeCapacity/Mb2Gb, 'f', p.RoundPrecision, 64)

				//Compression Ratio Total =   (totalPoolCapacity  - availablePhysicalVolumeCapacity ) / usedPhysicalCapacity
				var CompressionRatioTotal float64
				CompressionRatioTotal = (ParsedMap["totalPoolCapacity"].(float64) - availablePhysicalVolumeCapacity) / ParsedMap["usedPhysicalCapacity"].(float64)
				PoolElement.CompressionRatioTotal = strconv.FormatFloat(CompressionRatioTotal, 'f', p.RoundPrecision, 64)

				//PoolElement.OverallSavings = strconv.FormatFloat(((1 - (ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64) / LdevMappedUsedArray[0])) * 100), 'f', p.RoundPrecision, 64)

				//FMC Only Values
				//FMC Compression Ratio =  usedFMCPoolVolumesCapacity / usedPhysicalFMCPoolVolumesCapacity
				PoolElement.FMCCompressionRatio = strconv.FormatFloat(ParsedMap["usedFMCPoolVolumesCapacity"].(float64)/ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64), 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapTotal = strconv.FormatFloat(ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapFree = strconv.FormatFloat(((ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64) - ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)) / 1024), 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapUsed = strconv.FormatFloat(ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)

				//Free physical capacity [GB]:  * compression ratio total
				CompressionRatioTotal = RoundFloat64(CompressionRatioTotal, p.RoundPrecision)
				PoolElement.EffectiveGBFree = strconv.FormatFloat(((availablePhysicalVolumeCapacity / Mb2Gb) * CompressionRatioTotal), 'f', p.RoundPrecision, 64)

				//select the output type
				// at the beginning it is checked that only these two values pass the script
				switch p.OutputStyle {
				case "stdout":
					OutData, State = PoolInfoFormatTable(PoolElement, p)
					if OutputStandardFormat(OutData, p) {
						Warning.Println("The function 'OutputStandardFormat' returned an Error.")
					}
				case "csv":
					OutData, State = PoolInfoFormatCSV(OutData, PoolElement, p)
					//As all Pools have to be listed in one Table the output function is called at the end of the function
				}
			}

		} else {

			//-------------------------------------
			// no FMC disk pool
			//-------------------------------------

			//"HDT" -> Hitachi Dynamic Tiering -> HDT
			//"RT" -> Hitachi Realtime Tiering -> HDT
			if ParsedMap["poolType"].(string) == "RT" || ParsedMap["poolType"].(string) == "HDT" {
				Verbose.Println("No FMC HDT Pool")
				/*
				   +--------------------------------+----------+
				   | Pool ID                        |       20 | ->  "poolId"
				   +--------------------------------+----------+
				   | Pool name                      | FMC_HDP  | ->  "poolName"
				   +--------------------------------+----------+
				   | Overall Savings [%]            |    94.97 | ->  (1-(("totalPoolCapacity"-"availablePhysicalVolumeCapacity")/"mapped capacity"))
				   +--------------------------------+----------+
				   | FMC Compression Ratio          |     n/a  |
				   +--------------------------------+----------+
				   | Physical FMC Pool Volumes      |     n/a  |
				   | Capacity TOTAL [GB]            |          |
				   +--------------------------------+----------+
				   | Physical FMC Pool Volumes      |     n/a  |
				   | Capacity FREE  [GB]            |          |
				   +--------------------------------+----------+
				   | Physical FMC Pool Volumes      |     n/a  |
				   | Capacity USED  [GB]            |          |
				   +--------------------------------+----------+
				   | Effective GB FREE [GB]         | 17339.75 | -> "availablePhysicalVolumeCapacity" / 1024
				   +--------------------------------+----------+
				   | Virtual/Mapped GB FREE [GB]    | 53833.17 | -> ("totalPoolCapacity"-"availablePhysicalVolumeCapacity") * UsedCapacityRate / 1024
				   +--------------------------------+----------+
				   UsedCapacityRate =  (*"mapped capacity"* / *"used capacity"*)

				*/

				PoolElement.PoolID = strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64)
				//PoolElement.PoolType = "HDT"
				PoolElement.PoolName = ParsedMap["poolName"].(string)
				//PoolElement.usedPhysicalCapacityRate = strconv.FormatFloat(ParsedMap["usedPhysicalCapacityRate"].(float64), 'f', p.RoundPrecision, 64)
				PoolElement.totalPhysicalCapacity = strconv.FormatFloat(ParsedMap["totalPoolCapacity"].(float64)/Mb2Gb, 'f', p.RoundPrecision, 64)
				PoolElement.usedPhysicalCapacity = strconv.FormatFloat((ParsedMap["totalPoolCapacity"].(float64)-availablePhysicalVolumeCapacity)/Mb2Gb, 'f', p.RoundPrecision, 64)
				PoolElement.availablePhysicalVolumeCapacity = strconv.FormatFloat(availablePhysicalVolumeCapacity/Mb2Gb, 'f', p.RoundPrecision, 64)
				//PoolElement.OverallSavings = strconv.FormatFloat(((1 - ((ParsedMap["totalPoolCapacity"].(float64) - availablePhysicalVolumeCapacity) / LdevMappedUsedArray[0])) * 100), 'f', p.RoundPrecision, 64)
				PoolElement.FMCCompressionRatio = "-1.00"
				//PoolElement.PhysFMCPoolVolCapTotal = "-1.00"
				//PoolElement.PhysFMCPoolVolCapFree = "-1.00"
				//PoolElement.PhysFMCPoolVolCapUsed = "-1.00"
				//Free physical capacity [GB]:  * compression ratio total
				PoolElement.CompressionRatioTotal = "-1.00"
				PoolElement.EffectiveGBFree = strconv.FormatFloat(availablePhysicalVolumeCapacity/Mb2Gb, 'f', p.RoundPrecision, 64)
				//VirtualMappedGBFree := strconv.FormatFloat(RoundFloat64(ParsedMap["totalPoolCapacity"].(float64)-availablePhysicalVolumeCapacity, p.RoundPrecision)*
				//(LdevMappedUsedArray[0]/(ParsedMap["totalPoolCapacity"].(float64)-RoundFloat64(availablePhysicalVolumeCapacity, p.RoundPrecision)))/1024, 'f', p.RoundPrecision, 64)

				//select the output type
				// at the beginning it is checked that only these two values pass the script
				switch p.OutputStyle {
				case "stdout":
					OutData, State = PoolInfoFormatTable(PoolElement, p)
					if OutputStandardFormat(OutData, p) {
						Warning.Println("The function 'OutputStandardFormat' returned an Error.")
					}
				case "csv":
					OutData, State = PoolInfoFormatCSV(OutData, PoolElement, p)
					//As all Pools have to be listed in one Table the output function is called at the end of the function
				}

			}

			//"HDP" -> Hitachi Dynamic Provisioning -> HDP
			if ParsedMap["poolType"] == "HDP" {
				Verbose.Println("No FMC HDP Pool")

				// Pool ID
				PoolElement.PoolID = strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64)

				//PoolElement.PoolType = ParsedMap["poolType"].(string)

				//Pool Name
				PoolElement.PoolName = ParsedMap["poolName"].(string)

				//Physical Capacity
				//Used(%)
				//PoolElement.usedPhysicalCapacityRate = strconv.FormatFloat(ParsedMap["usedCapacityRate"].(float64), 'f', p.RoundPrecision, 64)
				//Total
				var TotalPoolCapacity float64
				TotalPoolCapacity = ParsedMap["totalPoolCapacity"].(float64) / Mb2Gb
				PoolElement.totalPhysicalCapacity = strconv.FormatFloat(TotalPoolCapacity, 'f', p.RoundPrecision, 64)
				//Free
				var FreePoolCapacity float64
				FreePoolCapacity = availablePhysicalVolumeCapacity / Mb2Gb
				PoolElement.availablePhysicalVolumeCapacity = strconv.FormatFloat(FreePoolCapacity, 'f', p.RoundPrecision, 64)
				//Used
				PoolElement.usedPhysicalCapacity = strconv.FormatFloat(TotalPoolCapacity-FreePoolCapacity, 'f', p.RoundPrecision, 64)

				//Compression Ratio Total =   (totalPoolCapacity  - availablePhysicalVolumeCapacity ) / usedPhysicalCapacity
				//var CompressionRatioTotal float64
				//CompressionRatioTotal = (ParsedMap["totalPoolCapacity"].(float64) - availablePhysicalVolumeCapacity)/ParsedMap["usedPhysicalCapacity"].(float64)/1027
				//PoolElement.CompressionRatioTotal = strconv.FormatFloat(CompressionRatioTotal, 'f', p.RoundPrecision, 64)
				PoolElement.CompressionRatioTotal = "-1.00"

				//PoolElement.OverallSavings = strconv.FormatFloat(((1 - (ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64) / LdevMappedUsedArray[0])) * 100), 'f', p.RoundPrecision, 64)

				//FMC Only Values
				//FMC Compression Ratio =  usedFMCPoolVolumesCapacity / usedPhysicalFMCPoolVolumesCapacity
				//PoolElement.FMCCompressionRatio = strconv.FormatFloat(ParsedMap["usedFMCPoolVolumesCapacity"].(float64)/ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64), 'f', p.RoundPrecision, 64)
				PoolElement.FMCCompressionRatio = "-1.00"
				//PoolElement.PhysFMCPoolVolCapTotal = strconv.FormatFloat(ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapFree = strconv.FormatFloat(((ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64) - ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)) / 1024), 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapUsed = strconv.FormatFloat(ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)

				//Free physical capacity [GB]:  * compression ratio total
				//CompressionRatioTotal = RoundFloat64(CompressionRatioTotal,p.RoundPrecision)
				//PoolElement.EffectiveGBFree = strconv.FormatFloat(((ParsedMap["availablePhysicalVolumeCapacity"].(float64)/1024)*CompressionRatioTotal), 'f', p.RoundPrecision, 64)
				PoolElement.EffectiveGBFree = PoolElement.availablePhysicalVolumeCapacity

				//select the output type
				// at the beginning it is checked that only these two values pass the script
				switch p.OutputStyle {
				case "stdout":
					OutData, State = PoolInfoFormatTable(PoolElement, p)
					if OutputStandardFormat(OutData, p) {
						Warning.Println("The function 'OutputStandardFormat' returned an Error.")
					}
				case "csv":
					OutData, State = PoolInfoFormatCSV(OutData, PoolElement, p)
					//As all Pools have to be listed in one Table the output function is called at the end of the function
				}

			}

			//if "poolType": "HTI" is Thin Pool then do not show the overall savings
			if ParsedMap["poolType"] == "HTI" {
				Verbose.Println("No FMC HTI Pool")

				// Pool ID
				PoolElement.PoolID = strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64)

				//PoolElement.PoolType = ParsedMap["poolType"].(string)

				//Pool Name
				PoolElement.PoolName = ParsedMap["poolName"].(string)

				//Physical Capacity
				//Used(%)
				//PoolElement.usedPhysicalCapacityRate = strconv.FormatFloat(ParsedMap["usedCapacityRate"].(float64), 'f', p.RoundPrecision, 64)
				//Total
				var TotalPoolCapacity float64
				TotalPoolCapacity = ParsedMap["totalPoolCapacity"].(float64) / Mb2Gb
				PoolElement.totalPhysicalCapacity = strconv.FormatFloat(TotalPoolCapacity, 'f', p.RoundPrecision, 64)
				//Free
				var FreePoolCapacity float64
				FreePoolCapacity = availablePhysicalVolumeCapacity / Mb2Gb
				PoolElement.availablePhysicalVolumeCapacity = strconv.FormatFloat(FreePoolCapacity, 'f', p.RoundPrecision, 64)
				//Used
				PoolElement.usedPhysicalCapacity = strconv.FormatFloat(TotalPoolCapacity-FreePoolCapacity, 'f', p.RoundPrecision, 64)

				//Compression Ratio Total =   (totalPoolCapacity  - availablePhysicalVolumeCapacity ) / usedPhysicalCapacity
				//var CompressionRatioTotal float64
				//CompressionRatioTotal = (ParsedMap["totalPoolCapacity"].(float64) - availablePhysicalVolumeCapacity)/ParsedMap["usedPhysicalCapacity"].(float64)/1027
				//PoolElement.CompressionRatioTotal = strconv.FormatFloat(CompressionRatioTotal, 'f', p.RoundPrecision, 64)
				PoolElement.CompressionRatioTotal = "-1.00"

				//PoolElement.OverallSavings = strconv.FormatFloat(((1 - (ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64) / LdevMappedUsedArray[0])) * 100), 'f', p.RoundPrecision, 64)

				//FMC Only Values
				//FMC Compression Ratio =  usedFMCPoolVolumesCapacity / usedPhysicalFMCPoolVolumesCapacity
				//PoolElement.FMCCompressionRatio = strconv.FormatFloat(ParsedMap["usedFMCPoolVolumesCapacity"].(float64)/ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64), 'f', p.RoundPrecision, 64)
				PoolElement.FMCCompressionRatio = "-1.00"
				//PoolElement.PhysFMCPoolVolCapTotal = strconv.FormatFloat(ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapFree = strconv.FormatFloat(((ParsedMap["availablePhysicalFMCPoolVolumesCapacity"].(float64) - ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)) / 1024), 'f', p.RoundPrecision, 64)
				//PoolElement.PhysFMCPoolVolCapUsed = strconv.FormatFloat(ParsedMap["usedPhysicalFMCPoolVolumesCapacity"].(float64)/1024, 'f', p.RoundPrecision, 64)

				//Free physical capacity [GB]:  * compression ratio total
				//CompressionRatioTotal = RoundFloat64(CompressionRatioTotal,p.RoundPrecision)
				//PoolElement.EffectiveGBFree = strconv.FormatFloat(((ParsedMap["availablePhysicalVolumeCapacity"].(float64)/1024)*CompressionRatioTotal), 'f', p.RoundPrecision, 64)
				PoolElement.EffectiveGBFree = PoolElement.availablePhysicalVolumeCapacity

				//select the output type
				// at the beginning it is checked that only these two values pass the script
				switch p.OutputStyle {
				case "stdout":
					OutData, State = PoolInfoFormatTable(PoolElement, p)
					if OutputStandardFormat(OutData, p) {
						Warning.Println("The function 'OutputStandardFormat' returned an Error.")
					}
				case "csv":
					OutData, State = PoolInfoFormatCSV(OutData, PoolElement, p)
					//As all Pools have to be listed in one Table the output function is called at the end of the function
				}

			}

		}

		//add the "storageDeviceId" to a slice to be able to choose afterwards
		//StorageSlice = append(StorageSlice, ParsedMap["storageDeviceId"].(string))

		//console output -> 1) VSP G1000 (Serial:50679 StorageDeviceID:800000050679 IP:10.70.5.145)
		Debug.Println("Get the Pool Information of Pool: " + strconv.FormatFloat(ParsedMap["poolId"].(float64), 'f', 0, 64) + " (" + ParsedMap["poolName"].(string) + ") completed")

	}

	// CSV output OutData
	if p.OutputStyle == "csv" {
		if OutputStandardFormat(OutData, p) {
			Warning.Println("The function 'OutputStandardFormat' returned an Error.")
		}
	}

	Info.Println("Get Pool information end")

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'PoolsGet' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'PoolsGet' return values State:", State)
	Debug.Println("Function 'PoolsGet' end")

	//state to OK
	State = false
	return "", State

}

//LdevCapSumGet is used to get the sum of all mapped LDEV Capacity [MB] and the sum of all used capacity of all mapped LDEVs [MB]
//return value (slice of two values ("sum of mapped capacity" and "sum of used capacity") (float64)) and the status of the request. if an error happened the state is true. Otherwise false.
//The function stops with exit status 50 ("JSON parsing error ("Unmarshal function threw an error).")
//The function stops with exit status 51 ("JSON parsing error (Return Format is not correct).")
//example: StorageRestAPIVersionGet("https", "443", "10.0.0.1", "c0492b4c-165d-4052-87e2-27053023e29f", "834000470018")
func LdevCapSumGet(p Params, PoolID float64) ([2]float64, bool) {
	//initial state is true that means NOK
	State := true

	Info.Println("Get all LDEVs to calculate the mapped and used capacity")

	var Out string
	Out = ""

	var MappedCapacity float64
	var UsedCapacity float64
	var SliceReturn [2]float64
	SliceReturn[0] = 0
	SliceReturn[1] = 0
	//initialize an array of two values with initial value 0
	//SliceReturn = make{[]float64, 0, 0}

	var ConfigurationManagerString string
	ConfigurationManagerString = "/ConfigurationManager/v1/objects/storages/"
	var LdevsString string
	LdevsString = "/ldevs?ldevOption=dpVolume&poolId="
	var ElementCountString string
	ElementCountString = "&count="

	//http://10.70.5.104/ConfigurationManager/v1/objects/storages/834000470018/ldevs?ldevOption=dpVolume&poolId=20&count=16384
	p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + ConfigurationManagerString + p.StorageDeviceID + LdevsString + strconv.FormatFloat(PoolID, 'f', 0, 64) + ElementCountString + strconv.FormatInt(p.MaxElementCount, 10)
	Verbose.Println(p.URL)
	p.RequestType = "GET"
	Out = HTTPRequest(p)

	//Testdata
	//Pool 20
	//output = `{    "data": [{        "ldevId": 2816,        "clprId": 0,        "emulationType": "OPEN-V-CVS",        "byteFormatCapacity": "1.00 T",        "blockCapacity": 2147483648,        "numOfPorts": 2,        "ports": [{            "portId": "CL1-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 1        }, {            "portId": "CL2-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 1        }],        "attributes": ["CVS", "HDP"],        "label": "FMC_HDP_TEST",        "status": "NML",        "mpBladeId": 2,        "ssid": "000F",        "poolId": 20,        "numOfUsedBlock": 533729280,        "isFullAllocationEnabled": false,        "resourceGroupId": 0,        "dataReductionStatus": "DISABLED",        "dataReductionMode": "disabled"    }, {        "ldevId": 2817,        "clprId": 0,        "emulationType": "OPEN-V-CVS",        "byteFormatCapacity": "1.00 T",        "blockCapacity": 2147483648,        "numOfPorts": 2,        "ports": [{            "portId": "CL1-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 2        }, {            "portId": "CL2-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 2        }],        "attributes": ["CVS", "HDP"],        "label": "FMC_HDP_TEST",        "status": "NML",        "mpBladeId": 0,        "ssid": "000F",        "poolId": 20,        "numOfUsedBlock": 800378880,        "isFullAllocationEnabled": false,        "resourceGroupId": 0,        "dataReductionStatus": "DISABLED",        "dataReductionMode": "disabled"    }, {        "ldevId": 2818,        "clprId": 0,        "emulationType": "OPEN-V-CVS",        "byteFormatCapacity": "1.00 T",        "blockCapacity": 2147483648,        "numOfPorts": 2,        "ports": [{            "portId": "CL1-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 4        }, {            "portId": "CL2-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 4        }],        "attributes": ["CVS", "HDP"],        "label": "FMC_HDP_TEST",        "status": "NML",        "mpBladeId": 2,        "ssid": "000F",        "poolId": 20,        "numOfUsedBlock": 106831872,        "isFullAllocationEnabled": false,        "resourceGroupId": 0,        "dataReductionStatus": "DISABLED",        "dataReductionMode": "disabled"    }, {        "ldevId": 2819,        "clprId": 0,        "emulationType": "OPEN-V-CVS",        "byteFormatCapacity": "1.00 T",        "blockCapacity": 2147483648,        "numOfPorts": 2,        "ports": [{            "portId": "CL1-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 5        }, {            "portId": "CL2-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 5        }],        "attributes": ["CVS", "HDP"],        "label": "FMC_HDP_TEST",        "status": "NML",        "mpBladeId": 0,        "ssid": "000F",        "poolId": 20,        "numOfUsedBlock": 1376256,        "isFullAllocationEnabled": false,        "resourceGroupId": 0,        "dataReductionStatus": "DISABLED",        "dataReductionMode": "disabled"    }, {        "ldevId": 2820,        "clprId": 0,        "emulationType": "OPEN-V-CVS",        "byteFormatCapacity": "1.00 T",        "blockCapacity": 2147483648,        "numOfPorts": 2,        "ports": [{            "portId": "CL1-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 6        }, {            "portId": "CL2-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 6        }],        "attributes": ["CVS", "HDP"],        "label": "FMC_HDP_TEST",        "status": "NML",        "mpBladeId": 2,        "ssid": "000F",        "poolId": 20,        "numOfUsedBlock": 518246400,        "isFullAllocationEnabled": false,        "resourceGroupId": 0,        "dataReductionStatus": "DISABLED",        "dataReductionMode": "disabled"    }, {        "ldevId": 2821,        "clprId": 0,        "emulationType": "OPEN-V-CVS",        "byteFormatCapacity": "1.00 T",        "blockCapacity": 2147483648,        "numOfPorts": 2,        "ports": [{            "portId": "CL1-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 7        }, {            "portId": "CL2-E",            "hostGroupNumber": 1,            "hostGroupName": "CB500_blade3_lpa",            "lun": 7        }],        "attributes": ["CVS", "HDP"],        "label": "FMC_HDP_TEST",        "status": "NML",        "mpBladeId": 0,        "ssid": "000F",        "poolId": 20,        "numOfUsedBlock": 544825344,        "isFullAllocationEnabled": false,        "resourceGroupId": 0,        "dataReductionStatus": "DISABLED",        "dataReductionMode": "disabled"    }]}`
	//Pool
	//Verbose.Println("RAW data: " + output)

	//is the string "data" in the output
	if CheckIsInString(Out, p.DataElement) {
		Verbose.Println(Out)
	}

	var JSONUnmarshalOut map[string]interface{}
	JSONUnmarshalOut, State = JSONUnmarshal(Out)
	Verbose.Println("JSON Unmarshal:", JSONUnmarshalOut)

	// all responses from hitachi rest api calls answer with only one element called data "{ "data": [{"
	if len(JSONUnmarshalOut) != 1 {
		//this should never happens
		//at hitachi the response always starts with "{ "data": [{"
		Error.Println("JSON parsing error (Return Format is not correct)")
		os.Exit(51)
	}

	Verbose.Println("Number of LDEVs:", len(JSONUnmarshalOut["data"].([]interface{})))
	for Key1, Value1 := range JSONUnmarshalOut["data"].([]interface{}) {
		//Verbose.Println("Key: "+strconv.Itoa(Key1), "Value: ", Value1)
		ParsedMap := Value1.(map[string]interface{})

		//Verbose.Println("Element:", key1)

		//mapped capacity
		if ParsedMap["blockCapacity"] != nil {
			if w, ok := ParsedMap["blockCapacity"].(float64); ok {
				Verbose.Println("Element: ", Key1, "Mapped Capacity [MB]: ", w*512/1024/1024, " in Blocks -> ", w)
				MappedCapacity = MappedCapacity + w*512/1024/1024
			}
		}

		//used capacity
		if ParsedMap["numOfUsedBlock"] != nil {
			if w, ok := ParsedMap["numOfUsedBlock"].(float64); ok {
				Verbose.Println("Element: ", Key1, "Used Capacity [MB]: ", w*512/1024/1024, " in Blocks -> ", w)
				UsedCapacity = UsedCapacity + w*512/1024/1024
			}
		}

	}
	// first value in array is the mapped capacity in [MB]
	SliceReturn[0] = MappedCapacity
	// second value in the array is the used capacity in [MB]
	SliceReturn[1] = UsedCapacity

	Verbose.Println("Return Array: ", SliceReturn)

	Info.Println("Get all LDEVs to calculate the mapped and used capacity completed")

	State = false
	return SliceReturn, State
}

//PoolInfoFormatTable formats the Pool data for standard output
func PoolInfoFormatTable(PoolDataSet PoolInfo, p Params) ([][]string, bool) {
	Debug.Println("Function 'PoolInfoFormatTable' strated.")
	//start timer
	TimeStart := time.Now()

	//initial state is true that means NOK
	State := true

	OutData := [][]string{}

	//table start line
	OutData = append(OutData, []string{p.ElementStringStart})

	OutData = append(OutData, []string{"Pool ID", PoolDataSet.PoolID})
	//OutData = append(OutData, []string{"Pool type", PoolDataSet.PoolType})
	OutData = append(OutData, []string{"Pool name", PoolDataSet.PoolName})

	OutData = append(OutData, []string{"Total physical capacity [GB]", PoolDataSet.totalPhysicalCapacity})
	OutData = append(OutData, []string{"Used physical capacity [GB]", PoolDataSet.usedPhysicalCapacity})
	OutData = append(OutData, []string{"Free physical capacity [GB]", PoolDataSet.availablePhysicalVolumeCapacity})

	// "usedFMCPoolVolumesCapacity" / "usedPhysicalFMCPoolVolumesCapacity"
	OutData = append(OutData, []string{"Compression ratio FMC", PoolDataSet.FMCCompressionRatio})

	//(totalPoolCapacity  - availablePhysicalVolumeCapacity ) / usedPhysicalCapacity
	OutData = append(OutData, []string{"Compression ratio total", PoolDataSet.CompressionRatioTotal})

	// "availablePhysicalFMCPoolVolumesCapacity"
	//OutData = append(OutData, []string{"Physical FMC Pool Volumes Capacity TOTAL [GB]", PoolDataSet.PhysFMCPoolVolCapTotal})
	// free_fmc_capacity
	// "availablePhysicalFMCPoolVolumesCapacity" - "usedPhysicalFMCPoolVolumesCapacity"
	//OutData = append(OutData, []string{"Physical FMC Pool Volumes Capacity FREE  [GB]", PoolDataSet.PhysFMCPoolVolCapFree})
	// "usedPhysicalFMCPoolVolumesCapacity"
	//OutData = append(OutData, []string{"Physical FMC Pool Volumes Capacity USED  [GB]", PoolDataSet.PhysFMCPoolVolCapUsed})
	// ("availablePhysicalFMCPoolVolumesCapacity" - "usedPhysicalFMCPoolVolumesCapacity") * ("usedFMCPoolVolumesCapacity" / "usedPhysicalFMCPoolVolumesCapacity")
	OutData = append(OutData, []string{"Effective total GB free [GB]", PoolDataSet.EffectiveGBFree})
	// (1-("usedPhysicalFMCPoolVolumesCapacity"/"*mapped capacity*"))
	//LdevMappedUsedArray[0] -> mapped capacity
	//LdevMappedUsedArray[1] -> used capacity
	//OutData = append(OutData, []string{"Overall Savings [%]", PoolDataSet.OverallSavings})
	// ("availablePhysicalFMCPoolVolumesCapacity" - "usedPhysicalFMCPoolVolumesCapacity") *  ("*mapped capacity*" / "usedPhysicalFMCPoolVolumesCapacity") / 1024
	//OutData = append(OutData, []string{"Virtual/Mapped GB FREE [GB]", VirtualMappedGBFree})

	//table end line
	OutData = append(OutData, []string{p.ElementStringEnd})

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'PoolInfoFormatTable' - Elapsed time ", TimeDiff)

	State = false
	Debug.Println("Function 'PoolInfoFormatTable' return values OutData:", OutData)
	Debug.Println("Function 'PoolInfoFormatTable' return values   State:", State)
	Debug.Println("Function 'PoolInfoFormatTable' ended.")

	State = false
	return OutData, State
}

//PoolInfoFormatCSV formats the Pool data for standard output
func PoolInfoFormatCSV(OutData [][]string, PoolDataSet PoolInfo, p Params) ([][]string, bool) {
	Debug.Println("Function 'PoolInfoFormatCSV' strated.")
	//start timer
	TimeStart := time.Now()

	//initial state is true that means NOK
	State := true

	var TempData [][]string
	TempData = OutData

	//table start line
	TempData = append(TempData, []string{p.ElementStringStart})

	TempData = append(TempData, []string{"Pool ID(string)", PoolDataSet.PoolID})
	//TempData = append(TempData, []string{"Pool type(string)", PoolDataSet.PoolType})
	TempData = append(TempData, []string{"Pool name(string)", PoolDataSet.PoolName})

	TempData = append(TempData, []string{"Total physical capacity [GB](float64)", PoolDataSet.totalPhysicalCapacity})
	TempData = append(TempData, []string{"Used physical capacity [GB](float64)", PoolDataSet.usedPhysicalCapacity})
	TempData = append(TempData, []string{"Free physical capacity [GB](float64)", PoolDataSet.availablePhysicalVolumeCapacity})

	// "usedFMCPoolVolumesCapacity" / "usedPhysicalFMCPoolVolumesCapacity"
	TempData = append(TempData, []string{"Compression ratio FMC", PoolDataSet.FMCCompressionRatio})

	// "usedFMCPoolVolumesCapacity" / "usedPhysicalFMCPoolVolumesCapacity"
	TempData = append(TempData, []string{"Compression ratio total(string)", PoolDataSet.CompressionRatioTotal})

	// "availablePhysicalFMCPoolVolumesCapacity"
	//TempData = append(TempData, []string{"Physical FMC Pool Volumes Capacity TOTAL [GB](float64)", PoolDataSet.PhysFMCPoolVolCapTotal})
	// free_fmc_capacity
	// "availablePhysicalFMCPoolVolumesCapacity" - "usedPhysicalFMCPoolVolumesCapacity"
	//TempData = append(TempData, []string{"Physical FMC Pool Volumes Capacity FREE [GB](float(64)", PoolDataSet.PhysFMCPoolVolCapFree})
	// "usedPhysicalFMCPoolVolumesCapacity"
	//TempData = append(TempData, []string{"Physical FMC Pool Volumes Capacity USED [GB](float64)", PoolDataSet.PhysFMCPoolVolCapUsed})
	// ("availablePhysicalFMCPoolVolumesCapacity" - "usedPhysicalFMCPoolVolumesCapacity") * ("usedFMCPoolVolumesCapacity" / "usedPhysicalFMCPoolVolumesCapacity")
	TempData = append(TempData, []string{"Effective total GB free [GB](float64)", PoolDataSet.EffectiveGBFree})
	// (1-("usedPhysicalFMCPoolVolumesCapacity"/"*mapped capacity*"))
	//LdevMappedUsedArray[0] -> mapped capacity
	//LdevMappedUsedArray[1] -> used capacity
	//TempData = append(TempData, []string{"Overall Savings [%](float64)", PoolDataSet.OverallSavings})
	// ("availablePhysicalFMCPoolVolumesCapacity" - "usedPhysicalFMCPoolVolumesCapacity") *  ("*mapped capacity*" / "usedPhysicalFMCPoolVolumesCapacity") / 1024
	//TempData = append(TempData, []string{"Virtual/Mapped GB FREE [GB]", VirtualMappedGBFree})

	//table end line
	TempData = append(TempData, []string{p.ElementStringEnd})

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'PoolInfoFormatCSV' - Elapsed time ", TimeDiff)

	State = false
	Debug.Println("Function 'PoolInfoFormatCSV' return values TempData:", TempData)
	Debug.Println("Function 'PoolInfoFormatCSV' return values   State:", State)
	Debug.Println("Function 'PoolInfoFormatCSV' ended.")
	return TempData, State
}

//OutputStandardFormat modyfies the output values to Standard Format
//this means that just the values get passed to the function and here all the text arount it is added.
func OutputStandardFormat(Data [][]string, p Params) bool {
	Debug.Println("Function 'OutputStandardFormat' started.")
	//start timer
	TimeStart := time.Now()

	//false -> OK
	State := false

	//select output type
	switch {
	case p.OutputStyle == "stdout":
		Debug.Print("OutputStype: " + p.OutputStyle)
		Debug.Print("Data: ", Data)
		State = OutputTable(Data, p.ElementStringStart, p.ElementStringEnd)
	case p.OutputStyle == "csv":
		Debug.Print("OutputStype: " + p.OutputStyle)
		Debug.Print("Data: ", Data)
		State = OutputCSV(Data, p.ElementStringStart, p.ElementStringEnd, p.CSVString)
	default:
		Warning.Print("Output Format (" + p.OutputStyle + ") invalid. stdout taken instead.")
		Debug.Print("OutputStype: " + p.OutputStyle)
		Debug.Print("Data: ", Data)
		State = OutputTable(Data, p.ElementStringStart, p.ElementStringEnd)
	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'OutputStandardFormat' - Elapsed time ", TimeDiff)

	//Debug.Println("Function 'OutputStandardFormat' return values OutData:", OutData)
	Debug.Println("Function 'OutputStandardFormat' ended.")
	return State
}

//OutputTable outputs the data to the command line
func OutputTable(Data [][]string, ElementStringStart string, ElementStringEnd string) bool {
	Debug.Println("Function 'OutputTable' started.")
	//start timer
	TimeStart := time.Now()

	//true -> NOK
	//false -> OK
	State := false

	var ElementStart bool
	ElementStart = false

	// if no data is available skip output
	if len(Data) == 0 {
		Error.Println("No Data to output.")
	} else {
		table := tablewriter.NewWriter(os.Stdout)
		//table.SetAlignment(tablewriter.ALIGN_RIGHT) // Set Alignment
		//table.SetAutoMergeCells(true)
		//table.SetBorder(false)                         // Set Border to false
		table.SetRowLine(true)
		// Change table lines
		//table.SetCenterSeparator("*")
		//table.SetColumnSeparator("‡")
		table.SetRowSeparator("-")
		//table.SetHeader([]string{"Pool ID", "Pool name", "Pool Used [%]", "Overall Savings [%] mapped/phys used", "Overall Compression Ratio"})
		//table.SetFooter([]string{"", "", "Total", "$146.93"}) // Add Footer
		//table.Append([]string{"Bla", ""})
		for i := 0; i < len(Data); i++ {
			//Info message Pool
			if ElementStart {
				//Info.Println(Data[i][0] + ": " + Data[i][1] + " " + Data[i+1][0] + ": " + Data[i+1][1])
				Info.Println("")
				ElementStart = false
			}

			//skip start and end string
			if Data[i][0] != ElementStringEnd && Data[i][0] != ElementStringStart {
				Debug.Println("Data:"+strconv.Itoa(i)+"]:", Data[i])
				table.Append(Data[i])
			} else {
				if Data[i][0] == ElementStringStart {
					ElementStart = true
				}
			}
		}
		table.Render() // Send output
	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'OutputTable' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'OutputStandardFormat' return values State:", State)
	Debug.Println("Function 'OutputTable' ended.")
	return State
}

//OutputCSV outputs the data to a comma separated file in the same directory.
func OutputCSV(Data [][]string, ElementStringStart string, ElementStringEnd string, SeparatorString string) bool {
	Debug.Println("Function 'OutputCSV' started.")
	//start timer
	TimeStart := time.Now()

	//true -> NOK
	//false -> OK
	State := false

	//RFC3339 time format used
	const TimeformatString = "Time(RFC3339)"
	var TimeFormatValue string
	TimeFormatValue = TimeStart.Format(time.RFC3339)

	var Descriptor string
	Descriptor = TimeformatString
	var Values string
	Values = TimeFormatValue

	// if no data is available skip output
	if len(Data) == 0 {
		Error.Println("No Data to output.")
	} else {
		//Descriptor
		//go through all elements
		for i := 0; i < len(Data); i++ {
			//skip start and end string
			if Data[i][0] != ElementStringEnd && Data[i][0] != ElementStringStart {
				//Debug.Println("Descriptor - Data["+strconv.Itoa(i)+"]:", Data[i][0])
				if Descriptor == "" {
					Descriptor = Data[i][0]
				} else {
					Descriptor = Descriptor + SeparatorString + Data[i][0]
				}
			}
			//exit for after all descriptors are once gone through
			if Data[i][0] == ElementStringEnd {
				break
			}
		}
		//descriptor line
		fmt.Println(Descriptor)

		//values
		//go through all elements
		for i := 0; i < len(Data); i++ {
			//skip start and end string
			if Data[i][0] != ElementStringEnd && Data[i][0] != ElementStringStart {
				//Debug.Println("Descriptor - Data["+strconv.Itoa(i)+"]:", Data[i][1])
				if Values == "" {
					Values = Data[i][1]
				} else {
					Values = Values + SeparatorString + Data[i][1]
				}
			} else {
				if Data[i][0] == ElementStringEnd {
					//print the values
					fmt.Println(Values)
					Values = TimeFormatValue
				}
			}
		}
	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'OutputCSV' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'OutputCSV' return values State:", State)
	Debug.Println("Function 'OutputCSV' ended.")
	return State
}

//StorageRestAPIVersionGet is used to get the RestAPI version
//return value (string) is the version used and the status of the request. if an error happened the state is true. Otherwise false.
//The function stops with exit status 30 ("JSON parsing error."")
//example: StorageRestAPIVersionGet("https", 10.0.0.1, username, password)
func StorageRestAPIVersionGet(p Params) (string, bool) {
	Debug.Println("Function 'StorageRestAPIVersionGet' started.")
	//start timer
	TimeStart := time.Now()

	//initial state is false that means OK
	var State bool
	// State = NOK
	State = true

	var Out string

	//specify constants
	const APIVersionElement string = "apiVersion"

	Verbose.Println("Get the Rest API version")

	p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + "/ConfigurationManager/configuration/version"
	Debug.Println("URL: " + p.URL)
	p.RequestType = "GET"

	Out = HTTPRequest(p)
	//is the apiVersion variable specified in the output
	if CheckIsInString(Out, APIVersionElement) {
		Error.Println(Out)
	}

	//{  "productName" : "Configuration Manager REST API", "apiVersion" : "1.5.0" }

	var JSONUnmarshalOut map[string]interface{}
	JSONUnmarshalOut, State = JSONUnmarshal(Out)
	Debug.Println("JSON Unmarshal:", JSONUnmarshalOut)

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'StorageRestAPIVersionGet' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'StorageRestAPIVersionGet' return values APIVersionElement:", JSONUnmarshalOut[APIVersionElement].(string))
	Debug.Println("Function 'StorageRestAPIVersionGet' return values State:", State)
	Debug.Println("Function 'StorageRestAPIVersionGet' ended.")

	Verbose.Println("APIVersion: " + JSONUnmarshalOut[APIVersionElement].(string) + " created.")
	Verbose.Println("Get the Rest API version completed")
	//state to OK
	State = false
	return JSONUnmarshalOut[APIVersionElement].(string), State

}

//StorageDeviceIDGet is used to get the RestAPI version
//return value (string) is the version used and the status of the request. if an error happened the state is true. Otherwise false.
//The function stops with exit status 10 ("JSON parsing error (Unmarshal function threw an error).")
//The function stops with exit status 11 ("JSON parsing error (Return Format is not correct).")
//example: StorageRestAPIVersionGet("https", 10.0.0.1, username, password)
func StorageDeviceIDGet(p Params) (string, bool) {
	//C:\>  curl -k -H "Accept:application/json" -H "Content-Type:application/json" -X GET "https://10.70.5.104/ConfigurationManager/configuration/version"
	Debug.Println("Function 'StorageDeviceIDGet' started.")
	//start timer
	TimeStart := time.Now()

	//initial state is false that means OK
	//state := true
	State := false

	var Out string
	Out = ""

	//specify constants
	const ModelElement string = "model"
	const SerialNumberElement string = "serialNumber"
	const StorageDeviceIDElement string = "storageDeviceId"

	//to check if the output contains the storage device id
	//ContainsstorageDeviceID := "storageDeviceId"

	Verbose.Println("Get the Storage Device ID")

	p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + "/ConfigurationManager/v1/objects/storages"
	Debug.Println(p.URL)
	p.RequestType = "GET"
	Out = HTTPRequest(p)

	//Verbose.Println(Out)

	// Testdata
	//One storage
	//Out = `{ "data" : [ { "storageDeviceId" : "834000470018", "model" : "VSP G400", "serialNumber" : 470018, "svpIp" : "10.70.5.104" } ] }`
	//more than one storage
	//Out = `{ "data" : [ { "storageDeviceId" : "800000050679", "model" : "VSP G1000", "serialNumber" : 50679, "svpIp" : "10.70.5.145"}, { "storageDeviceId" : "834000470018", "model" : "VSP G600", "serialNumber" : 470018, "svpIp" : "10.70.5.104"} ] }`

	//is the string "data" in the output
	if CheckIsInString(Out, p.DataElement) {
		Error.Println(Out)
	}

	var JSONUnmarshalOut map[string]interface{}
	JSONUnmarshalOut, State = JSONUnmarshal(Out)
	Debug.Println("JSON Unmarshal:", JSONUnmarshalOut)

	// all responses from hitachi rest api calls answer with only one element called data "{ "data": [{"
	if len(JSONUnmarshalOut) != 1 {
		//this should never happens
		//at hitachi the response always starts with "{ "data": [{"
		Error.Println("JSON parsing error (Return Format is not correct).")
		os.Exit(11)
	}

	//How many storage systems do we see

	//this variable is used to convert a map of key(string) and value(interface) to an indexable variable
	var ParsedMap map[string]interface{}

	//only one storage system
	if len(JSONUnmarshalOut[p.DataElement].([]interface{})) == 1 {
		Verbose.Println("Only one storage system")

		//get the map out of the parsed json response
		for key, value := range JSONUnmarshalOut[p.DataElement].([]interface{}) {
			Debug.Println("Key: "+strconv.Itoa(key), "Value: ", value)

			//this is to store the map in a variable to make it indexable
			ParsedMap = value.(map[string]interface{})
		}

		Debug.Println("Storage Model: " + ParsedMap[ModelElement].(string))
		Debug.Println("Storage Serial: " + strconv.FormatFloat(ParsedMap[SerialNumberElement].(float64), 'f', 0, 64))
		Verbose.Println("Return the Storage Device ID: " + ParsedMap[StorageDeviceIDElement].(string))

		Verbose.Println("Get the Storage Device ID completed")
		//state to OK
		State = false

		//return the "storageDeviceId" and its state
		return ParsedMap[StorageDeviceIDElement].(string), State
	}

	//more than one storage system
	Verbose.Println("More than one storage system or none")

	//slice of all storageDeviceId's
	var StorageDeviceIDSlice []string

	//slice of all storageDeviceId's
	var StorageSerialSlice []float64

	fmt.Println("Choose one Storage System by the number: ")
	//fmt.Println(len(parsed["data"].([]interface{})))
	//exit line press 0
	fmt.Println("0) press 0 to exit script")
	fmt.Println("1) Register new Storage System")
	var ElementNumber int
	ElementNumber = 2

	for key1, value1 := range JSONUnmarshalOut["data"].([]interface{}) {
		Debug.Println("Key: "+strconv.Itoa(key1), "Value: ", value1)
		ParsedMap := value1.(map[string]interface{})

		//add the "storageDeviceId" to a slice to be able to choose afterwards
		StorageDeviceIDSlice = append(StorageDeviceIDSlice, ParsedMap["storageDeviceId"].(string))
		//add the "serialNumber" to a slice t check afterward if the storage is already registered
		StorageSerialSlice = append(StorageSerialSlice, ParsedMap["serialNumber"].(float64))
		//console output -> 1) VSP G1000 (Serial:50679 StorageDeviceID:800000050679 IP:10.70.5.145)
		fmt.Println(strconv.Itoa(ElementNumber) + ") " + ParsedMap["model"].(string) + " (Serial:" + strconv.FormatFloat(ParsedMap["serialNumber"].(float64), 'g', -1, 64) + " StorageDeviceID:" + ParsedMap["storageDeviceId"].(string) + " IP:" + ParsedMap["svpIp"].(string) + ")")
		ElementNumber = ElementNumber + 1
	}

	Debug.Println("Array of all storageDeviceIds: ", StorageDeviceIDSlice)
	Debug.Println("Array of all storageDeviceIds: ", StorageSerialSlice)

	var inputint int
	//ask for an option
	fmt.Scanln(&inputint)
	fmt.Println("Choosen option: ", inputint)

	//check that only an available value was entered
	if inputint > ElementNumber {
		fmt.Println("Wrong option value pressed")
		fmt.Println("Exiting Script")
		os.Exit(100)
	}

	// if 0 pressed then exit the script
	if inputint == 0 {
		fmt.Println("0 or wrong entry pressed")
		fmt.Println("Exiting Script")
		os.Exit(100)
	}

	// if 1 pressed or another wrong value then exit the script
	if inputint == 1 {
		//HCS Configuration Manager - register storage
		Out, State = HCSRegisterStorage(p, StorageSerialSlice, StorageDeviceIDSlice)
		if State {
			Error.Println("The \"HCSRegisterStorage\" function returned a bad state and the following output: " + Out)
		}

		Verbose.Println("Function 'StorageDeviceIDGet' return value(s) StorageDeviceID:", Out)
		Verbose.Println("Get the Storage Device ID completed")
		//state to OK
		State = false
		return Out, State
	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'StorageDeviceIDGet' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'StorageDeviceIDGet' return value(s) StorageDeviceID:", StorageDeviceIDSlice[inputint-2])
	Debug.Println("Function 'StorageDeviceIDGet' return values State:", State)
	Debug.Println("Function 'StorageDeviceIDGet' ended.")
	Verbose.Println("StorageDeviceID:", StorageDeviceIDSlice[inputint-2])
	Verbose.Println("Get the Storage Device ID completed")

	//state to OK
	State = false
	//return the coosen "storageDeviceId" and its state
	return StorageDeviceIDSlice[inputint-2], State
}

//HCSRegisterStorage is used to register the Storage to HCS Configuration Manager
//return value (string) is the version used and the status of the request. if an error happened the state is true. Otherwise false.
//The function stops with exit status 10 ("JSON parsing error (Unmarshal function threw an error).")
//The function stops with exit status 11 ("JSON parsing error (Return Format is not correct).")
//example: StorageRestAPIVersionGet("https", 10.0.0.1, username, password)
func HCSRegisterStorage(p Params, StorageSerialSlice []float64, StorageDeviceIDSerialSlice []string) (string, bool) {
	Debug.Println("Function 'HCSRegisterStorage' started.")
	//start timer
	TimeStart := time.Now()

	//initial state is false that means OK
	State := true

	Verbose.Println("Register storage system")

	var Out string
	Out = ""

	var StorageIPOrHostname string
	StorageIPOrHostname = ""

	//specify constants
	const SVPIPElement string = "svpIp"
	const SerialNumberElement string = "serialNumber"
	const ModelElement string = "model"
	const StorageDeviceIDElement string = "storageDeviceId"

	//JSON input
	var SvpIP string
	SvpIP = ""
	var SerialNumber string
	SerialNumber = ""
	var Model string
	Model = ""
	var JSONInputString string
	JSONInputString = ""
	var StorageDeviceID string
	StorageDeviceID = ""

	//specify the storage port dependinge port used to conntact hcs
	var StoragePort string
	if p.Protocol == "https" {
		StoragePort = "443"
	} else {
		StoragePort = "80"
	}

	//specify the hostname or ip of the svp
	fmt.Println("Please enter the IP/hostname of the storage system:")
	//ask for input
	var StorageIPHostnameInputstring string
	_, err := fmt.Scanln(&StorageIPHostnameInputstring)
	if err != nil {
		Error.Println("Error: ", err)
	}
	StorageIPOrHostname = StorageIPHostnameInputstring
	Debug.Println(StorageIPOrHostname)

	//was the username and password the hcs ones?
	fmt.Println("")
	fmt.Println("!!! The user you specified in the command line must be the storage user otherwise the script will not work. !!!")
	fmt.Println("")

	//connect to svp and get the storage parmeter
	//check if the StorageDeviceID is not already registered
	Verbose.Println("Get the storage information")

	//if unsecureuse port 80
	p.URL = p.Protocol + "://" + StorageIPOrHostname + ":" + StoragePort + "/ConfigurationManager/v1/objects/storages"
	Debug.Println("URL: " + p.URL)
	p.RequestType = "GET"
	Out = HTTPRequest(p)

	//get data
	//Verbose.Println("RAW data: " + output)
	//is the svpIp variable specified in the output
	if CheckIsInString(Out, SVPIPElement) {
		Error.Println(Out)
	}

	//is the string "data" in the output
	if CheckIsInString(Out, p.DataElement) {
		Error.Println(Out)
	}

	var JSONUnmarshalOut map[string]interface{}
	JSONUnmarshalOut, State = JSONUnmarshal(Out)
	Debug.Println("JSON Unmarshal:", JSONUnmarshalOut)

	// all responses from hitachi rest api calls answer with only one element called data "{ "data": [{"
	if len(JSONUnmarshalOut) != 1 {
		//this should never happens
		//at hitachi the response always starts with "{ "data": [{"
		Error.Println("JSON parsing error (Return Format is not correct).")
		os.Exit(11)
	}

	//this variable is used to convert a map of key(string) and value(interface) to an indexable variable
	var ParsedMap map[string]interface{}

	//get the map out of the parsed json response
	for key, value := range JSONUnmarshalOut[p.DataElement].([]interface{}) {
		Debug.Println("Key: "+strconv.Itoa(key), "Value: ", value)

		//this is to store the map in a variable to make it indexable
		ParsedMap = value.(map[string]interface{})
	}

	Debug.Println("SVP IP: " + ParsedMap[SVPIPElement].(string))
	SvpIP = ParsedMap[SVPIPElement].(string)
	Debug.Println("Serial Number: " + strconv.FormatFloat(ParsedMap[SerialNumberElement].(float64), 'f', 0, 64))
	SerialNumber = strconv.FormatFloat(ParsedMap[SerialNumberElement].(float64), 'f', 0, 64)
	Debug.Println("Storage model: " + ParsedMap[ModelElement].(string))
	Model = ParsedMap[ModelElement].(string)

	Verbose.Println("Get the storage information completed")
	Verbose.Println("Register the storage")
	var Position int
	Position = 0

	var AlreadyExistent bool
	AlreadyExistent = false
	//ckeck if the storage already exists
	Verbose.Println("Check if the storage is already registered.")
	for key, value := range StorageSerialSlice {
		Debug.Println("Key: "+strconv.Itoa(key), "Value: ", value)
		if SerialNumber == strconv.FormatFloat(value, 'f', 0, 64) {
			AlreadyExistent = true
			Position = key
			Warning.Println("The Storage: " + strconv.FormatFloat(value, 'f', 0, 64) + " already exists. The registration process will be skipped.")
		}
	}

	if AlreadyExistent {
		//is already existent
		StorageDeviceID = StorageDeviceIDSerialSlice[Position]
	} else {
		//not existent
		// register storage
		//create put string
		//doubleqouote in a string must look like \"
		JSONInputString = "{\"" + SVPIPElement + "\":\"" + SvpIP + "\",\"" + SerialNumberElement + "\":" + SerialNumber + ",\"" + ModelElement + "\":\"" + Model + "\"}"
		Debug.Println("JSON Request body:" + JSONInputString)
		//curl -kv -H "Accept:application/json" -H "Content-Type:application/json" -u raidcom:raidcom -X POST --data-binary "@./g600.txt" https://10.70.4.84:23451/ConfigurationManager/v1/objects/storages

		p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + "/ConfigurationManager/v1/objects/storages"
		Debug.Println("URL: " + p.URL)
		p.RequestType = "POST"
		p.RequestBody = JSONInputString
		Out = HTTPRequest(p)

		//get data
		//Verbose.Println("RAW data: " + output)
		//is the storageDeviceId variable specified in the output
		if CheckIsInString(Out, StorageDeviceIDElement) {
			Debug.Println(Out)
		}

		JSONUnmarshalOut, State = JSONUnmarshal(Out)
		Debug.Println("JSON Unmarshal:", JSONUnmarshalOut)

		StorageDeviceID = JSONUnmarshalOut[StorageDeviceIDElement].(string)
	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'HCSRegisterStorage' - Elapsed time ", TimeDiff)
	Verbose.Println("Function 'HCSRegisterStorage' return values StorageDeviceID:", StorageDeviceID)
	Debug.Println("Function 'HCSRegisterStorage' return values State:", State)
	Debug.Println("Function 'HCSRegisterStorage' ended.")
	Verbose.Println("Register storage system completed")

	//state to OK
	State = false

	//return the "storageDeviceId" and its state
	//	return ParsedMap["storageDeviceId"].(string), state
	return StorageDeviceID, State
}

//HCSDeletetorage
//delete storage
//curl -kv -H "Accept:application/json" -H "Content-Type:application/json" -u raidcom:raidcom -X DELETE https://10.70.4.84:23451/ConfigurationManager/v1/objects/storages/834000470018

//TokenGet is used to get the Security token
//return values are the security token(string) and the session id(float64). if an error happened the state is true. Otherwise false.
//The function stops with exit status 20 ("The StorageDeviceId must not be empty.")
//The function stops with exit status 21 ("JSON parsing error.")
//example: TokenGet("https", 10.0.0.1, username, password, StorageDeviceID)
func TokenGet(p Params) (string, float64, bool) {
	//curl -k -H "Accept:application/json" -H "Content-Type:application/json" -u maintenance:raid-maintenance -X POST "https://10.70.5.104/ConfigurationManager/v1/objects/storages/834000470018/sessions/"

	Debug.Println("Function 'TokenGet' started.")
	//start timer
	TimeStart := time.Now()

	//state to NOK
	var State bool
	State = true

	var Out string

	//specify constants
	const TokenElement string = "token"
	const SessionIDElement string = "sessionId"

	Verbose.Println("Get the Security Token")

	//check if StorageDeviceID is not empty
	if p.StorageDeviceID == "" {
		Error.Println("The StorageDeviceId must not be empty")
		os.Exit(20)
	}

	p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + "/ConfigurationManager/v1/objects/storages/" + p.StorageDeviceID + "/sessions/"
	Debug.Println("URL: " + p.URL)
	p.RequestType = "POST"
	Out = HTTPRequest(p)

	//is the Token variable specified in the output
	if CheckIsInString(Out, Token) {
		Error.Println(Out)
	}

	/*
	  {
	    "token" : "5f84dc06-db56-4800-8fa1-67e3f71bbd41",
	    "sessionId" : 5
	  }
	*/

	//raw output
	//Verbose.Println("RAW output: ", Out)

	var JSONUnmarshalOut map[string]interface{}
	JSONUnmarshalOut, State = JSONUnmarshal(Out)

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'TokenGet' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'TokenGet' return values Token:", JSONUnmarshalOut[TokenElement].(string))
	Debug.Println("Function 'TokenGet' return values Session ID:", JSONUnmarshalOut[SessionIDElement].(float64))
	Debug.Println("Function 'TokenGet' return values State:", State)
	Debug.Println("Function 'TokenGet' ended.")

	Verbose.Println("Security Token: " + JSONUnmarshalOut[TokenElement].(string) + " with Session ID: " + strconv.FormatFloat(JSONUnmarshalOut[SessionIDElement].(float64), 'g', -1, 64) + " created.")
	Verbose.Println("Get the Security Token completed")

	//state to OK
	State = false
	return JSONUnmarshalOut[TokenElement].(string), JSONUnmarshalOut[SessionIDElement].(float64), State
}

//TokenDelete is used to delete the Security Session with the token
//return values are the security token(string) and the session id(float64). if an error happened the state is true. Otherwise false.
//The function stops with exit status 20 (The StorageDeviceId must not be empty.)
//example: TokenGet("https", 10.0.0.1, token, StorageDeviceID)
func TokenDelete(p Params) (string, bool) {
	//C:\>curl -k -H "Accept:application/json" -H "Content-Type:application/json" -H "Authorization:Session 178b4507-464e-49a9-9ea9-192cc781f3b7" -X DELETE "https://10.70.5.104/ConfigurationManager/v1/objects/storages/834000470018/sessions/8"

	Debug.Println("Function 'TokenDelete' started.")
	//start timer
	TimeStart := time.Now()

	//state to NOK
	var State bool
	State = true

	var Out string
	Out = ""

	//specify constants

	Verbose.Println("Delete the Security Token started")
	Debug.Println("Delete the Security Token: " + p.Token + " with Session ID: " + strconv.FormatFloat(p.SessionID, 'g', -1, 64))

	//check if StorageDeviceID is not empty
	if p.StorageDeviceID == "" {
		Error.Println("The StorageDeviceId must not be empty")
		os.Exit(20)
	}

	Debug.Println("Token: ", p.Token, "SessionID: ", p.SessionID)
	p.URL = p.Protocol + "://" + p.Host + ":" + p.Port + "/ConfigurationManager/v1/objects/storages/" + p.StorageDeviceID + "/sessions/" + strconv.FormatFloat(p.SessionID, 'g', -1, 64)
	Debug.Println("URL: " + p.URL)
	//Verbose.Println("Token: " + Token)
	p.RequestType = "DELETE"
	Out = HTTPRequest(p)

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'TokenDelete' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'TokenDelete' return values Out:", Out)
	Debug.Println("Function 'TokenDelete' return values State:", State)
	Debug.Println("Function 'TokenDelete' ended.")

	Verbose.Println("Security Token: " + p.Token + " with Session ID: " + strconv.FormatFloat(p.SessionID, 'g', -1, 64) + " deleted")
	Verbose.Println("Delete the Security Token completed")

	//state to OK
	State = false
	return Out, State
}

//JSONUnmarshal returns the Unmashalled JSON response
func JSONUnmarshal(JSON string) (map[string]interface{}, bool) {
	Debug.Println("Function 'JSONUnmarshal' started.")

	//start timer
	TimeStart := time.Now()

	//initial state is true that means NOK
	var State bool
	State = true

	//pr(p.Requesttype, p.URL, p.Username, p.Password, p.Token, p.Requestbody)
	//JSONOutput := HTTPRequest(p.RequestType, p.URL, p.Username, p.Password, p.Token, p.RequestBody)

	//check if output is in JSON format
	if IsJSONString(JSON) {
		Error.Println("The RestAPI response is not in JSON format.")
		os.Exit(20)
	}

	//fmt.Println("JSON RAW Data:",JSONOutput)

	//convert to byte
	JSONByt := []byte(JSON)
	//load the output in an unspecified interface as map of string
	var JSONUnmarshaled interface{}
	err := json.Unmarshal(JSONByt, &JSONUnmarshaled)
	if err != nil {
		Error.Println(err)
		os.Exit(21)
	}

	//change the type of the output
	JSONUnmarshaledType := JSONUnmarshaled.(map[string]interface{})
	Debug.Println("JSONUnmarshaledType", JSONUnmarshaledType)

	/*
		//go through the array and look for an array in the array
		for Key1, Value1 := range JSONUnmarshaledType {
			//assert the correct type
			fmt.Println("Key:", Key1)
			switch ValueType := Value1.(type) {
			case bool:
				fmt.Println("value:", Value1.(bool))
				//OutputArray[Key1] = Value1.(bool)
			case string:
				fmt.Println("value:", Value1.(string))
				//OutputArray[Key1] = Value1.(string)
			case float64:
				fmt.Println("value:", Value1.(float64))
				//OutputArray[Key1] = Value1.(float64)
			case []interface{}:
				fmt.Println("value:", Value1)
				fmt.Println(ValueType[0])
				for i, u := range ValueType {
					fmt.Println(i, u)
				}

			}
		}
	*/

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'JSONUnmarshal' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'JSONUnmarshal' return values JSONUnmarshaledType:", JSONUnmarshaledType)
	Debug.Println("Function 'JSONUnmarshal' return values State:", State)
	Debug.Println("Function 'JSONUnmarshal' ended.")

	//state to OK
	State = false
	return JSONUnmarshaledType, State
}

//IsJSONString checks if the string is a valid JSON format
func IsJSONString(s string) bool {
	Debug.Println("Function 'IsJSONString' started.")
	//start timer
	TimeStart := time.Now()

	var js string

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'JSONUnmarshal' - Elapsed time ", TimeDiff)
	//Debug.Println("Function 'JSONUnmarshal' return values State:", State)
	Debug.Println("Function 'JSONUnmarshal' ended.")

	return json.Unmarshal([]byte(s), &js) == nil

}

//IsJSONUnmarshal checks if the output is a valid JSON format
func IsJSONUnmarshal(s string) bool {
	Debug.Println("Function 'IsJSONUnmarshal' started.")
	//start timer
	TimeStart := time.Now()

	var js map[string]interface{}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'IsJSONUnmarshal' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'IsJSONUnmarshal' return values State:", State)
	Debug.Println("Function 'IsJSONUnmarshal' ended.")

	return json.Unmarshal([]byte(s), &js) == nil
}

//HTTPRequest is used to send a http(s) request with user and password or with a security token
//return value (string) is the output body.
//The function stops exit code 100 (The requesttype was set other than "GET, "POST", "DELETE".)
//The function stops exit code 101 (A webrequest cannot be executed as no user nor a token was specified.)
//The function stops exit code 102 ("A webrequest cannot be executed with an empty password.)
//example: HttpRequest("GET", protocol, url, username, passwd, token)
func HTTPRequest(p Params) string {
	Debug.Println("Function 'HTTPRequest' started.")
	//start timer
	TimeStart := time.Now()

	//initial state is false that means OK
	var State bool
	// State = NOK
	State = true

	var Out string

	// check protocol
	Out, State = CheckProtocol(p.Protocol)
	if State {
		Error.Println(Out)
	}

	//if the host/IP does not answer to the request
	Out, State = CheckHostExists(p.Host)
	if State {
		Error.Println(Out)
	}

	var RequesttypGet, RequesttypPost, RequesttypDelete string
	RequesttypGet = "GET"
	RequesttypPost = "POST"
	RequesttypDelete = "DELETE"

	//skip unsecure certificate
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest(p.RequestType, p.URL, nil)
	switch p.RequestType {
	case RequesttypGet: //GET
		Debug.Println("Webrequest type: " + p.RequestType)
		//GET
		req, err = http.NewRequest(p.RequestType, p.URL, nil)
		// prefer token if exists
		if p.Token != "" {
			req.Header.Set("Authorization", "Session "+p.Token)
			Debug.Println("Authorization: Session " + p.Token)
		} else {
			if p.Username != "" && p.Password != "" {
				Debug.Println("Set basic authorization with user: " + p.Username)
				req.SetBasicAuth(p.Username, p.Password)
			} else {
				Error.Println("A webrequest cannot be executed as no token or username/password was specified.")
				//stops with exit code 101
				os.Exit(101)
			}
		}
	case RequesttypPost: //POST
		Debug.Println("Webrequest type: " + p.RequestType)
		Debug.Println("Requestbody:" + p.RequestType)
		//POST
		if p.RequestBody == "" {
			req, err = http.NewRequest(p.RequestType, p.URL, nil)
		} else {
			req, err = http.NewRequest(p.RequestType, p.URL, strings.NewReader(p.RequestBody))
		}
		// prefer token if exists
		if p.Token != "" {
			req.Header.Set("Authorization", "Session "+p.Token)
			Debug.Println("Authorization: Session " + p.Token)
		} else {
			if p.Username != "" && p.Password != "" {
				Debug.Println("Set basic authorization with user: " + p.Username)
				req.SetBasicAuth(p.Username, p.Password)
			} else {
				Error.Println("A webrequest cannot be executed as no token or username/password was specified.")
				//stops with exit code 101
				os.Exit(101)
			}
		}
	case RequesttypDelete: //DELETE
		Debug.Println("Webrequest type: " + p.RequestType)
		//DELETE
		jsonStr := []byte(`{"force": false}`)
		req, err = http.NewRequest(p.RequestType, p.URL, bytes.NewBuffer(jsonStr))
		if p.Token == "" {
			Error.Println("A webrequest cannot be executed as no token was specified.")
			//stops with exit code 102
			os.Exit(102)
		}
		req.Header.Set("Authorization", "Session "+p.Token)
		Debug.Println("Authorization: Session " + p.Token)
	default: //OTHER
		Error.Println("The requesttype must be 'GET', 'POST' or 'DELETE'. The specified requesttype is wrong(" + p.RequestType + ").")
		os.Exit(100)
	}

	req.Header.Set("Accept", "application/json")
	Debug.Println("Accept: application/json")
	req.Header.Set("Content-Type", "application/json")
	Debug.Println("Content-Type: application/json")

	Debug.Println("Waiting for the response ...")

	resp, err := client.Do(req)
	if err != nil {
		//panic(err)
		Error.Println("The webrequest cannot be executed as the Host/IP does not exist or Port number does not match ('" + p.URL + "').")
		//stops with exit code 1
		os.Exit(100)
	}
	defer resp.Body.Close()

	Debug.Println("Response Status:", resp.Status)
	Debug.Println("Response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	Debug.Println("Response Body:", string(body))

	if strings.Index(string(body), "message") != -1 {
		//load the output in an unspecified interface as map of string
		//already declared
		var parsed map[string]interface{}
		err := json.Unmarshal(body, &parsed)
		if err != nil {
			Error.Println("JSON parsing error (Unmarshal function threw an error).")
			os.Exit(103)
		}
		//throw an error with the message
		Error.Println("The request with the URL:\"" + p.URL + "\" with requesttype:\"" + p.RequestType + "\" ended with an error.")
		Error.Println("Error message: " + parsed["message"].(string))
		os.Exit(104)
	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'HTTPRequest' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'HTTPRequest' return values State:", State)
	Debug.Println("Function 'HTTPRequest' ended.")

	return string(body)
}

//CheckProtocol is used to check if the protocol is http or https
//return value (string) is the message output. state is true if NOK or false if OK
//The function stops exit code 200 ("The protocol specified is not correct. The only available options are 'http' or 'https'.")
func CheckProtocol(protocol string) (string, bool) {
	Debug.Println("Function 'CheckProtocol' started.")
	//start timer
	TimeStart := time.Now()
	// check protocol
	if (protocol != "http") && (protocol != "https") {
		//throw an error an strop the program
		Error.Println("The protocol specified is not correct. The only available options are 'http' or 'https'.")
		os.Exit(200)
		//return "The protocol specified is not correct. The only available options are 'http' or 'https'.", true
	}
	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'CheckProtocol' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'CheckProtocol' return values State: false")
	Debug.Println("Function 'CheckProtocol' ended.")

	return "", false
}

//CheckHostExists is used to check if the host answers a http(s) request
//return state is true if NOK or false if OK
//The function stops exit code 201 ("The specified host/IP does not answer on requests.")
func CheckHostExists(output string) (string, bool) {
	Debug.Println("Function 'CheckHostExists' started.")
	//start timer
	TimeStart := time.Now()

	if strings.Index(output, "Access Error: 404 -- Not Found") != -1 {
		/*
		  if the server does not respond to http requests
		  <HTML><HEAD><TITLE>Document Error: Not Found</TITLE></HEAD>
		  <BODY><H2>Access Error: 404 -- Not Found</H2>
		  </BODY></HTML>
		*/
		//throw an error an strop the program
		Error.Println("The specified host/IP does not answer on requests.")
		os.Exit(201)
		//return "The specified host/IP does not answer on requests.", true
	}
	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'CheckHostExists' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'CheckHostExists' return values State: false")
	Debug.Println("Function 'CheckHostExists' ended.")

	return "", false
}

//CheckIsInString is used to check if the string contains a certain value
//return value (string) is the message output. state is true if NOK or false if OK
//The function stops exit code 202 ("The specified return value does not contain (" + ToCompare + ").")
func CheckIsInString(output string, ToCompare string) bool {
	Debug.Println("Function 'CheckIsInString' started.")
	//start timer
	TimeStart := time.Now()

	//is the apiVersion variable specified in the output
	if strings.Index(output, ToCompare) == -1 {
		//throw an error an strop the program
		Error.Println("The specified return value does not contain (" + ToCompare + ").")
		os.Exit(202)
		//return The specified return value does not contain (" + ToCompare + ").", true
	}
	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'CheckIsInString' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'CheckIsInString' return values State: false")
	Debug.Println("Function 'CheckIsInString' ended.")

	return false
}

//CheckVersion is used to check if a version is bigger than a certain version
//return value (boolean) is true if the version is below the VersionToCheckAgainst. if it is bigger the false is returned
//The function stops exit code 203 ("The RestAPI version could not be converted.")
//The function stops exit code 204 ("The Storage RestAPI version you are running on (" + InputValue + ") is not supported to provide the data needed.\nIt must be at least version " + strconv.FormatFloat(VersionToCheckAgainst, 'g', -1, 64) + ".")
func CheckVersion(InputValue string, VersionToCheckAgainst string) bool {
	Debug.Println("Function 'CheckVersion' started.")
	//start timer
	TimeStart := time.Now()

	//get the digits out of the version ex: 1.6.4 or 1.11.2
	ActualVersionFirst, err := strconv.ParseInt(InputValue[0:strings.Index(InputValue, ".")], 10, 64)
	if err != nil {
		//throw an error an strop the program
		Error.Println("The actual RestAPI version (" + InputValue + ") could not be converted.")
		os.Exit(203)
	}
	ActualVersionMid, err := strconv.ParseInt(InputValue[strings.Index(InputValue, ".")+1:strings.LastIndex(InputValue, ".")], 10, 64)
	if err != nil {
		//throw an error an strop the program
		Error.Println("The actual RestAPI version (" + InputValue + ") could not be converted.")
		os.Exit(203)
	}
	ActualVersionLast, err := strconv.ParseInt(InputValue[strings.LastIndex(InputValue, ".")+1:], 10, 64)
	if err != nil {
		//throw an error an strop the program
		Error.Println("The actual RestAPI version (" + InputValue + ") could not be converted.")
		os.Exit(203)
	}

	VersionToCheckFirst, err := strconv.ParseInt(VersionToCheckAgainst[0:strings.Index(VersionToCheckAgainst, ".")], 10, 64)
	if err != nil {
		//throw an error an strop the program
		Error.Println("The check RestAPI version (" + VersionToCheckAgainst + ") could not be converted.")
		os.Exit(203)
	}
	VersionToCheckMid, err := strconv.ParseInt(VersionToCheckAgainst[strings.Index(VersionToCheckAgainst, ".")+1:strings.LastIndex(VersionToCheckAgainst, ".")], 10, 64)
	if err != nil {
		//throw an error an strop the program
		Error.Println("The check RestAPI version (" + VersionToCheckAgainst + ") could not be converted.")
		os.Exit(203)
	}
	VersionToCheckLast, err := strconv.ParseInt(VersionToCheckAgainst[strings.LastIndex(VersionToCheckAgainst, ".")+1:], 10, 64)
	if err != nil {
		//throw an error an strop the program
		Error.Println("The check RestAPI version (" + VersionToCheckAgainst + ") could not be converted.")
		os.Exit(203)
	}

	if ActualVersionFirst < VersionToCheckFirst {
		//version to old
		Error.Println("The Storage RestAPI version you are running on (" + InputValue + ") is not supported to provide the data needed.\nIt must be at least version '" + VersionToCheckAgainst + "'.")
		os.Exit(204)
	} else {
		//only check further if the version is equal
		if ActualVersionFirst == VersionToCheckFirst {
			if ActualVersionMid < VersionToCheckMid {
				Error.Println("The Storage RestAPI version you are running on (" + InputValue + ") is not supported to provide the data needed.\nIt must be at least version '" + VersionToCheckAgainst + "'.")
				os.Exit(204)
			} else {
				//only check further if the version is equal
				if ActualVersionMid == VersionToCheckMid {
					if ActualVersionLast < VersionToCheckLast {
						Error.Println("The Storage RestAPI version you are running on (" + InputValue + ") is not supported to provide the data needed.\nIt must be at least version '" + VersionToCheckAgainst + "'.")
						os.Exit(204)
					}
				}
			}
		}
	}

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'CheckVersion' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'CheckVersion' return values State: false")
	Debug.Println("Function 'CheckVersion' ended.")

	return false
}

//HelpOutput creates the help output
func HelpOutput(Version string) {
	Debug.Println("Function 'HelpOutput' started.")
	//start timer
	TimeStart := time.Now()

	LineIn := "  "
	SecondLineIn := "  "

	//Version
	fmt.Println("VERSION:")
	fmt.Println(LineIn + Version)
	fmt.Println()
	fmt.Println("NAME:")
	fmt.Fprintf(os.Stderr, LineIn+"%s\n", os.Args[0])
	fmt.Println()
	fmt.Println("SYNOPSIS:")
	fmt.Fprintf(os.Stderr, LineIn+"%s -user <username> -password <password> [-host <hostname/IP>] [-port <HttpRequestPortnumber>] [-type pool/reserve] [-verbose]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, LineIn+"%s --user <username> --password <password> [--host <hostname/IP>] [--port <HttpRequestPortnumber>] [--type pool/reserve] [--verbose]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, LineIn+"%s -h/-help\n", os.Args[0])
	fmt.Fprintf(os.Stderr, LineIn+"%s --h/--help\n", os.Args[0])
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println(LineIn + "This Script generates two different outputs. One shows the important Pool values and the other one shows all reserves on any LUN on the Storage System.")
	fmt.Println()
	fmt.Println("REQUIREMENT:")
	fmt.Println(LineIn + "None.")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println(LineIn + "All options can be used with one dash ('-') or two dashes ('--')")
	//host option
	fmt.Println(LineIn + "-host string")
	fmt.Println(LineIn + SecondLineIn + "host to send request to. (Optional) (default 'localhost')")
	//user option
	fmt.Println(LineIn + "-user string")
	fmt.Println(LineIn + SecondLineIn + "User you want to use to contact the storage. This must be the storage user even if you contact the HCS Rest API (Required)")
	//password option
	fmt.Println(LineIn + "-password string")
	fmt.Println(LineIn + SecondLineIn + "Password you want to use to contact the storage. This must be the storage password even if you contact the HCS Rest API (Required)")
	//port option
	fmt.Println(LineIn + "-port string")
	fmt.Println(LineIn + SecondLineIn + "Port to be used to contact the host. The storage RestAPI uses 443 (https). The HCS Rest API uses 23451. (Optional) (default '443')")
	//output option
	fmt.Println(LineIn + "-output string")
	fmt.Println(LineIn + SecondLineIn + "Specify the way you want to send the output to. Options are 'stdout' or 'csv'. 'stdout' sends the output to the command line. 'csv' create a file containing comma separated data. (Optional) (default 'stdout')")
	//type option
	fmt.Println(LineIn + "-type string")
	fmt.Println(LineIn + SecondLineIn + "Sets the type of output you want. 'pool' get all pool data. 'reserve' gets you all LUNs/LDEVs that have a reserve. (Optional) (default 'pool')")
	//verbose option
	fmt.Println(LineIn + "-verbose")
	fmt.Println(LineIn + SecondLineIn + "Sets the output mode to verbose. (Optional)")
	//trace option
	fmt.Println(LineIn + "-trace")
	fmt.Println(LineIn + SecondLineIn + "Sets the output mode to trace. Only needed for troubleshooting. (Optional)")
	fmt.Println()
	fmt.Println("FILES:")
	fmt.Fprintf(os.Stderr, LineIn+"%s\n", os.Args[0])
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println(LineIn + "All options can be used with one dash ('-') or two dashes ('--')")
	fmt.Println(LineIn + "Shows help")
	fmt.Fprintf(os.Stderr, LineIn+"%s -h\n", os.Args[0])
	fmt.Println(LineIn + "Shows the pool output. It connects to the restserver on host localhost with the user credentials in table format")
	fmt.Fprintf(os.Stderr, LineIn+"%s -user restuser -password restpass\n", os.Args[0])
	fmt.Println(LineIn + "Shows the pool output. It connects to the restserver on host localhost with the user credentials in csv format")
	fmt.Fprintf(os.Stderr, LineIn+"%s -user restuser -password restpass -output csv\n", os.Args[0])
	fmt.Println(LineIn + "Shows the pool output. It connects to the restserver on host 10.0.1.1 on port 23451 (HCS) with the user credentials in table format and shows detailed logging output")
	fmt.Fprintf(os.Stderr, LineIn+"%s -user restuser -password restpass -host 10.0.1.1 -port 23451 -verbose\n", os.Args[0])
	fmt.Println(LineIn + "Shows all the reserves on LUNs on a Storage System. It connects to the restserver on host 10.0.1.1 on port 23451 (HCS) with the user credentials in table format")
	fmt.Fprintf(os.Stderr, LineIn+"%s -user restuser -password restpass -host 10.0.1.1 -port 23451 -type reserve\n", os.Args[0])
	fmt.Println()

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'HelpOutput' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'HelpOutput' ended.")
}

// -------------------------------------
// Helpful func
// -------------------------------------

//RoundFloat64 rounds up a float64 value up to a certain digit
//val float64 is the value you want to round
//places int is the number of digits you want to have after the round function
func RoundFloat64(val float64, places int) (newVal float64) {
	Debug.Println("Function 'RoundFloat64' started.")
	//start timer
	TimeStart := time.Now()

	//this would be the request if you want to specify the roundOn value
	//func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	//after this value you round up (normally you start to round up the value 0.5)
	const roundOn float64 = 0.5
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow

	TimeEnd := time.Now()
	TimeDiff := TimeEnd.Sub(TimeStart)
	Debug.Println("Function 'RoundFloat64' - Elapsed time ", TimeDiff)
	Debug.Println("Function 'RoundFloat64' return values State: ", newVal)
	Debug.Println("Function 'RoundFloat64' ended.")

	return newVal
}
