package scrapper

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

var baseURL string = "https://ca.indeed.com/jobs?q=golang"
var lastPageURL string = "https://ca.indeed.com/jobs?q=golang&start=9999"

type extractedJob struct {
	id       string
	title    string
	company  string
	location string
	salary   string
}

func main() {
	mainC := make(chan []extractedJob)
	var combinedJobs []extractedJob
	totalPages := getPages()

	for i := 0; i < totalPages; i++ {
		go getPage(i, mainC)
	}

	for i := 0; i < totalPages; i++ {
		extractedJobs := <-mainC
		combinedJobs = append(combinedJobs, extractedJobs...)
	}

	writeJobs(combinedJobs)

	fmt.Println("DONE")
}

func writeJobs(jobs []extractedJob) {
	file, err := os.Create("jobs.csv")
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"Link", "Title", "Company", "Location", "Salary"}
	wError := w.Write(headers)
	checkErr(wError)

	for _, job := range jobs {
		jobSlide := []string{"https://ca.indeed.com/viewjob?jk=" + job.id, job.title, job.company, job.location, job.salary}
		wError := w.Write(jobSlide)
		checkErr(wError)
	}
}

func getPage(page int, mainC chan<- []extractedJob) {
	c := make(chan extractedJob)

	var jobs []extractedJob
	pageURL := baseURL + "&start=" + strconv.Itoa(page*10)

	resp, err := http.Get(pageURL)
	checkErr(err)
	checkCode(resp)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	searchCards := doc.Find(".job_seen_beacon")

	searchCards.Each(func(i int, s *goquery.Selection) {
		go extractJob(s, c)

	})

	for i := 0; i < searchCards.Length(); i++ {
		job := <-c
		jobs = append(jobs, job)
	}

	mainC <- jobs
}

func extractJob(s *goquery.Selection, c chan<- extractedJob) {
	id, _ := s.Find(".jcs-JobTitle").Attr("data-jk")
	title := s.Find(".jcs-JobTitle>span").Text()
	company := s.Find(".companyName>a").Text()
	location := s.Find(".companyLocation").Text()
	salary := s.Find(".attribute_snippet").Text()

	c <- extractedJob{
		id:       id,
		title:    title,
		company:  company,
		location: location,
		salary:   salary,
	}
}

func getPages() int {
	pages := 0
	resp, err := http.Get(lastPageURL)
	checkErr(err)
	checkCode(resp)

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find(".pagination-list").Each(func(i int, s *goquery.Selection) {
		pages, _ = strconv.Atoi(s.Find("b").Text())
	})

	return pages
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(r *http.Response) {
	if r.StatusCode != 200 {
		log.Fatalln("Request failed with Status:", r.StatusCode)
	}
}
