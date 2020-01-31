package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
)

type ViewData struct {
	Title   string
	Servers []string
}

type ServerReportItem struct {
	Title  string
	Link   string
	Report [][]string
}

type ServerDetail struct {
	Title             string
	ServerId          string
	InstanceName      string
	InstanceType      string
	OsName            string
	Purpose           string
	IpAdr             string
	Passwd            string
	CpuUtil           string
	RamUtil           string
	RedCloakInstalled string
	Infected          string
	Comments          string
	Analize           string
	Items             []ServerReportItem
}

type Stat struct {
	Count     int
	Processes map[string]int
	Packages  map[string]int
}

type ServerStat struct {
	Stat map[string]Stat
}

var ServersStats ServerStat = ServerStat{Stat: make(map[string]Stat)}

var Titles map[string]string

func CalcStat() {

	dirs, err := ioutil.ReadDir("./reports")
	if err != nil {
		log.Fatal(err)
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			// Get OS name setup on the server
			hostFile, opnErr := os.Open("./reports/" + dir.Name() + "/host-report.csv")
			if opnErr != nil {
				log.Print(opnErr)
			}

			reader := csv.NewReader(bufio.NewReader(hostFile))

			var rdErr error
			var record []string
			for rdErr != io.EOF {
				record, rdErr = reader.Read()
				if rdErr != nil && rdErr != io.EOF {
					fmt.Println("Error reading file ", hostFile.Name(), rdErr)
				}

				if len(record) == 0 {
					fmt.Println("Records is empty in:", hostFile.Name())
					continue
				}
				if distIdx := strings.Index(record[0], "Distributor ID"); distIdx != -1 {
					osName := record[1]
					osName = strings.TrimSpace(osName)
					statOS, ok := ServersStats.Stat[osName]
					if !ok {
						statOS = Stat{
							Processes: make(map[string]int, 1),
							Packages:  make(map[string]int, 1),
						}
					}
					statOS.Count++

					// Analyze processes
					procesesFile, opnErr := os.Open("./reports/" + dir.Name() + "/run-report.csv")
					if opnErr != nil {
						log.Print(opnErr)
					}
					procReader := csv.NewReader(bufio.NewReader(procesesFile))

					var rdProcErr error
					var procRecord []string

					for rdProcErr != io.EOF {
						procRecord, rdProcErr = procReader.Read()

						if rdProcErr != nil && rdProcErr != io.EOF {
							fmt.Println("Error reading package file!", rdProcErr.Error(), "./reports/"+dir.Name()+"/run-report.csv")
							break
						}

						if len(procRecord) < 11 {
							fmt.Println("Records is empty or column count < 11:", "./reports/"+dir.Name()+"/run-report.csv")
							continue
						}
						statOS.Processes[procRecord[10]]++
					}
					procesesFile.Close()

					//Analyze packages
					pkgFile, opnErr := os.Open("./reports/" + dir.Name() + "/pkg-report.csv")
					if opnErr != nil {
						log.Print(opnErr)
					}
					pkgReader := csv.NewReader(bufio.NewReader(pkgFile))

					var rdPkgErr error
					var pkgRecord []string

					for rdPkgErr != io.EOF {
						pkgRecord, rdPkgErr = pkgReader.Read()

						if rdPkgErr != nil && rdPkgErr != io.EOF {
							fmt.Println("Error reading package file!", rdPkgErr.Error(), "./reports/"+dir.Name()+"/pkg-report.csv")
							break
						}

						if len(pkgRecord) == 0 {
							fmt.Println("Records is empty in:", "./reports/"+dir.Name()+"/pkg-report.csv")
							continue
						}

						if osName == "CentOS" {
							columns := strings.FieldsFunc(pkgRecord[0], func(c rune) bool {
								if c == ':' {
									return true
								}
								return false
							})
							statOS.Packages[columns[0]]++
						}

						if osName == "Ubuntu" {
							statOS.Packages[pkgRecord[0]]++
						}
					}
					pkgFile.Close()

					ServersStats.Stat[osName] = statOS

					break
				}
			}
			hostFile.Close()
		}
	}
}

func InitTitles() {
	Titles = make(map[string]string)

	Titles["host-report.csv"] = "Host:"
	Titles["pkg-report.csv"] = "Installed packages:"
	Titles["run-report.csv"] = "Running processes:"
	Titles["service-report.csv"] = "Services:"
	Titles["connections-report.csv"] = "Connections:"
	Titles["mysqldb-report.csv"] = "MySQL DBs:"
	Titles["mongodb-report.csv"] = "Mongo DBs:"
	Titles["webapps-report.csv"] = "Web Apps"
}

func main() {

	InitTitles()
	CalcStat()

	data := ViewData{
		Title:   "Balfour Servers List",
		Servers: []string{},
	}

	server := ServerDetail{
		Title: "Server's detailed report",
		Items: make([]ServerReportItem, 0, 1),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dirs, err := ioutil.ReadDir("./reports")
		if err != nil {
			log.Fatal(err)
		}
		data.Servers = data.Servers[:0]
		for _, dir := range dirs {
			if dir.IsDir() {
				data.Servers = append(data.Servers, dir.Name())
			}
		}
		tmpl, _ := template.ParseFiles("templates/index.html")
		tmpl.Execute(w, data)
	})
	// Process statistics
	http.HandleFunc("/proceses", func(w http.ResponseWriter, r *http.Request) {

		tmpl, _ := template.ParseFiles("templates/proceses.html")
		tmpl.Execute(w, ServersStats)
	})
	// Package statistics
	http.HandleFunc("/packages", func(w http.ResponseWriter, r *http.Request) {

		tmpl, _ := template.ParseFiles("templates/packages.html")
		tmpl.Execute(w, ServersStats)
	})
	// Server details
	http.HandleFunc("/server_detail", func(w http.ResponseWriter, r *http.Request) {

		var dir string
		var serverName string

		server.InstanceType = "VM"

		if r.Method == http.MethodPost {
			server.InstanceName = r.FormValue("InstanceName")
			server.InstanceType = r.FormValue("InstanceType")
			server.OsName = r.FormValue("OsName")
			server.Purpose = r.FormValue("Purpose")
			server.IpAdr = r.FormValue("IpAdr")
			server.Passwd = r.FormValue("Passwd")
			server.CpuUtil = r.FormValue("CpuUtil")
			server.RamUtil = r.FormValue("RamUtil")
			server.RedCloakInstalled = r.FormValue("RedCloakInstalled")
			server.Infected = r.FormValue("Infected")
			server.Comments = r.FormValue("Comments")
			server.Analize = r.FormValue("Analize")

			dir = "./reports/" + r.FormValue("server")
			serverName = r.FormValue("server")
			server.ServerId = serverName
			serverName = serverName[:(len(serverName) - 8)]
			server.Title = serverName
			// Write to csv file

			aboutRecords := make([][]string, 0)

			s := reflect.ValueOf(&server).Elem()
			typeOfT := s.Type()

			for i := 0; i < s.NumField(); i++ {
				f := s.Field(i)
				name := typeOfT.Field(i).Name
				if name == "Items" || name == "Title" || name == "ServerId" {
					continue
				}
				record := make([]string, 2)
				record[0] = name
				record[1] = f.Interface().(string)
				aboutRecords = append(aboutRecords, record)
			}

			csvFile, err := os.Create(dir + "/about.csv")

			if err != nil {
				log.Fatalf("failed creating file: %s", err)
			}

			csvwriter := csv.NewWriter(csvFile)
			wrErr := csvwriter.WriteAll(aboutRecords)
			if wrErr != nil {
				fmt.Println("Error writing file %s", wrErr.Error())
			}
			fmt.Println("About server data saved to %s", dir+"/about.csv")
			csvFile.Close()

		} else {

			urlQuery := r.URL.Query()
			dir = "./reports/" + urlQuery.Get("server")
			serverName = urlQuery.Get("server")
			server.ServerId = serverName
			serverName = serverName[:(len(serverName) - 8)]
			server.Title = serverName
		}

		server.InstanceName = serverName

		files, err := ioutil.ReadDir(dir)
		if err != nil {
			fmt.Println("Error reading dir", dir, err)
			return
		}

		// Set server detail params
		_, statErr := os.Stat(dir + "/about.csv")
		if os.IsExist(statErr) {
			csvAboutFile, _ := os.Open(dir + "/about.csv")
			aboutReader := csv.NewReader(bufio.NewReader(csvAboutFile))

			aboutReader.Comma = ','
			aboutReader.LazyQuotes = true

			aboutProperties := make(map[string]string, 1)
			var aboutRecords []string
			var rdErr error
			for rdErr != io.EOF {
				aboutRecords, rdErr = aboutReader.Read()
				if rdErr != nil && rdErr != io.EOF {
					continue
				}
				aboutProperties[aboutRecords[0]] = aboutRecords[1]
			}

			server.InstanceName = aboutProperties["InstanceName"]
			server.InstanceType = aboutProperties["InstanceType"]
			server.OsName = aboutProperties["OsName"]
			server.Purpose = aboutProperties["Purpose"]
			server.IpAdr = aboutProperties["IpAdr"]
			server.Passwd = aboutProperties["Passwd"]
			server.CpuUtil = aboutProperties["CpuUtil"]
			server.RamUtil = aboutProperties["RamUtil"]
			server.RedCloakInstalled = aboutProperties["RedCloakInstalled"]
			server.Infected = aboutProperties["Infected"]
			server.Comments = aboutProperties["Comments"]
			server.Analize = aboutProperties["Analize"]

			csvAboutFile.Close()

		}

		// -------------------------
		server.Items = server.Items[:0]
		for _, file := range files {
			if !file.IsDir() {

				if file.Name() == "about.csv" {
					continue
				}

				csvFile, _ := os.Open(dir + "/" + file.Name())
				reader := csv.NewReader(bufio.NewReader(csvFile))

				reader.Comma = ','
				reader.LazyQuotes = true

				records, err := reader.ReadAll()
				if err != nil {
					fmt.Println("Parsing error ", err.Error(), file.Name())
				}
				link := file.Name()
				link = link[:(len(link) - 11)]
				item := ServerReportItem{
					Title: Titles[file.Name()],
					Link:  link,
				}
				item.Report = records
				server.Items = append(server.Items, item)
			}
		}

		tmpl, _ := template.ParseFiles("templates/server_detail.html")
		tmpl.Execute(w, server)
	})

	// About server form
	http.HandleFunc("/about_server", func(w http.ResponseWriter, r *http.Request) {

		var dir string
		var serverName string
		urlQuery := r.URL.Query()
		dir = "./reports/" + urlQuery.Get("server")
		serverName = urlQuery.Get("server")
		server.ServerId = serverName
		serverName = serverName[:(len(serverName) - 8)]
		server.Title = serverName

		// Set server detail params
		_, statErr := os.Stat(dir + "/about.csv")
		if os.IsExist(statErr) {
			csvAboutFile, _ := os.Open(dir + "/about.csv")
			aboutReader := csv.NewReader(bufio.NewReader(csvAboutFile))

			aboutReader.Comma = ','
			aboutReader.LazyQuotes = true

			aboutProperties := make(map[string]string, 1)
			var aboutRecords []string
			var rdErr error
			for rdErr != io.EOF {
				aboutRecords, rdErr = aboutReader.Read()
				if rdErr != nil && rdErr != io.EOF {
					continue
				}
				aboutProperties[aboutRecords[0]] = aboutRecords[1]
			}

			server.InstanceName = aboutProperties["InstanceName"]
			server.InstanceType = aboutProperties["InstanceType"]
			server.OsName = aboutProperties["OsName"]
			server.Purpose = aboutProperties["Purpose"]
			server.IpAdr = aboutProperties["IpAdr"]
			server.Passwd = aboutProperties["Passwd"]
			server.CpuUtil = aboutProperties["CpuUtil"]
			server.RamUtil = aboutProperties["RamUtil"]
			server.RedCloakInstalled = aboutProperties["RedCloakInstalled"]
			server.Infected = aboutProperties["Infected"]
			server.Comments = aboutProperties["Comments"]
			server.Analize = aboutProperties["Analize"]

			csvAboutFile.Close()

		} else {
			server.InstanceName = serverName
			server.InstanceType = "VM"
		}

		// -------------------------
		server.Items = server.Items[:0]

		tmpl, _ := template.ParseFiles("templates/about_form.html")
		tmpl.Execute(w, server)
	})

	fmt.Println("Server is listening...")
	http.ListenAndServe(":8181", nil)
}
