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
	Title string
	Items []ServerReportItem
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
			// Get OS name setuped in server
			hostFile, opnErr := os.Open("./reports/" + dir.Name() + "/host-report.csv")
			if opnErr != nil {
				log.Print(opnErr)
			}

			reader := csv.NewReader(bufio.NewReader(hostFile))

			var rdErr error
			var record []string
			for rdErr != io.EOF {
				record, rdErr = reader.Read()
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

					// Analyze proceses
					// procesesFile, opnErr := os.Open("./reports/" + dir.Name() + "/run-report.txt")
					// if opnErr != nil {
					// 	log.Print(opnErr)
					// }
					// procReader := bufio.NewReader(procesesFile)

					// var rdProcErr error
					// var procLine string
					// for true {
					// 	procLine, rdProcErr = procReader.ReadString('\n')

					// 	if rdProcErr == io.EOF {
					// 		break
					// 	}

					// 	if strings.Index(procLine, "%CPU %MEM") != -1 {
					// 		fmt.Println("%CPU %MEM Skiped")
					// 		continue
					// 	}
					// 	procLine = procLine[16:]
					// 	statOS.Processes[procLine]++
					// }
					// procesesFile.Close()

					// Analyze packages
					// pkgFile, opnErr := os.Open("./reports/" + dir.Name() + "/pkg-report.txt")
					// if opnErr != nil {
					// 	log.Print(opnErr)
					// }
					// pkgReader := bufio.NewReader(pkgFile)

					// var rdPkgErr error
					// var pkgLine string
					// var lineCounter int = 0
					// var oldFieldsCount int = 3
					// for true {
					// 	pkgLine, rdPkgErr = pkgReader.ReadString('\n')

					// 	if rdPkgErr == io.EOF {
					// 		break
					// 	}
					// 	lineCounter++
					// 	if osName == "CentOS" {
					// 		if lineCounter <= 2 {
					// 			fmt.Println("CentOS package line Skiped")
					// 			continue
					// 		}

					// 		columns := strings.Fields(pkgLine)

					// 		oldFieldsCount += len(columns)
					// 		if len(columns) == 3 {
					// 			statOS.Packages[columns[0]]++
					// 		} else {
					// 			if oldFieldsCount%3 == 1 {
					// 				statOS.Packages[columns[0]]++
					// 			}
					// 		}
					// 	}

					// 	if osName == "Ubuntu" {
					// 		if lineCounter <= 6 {
					// 			fmt.Println("Ubuntu package line Skiped")
					// 			continue
					// 		}

					// 		columns := strings.Fields(pkgLine)

					// 		statOS.Packages[columns[1]]++
					// 	}

					// }
					// pkgFile.Close()

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

}

func main() {

	InitTitles()
	CalcStat()

	data := ViewData{
		Title:   "Balfour Servers List",
		Servers: []string{},
	}

	server := ServerDetail{
		Title: "Server detail report",
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
	// Proceses detail
	http.HandleFunc("/proceses", func(w http.ResponseWriter, r *http.Request) {

		tmpl, _ := template.ParseFiles("templates/proceses.html")
		tmpl.Execute(w, ServersStats)
	})
	// Pacakges detail
	http.HandleFunc("/packages", func(w http.ResponseWriter, r *http.Request) {

		tmpl, _ := template.ParseFiles("templates/packages.html")
		tmpl.Execute(w, ServersStats)
	})
	// Server detail
	http.HandleFunc("/server_detail", func(w http.ResponseWriter, r *http.Request) {
		urlQuery := r.URL.Query()
		dir := "./reports/" + urlQuery.Get("server")
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Fatal(err)
		}

		serverName := urlQuery.Get("server")
		serverName = serverName[:(len(serverName) - 8)]
		server.Title = "Server: " + serverName + " detailed report"
		server.Items = server.Items[:0]
		for _, file := range files {
			if !file.IsDir() {
				csvFile, _ := os.Open(dir + "/" + file.Name())
				reader := csv.NewReader(bufio.NewReader(csvFile))

				reader.Comma = ','
				reader.LazyQuotes = true

				records, err := reader.ReadAll()
				if err != nil {
					fmt.Println("Parse error ", err.Error(), file.Name())
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

	fmt.Println("Server is listening...")
	http.ListenAndServe(":8181", nil)
}
