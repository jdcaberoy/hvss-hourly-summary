package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"io"
	"time"
	"strconv"
)

type hourlyCfg struct {
	date time.Time
	title string
	values []float64
}

type hourlyCfgs []hourlyCfg

func main(){

	// open TR1.CSV and TR2.CSV
	tr1, tr2 := openFiles()
	// open another copy for oil2 and wind2
	tr1c, tr2c := openFiles()

	// user inputs month, date, and year
	date := inputDateValues()
	var err error

	tf1cfgs := generateTF1Cfgs(date)
	tf2cfgs := generateTF2Cfgs(date)

	tf1cfgs[0].values, err = interpretHourly(tr1, tf1cfgs[0])
	if err!= nil{
		fmt.Printf("error interpreting oil1 values")
	}
	tf1cfgs[1].values, err = interpretHourly(tr1c, tf1cfgs[1])
	if err!= nil{
		fmt.Printf("error interpreting wind1 values")
	}

	tf2cfgs[0].values, err = interpretHourly(tr2, tf2cfgs[0])
	if err!= nil{
		fmt.Printf("error interpreting oil2 values")
	}
	tf2cfgs[1].values, err = interpretHourly(tr2c, tf2cfgs[1])
	if err!= nil{
		fmt.Printf("error interpreting wind2 values")
	}
	cfgs := append(tf1cfgs, tf2cfgs...)

	for _, v := range cfgs{
		printValues(v)
	}
}

func openFiles()(*os.File, *os.File){
	tr1, err := os.Open("TR1.CSV")
	if err != nil {
		fmt.Printf("error in opening TR 1 file : %v", err)
		panic(err)
	}
	tr2, err := os.Open("TR2.CSV")
	if err != nil{
		fmt.Printf("error in opening TR 2 file: %v", err)
		panic(err)
	}
	return tr1, tr2
}

func inputDateValues()time.Time{
	var inputmonth, inputday, inputyear int
	today := time.Now()
	fmt.Println("input month:")
	fmt.Scanln(&inputmonth)
	fmt.Println("input day:")
	fmt.Scanln(&inputday)
	fmt.Println("input year:")
	fmt.Scanln(&inputyear)
	if inputmonth == 0 {
		today.Month()
	}
	if inputday == 0 {
		today.Day()
	}
	if inputyear == 0 {
		today.Year()
	}
	return time.Date(inputyear,time.Month(inputmonth),inputday,0,0,0,0,time.Local)
}

func generateTF1Cfgs(date time.Time)hourlyCfgs{
	oil1 := hourlyCfg{
		date: date,
		title: "NO.1 OIL TEMPERATURE",
	}
	wind1 := hourlyCfg{
		date: date,
		title: "NO.1 WINDING TEMPERATURE",
	}
	return hourlyCfgs{oil1, wind1}
}

func generateTF2Cfgs(date time.Time)hourlyCfgs{
	oil2 := hourlyCfg{
		date: date,
		title: "NO.2 OIL TEMPERATURE",
	}
	wind2 := hourlyCfg{
		date: date,
		title: "NO.2 WINDING TEMPERATURE",
	}
	return hourlyCfgs{oil2, wind2}
}

func interpretHourly (f *os.File, cfg hourlyCfg)([]float64, error){
	var err error
	reader := csv.NewReader(f)
	reader.Comma = ','
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	var hourlyValues []float64
	timeValue := make(map[time.Time]float64)

	// Read the headers of csv
	if _, err = reader.Read(); err != nil {
		fmt.Printf("err check1err: %v, title: %v", err, cfg.title)
		return hourlyValues, err
	}
	// Read all if implementation is easier in the future
	// Using reader.ReadAll() will get error on 2nd run for the next titles to be searched
	// rows, err := reader.ReadAll()
	// if err != nil {
	// 	fmt.Printf("error reading all rows: %v", err)
	// }
	// fmt.Printf("rows[0]: %v", rows[0][1])
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("err in reading csv row: %v", err)
		}
		csvTitle := row[1]
		if csvTitle != cfg.title {
			continue
		}
		dateString := row[2]
		csvDate, err := time.Parse("02/01/2006 15:04:05", dateString)
		if err != nil {
			fmt.Printf("error parsing csv time: %v", err)
			return []float64{}, err
		}
		csvValues, err := strconv.ParseFloat(strings.TrimSpace(row[6]), 64)
		if err != nil {
			fmt.Printf("error parsing csv float values: %v", err)
			return []float64{}, err
		}
		// save date/time 
		timeValue[csvDate] = csvValues
	}
	// fmt.Printf("rows of rows check: %v", len(rows))
	// for i:=0;i<len(rows);i++{
	// 	for _, v := range rows[i]{
	// 		fmt.Printf("v: %v\nlenght of rows: %v\n",v, len(v))
	// 		break
	// 		// if csvTitle != cfg.title {
	// 		// 	continue
	// 		// }
	// 		// dateString := row[2]
	// 		// csvDate, err := time.Parse("02/01/2006 15:04:05", dateString)
	// 		// if err != nil {
	// 		// 	fmt.Printf("error parsing time: %v", err)
	// 		// }
	// 		// csvValues, err := strconv.ParseFloat(strings.TrimSpace(row[6]), 64)
	// 		// if err != nil {
	// 		// 	fmt.Printf("error parsing values: %v", err)
	// 		// }
	// 		// // fmt.Printf("map sample: %v", timeValue)
	// 		// timeValue[csvDate] = csvValues
	// 	}
	// 	break
	// }
	// targetTime := cfg.date.Add((time.Hour * time.Duration(i)) + (time.Minute * 50))
	
	// get values hourly in a 24 hour period
	for i:=0; i<24; i++ {
		targetTime := cfg.date.Add((time.Hour * time.Duration(i)) + (time.Minute * 50))
		for k, v := range timeValue {
			// if value is between allotted time of H:50 to H:60, save in array and break loop
			if k.After(targetTime) && k.Before(targetTime.Add(time.Minute * 10)){
				hourlyValues = append(hourlyValues, v)
				// DEBUGGER
				// fmt.Printf("target time: %v, hourly value: %v, timevalue: %v\n", targetTime, v, k)
				break
			}
		}
	}
	return hourlyValues, nil
}

func printValues (cfg hourlyCfg){
	fmt.Printf("TITLE: %v\n", cfg.title)
	for k, v := range cfg.values{
		fmt.Printf("hour: %v = %v\n", k, v)
	}
	fmt.Printf("\n")
}