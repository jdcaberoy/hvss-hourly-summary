package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
)

type hourlyCfg struct {
	date   time.Time
	title  string
	values []float64
}

type hourlyCfgs []hourlyCfg

func main() {

	filesnames := []string{
		"TR1.CSV",
		"TR2.CSV",
		"Daily Operation Performance Monitoring.CSV",
	}
	// open TR1.CSV and TR2.CSV
	f := openFiles(filesnames)

	if len(f) == 0 {
		fmt.Printf("The following files names in {name.CSV} format are accepted: %v. contact JDC for requests.\n", filesnames)
		for {
			fmt.Println("Press CTRL+C to exit...")
			fmt.Scanln()
		}
	}

	date := inputDateValues()

	r := convertToReadables(f)
	// fmt.Printf("DEBUGGER: flag1\n")
	trendList := r.getTrends()
	// fmt.Printf("DEBUGGER: flag2\n")
	// fmt.Printf("DEBUGGER: trend list: %v\n\n", trendList)

	//csv data sorted by trend
	var trendcsv []csvDatas

	// get trend from all csvs and compile to trendcsv variable
	// TODO: compile per trend and per object complete name
	for _, v := range trendList {
		var t csvDatas
		t = r.compilePerTrend(v)
		trendcsv = append(trendcsv, t)
	}

	// fmt.Printf("DEBUGGER: flag3\n")

	// interpret data hourly per trend extracted from csv
	e := excelize.NewFile()

	for _, v := range trendList {
		e.NewSheet(v)
	}

	for _, v := range trendcsv {
		activeTrend := v[0].trend
		offset := 2
		fmt.Printf("Trend: %v\n", activeTrend)
		if strings.Contains(activeTrend, "Wh") || strings.Contains(activeTrend, "Rh") {
			lastReading := v[len(v)-1].value
			lastReadingTime := v[len(v)-1].date
			firstReading := v[offset].value
			firstReadingTime := v[offset].date
			total := lastReading - firstReading
			var h hValues
			p := excelParams{
				name:      activeTrend,
				hasTotal:  true,
				onlyTotal: true,
				totalHParams: totalHParams{
					firstReading:     firstReading,
					lastReading:      lastReading,
					firstReadingTime: firstReadingTime,
					lastReadingTime:  lastReadingTime,
					total:            total,
				},
			}

			if activeTrend == "ENERGY MWh EXPORT" {
				fmt.Printf("NOTE: DUE TO STUPID SCADA IMPLEMENTATION, READINGS BEFORE 6AM ARE NOT RECORDED. PLEASE MANUALLY REFER VALUES ON WRITTEN FORMS\n")
				offset--
				h = v.interpretHourly(date)
				h.calculateDValues()
				h.printValues(true)
				p.onlyTotal = false
			}
			h.encodeExcel(e, p)

			fmt.Printf("@hour 0 time = %v	read value = %v\n", firstReadingTime, firstReading)
			fmt.Printf("@hour 24 time = %v	read value = %v\n", lastReadingTime, lastReading)
			fmt.Printf("TOTAL: %.2f\n", total)

			fmt.Printf("\n")
		} else {
			h := v.interpretHourly(date)
			h.calculateDValues()
			h.printValues(false)
			h.encodeExcel(e, excelParams{
				name:     activeTrend,
				hasTotal: false,
			})
			fmt.Printf("\n")
		}
	}

	fileTitle := fmt.Sprintf("Daily Operations  %v-%v-%v.xlsx", date.Year(), int(date.Month()), date.Day())
	err := e.SaveAs(fileTitle)
	if err != nil {
		for {
			fmt.Printf("err: %v\n", err)
			fmt.Println("error creating excel. Press CTRL+C to exit...")
			fmt.Scanln()
		}
	}

	fmt.Println("Excel file created successfully.")
	for {
		fmt.Println("Press CTRL+C to exit...")
		fmt.Scanln()
	}
}

func openFiles(n []string) []os.File {
	var files []os.File

	for _, v := range n {
		f, err := os.Open(v)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("error: file name %v does not exist\n", v)
			} else {
				fmt.Printf("error opening file: %v\n", err)
				panic(err)
			}
		}
		if f != nil {
			files = append(files, *f)
			fmt.Printf("%v successfully opened.\n", v)
		}
	}

	return files
}

func inputDateValues() time.Time {
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
	fmt.Println()

	return time.Date(inputyear, time.Month(inputmonth), inputday, 0, 0, 0, 0, time.UTC)
}

type csvData struct {
	trend  string
	object string
	date   time.Time
	value  float64
}

type csvDatas []csvData

func (c *csvDatas) getTrends() []string {
	var trendList []string
	for _, v := range *c {
		if !isElementExist(trendList, v.trend) {
			trendList = append(trendList, v.trend)
		}
	}
	return trendList
}

func isElementExist(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func (c *csvDatas) compilePerTrend(name string) csvDatas {
	var compiledTrend csvDatas
	for k, v := range *c {
		if v.trend == name {
			compiledTrend = append(compiledTrend, (*c)[k])
		}
	}
	return compiledTrend
}

func convertToReadables(f []os.File) csvDatas {
	var c csvDatas
	for _, v := range f {
		compileCsv(&v, &c)
	}
	return c
}

func compileCsv(f *os.File, c *csvDatas) csvDatas {
	var err error
	reader := csv.NewReader(f)
	reader.Comma = ','
	reader.Comment = '#'
	reader.TrimLeadingSpace = true

	// Read the headers of csv
	if _, err = reader.Read(); err != nil {
		fmt.Printf("error reading csv header: %v", err)
	}

	var d csvDatas

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Printf("err in reading csv row: %v", err)
		}
		var l csvData
		fullObjName := strings.Split(row[0], " / ")
		l.trend = fullObjName[2] + " | " + row[1]

		//max sheet name characters is 22
		if len(fullObjName[2]+" | "+row[1]) > 21 {
			equipmentNameLen := len(fullObjName[2])
			trendName := row[1]
			remainder := 22 - equipmentNameLen
			l.trend = trendName[len(trendName)-remainder:]
		}

		l.date, err = time.Parse("02/01/2006 15:04:05", row[2])
		if err != nil {
			fmt.Printf("error parsing date: %v", err)
			panic(err)
		}
		l.value, err = strconv.ParseFloat(strings.TrimSpace(row[6]), 64)
		if err != nil {
			fmt.Printf("error parsing float: %v", err)
			panic(err)
		}
		d = append(d, l)
	}

	// fmt.Printf("DEBUGGER: len d: %v\n", len(d))

	*c = append(*c, d...)

	// fmt.Printf("DEBUGGER: csv datas lenght: %v\n", len(d))
	return d
}

// func interpretHourly(c csvDatas, date time.Time) map[time.Time]float64 {
// 	hourlyValues := make(map[time.Time]float64)
// 	for i := 0; i < 24; i++ {
// 		targetTime := date.Add((time.Hour * time.Duration(i)) + (time.Minute * 50))
// 		for _, v := range c {
// 			// if value is between allotted time of H:50 to H:60, save in array and break loop
// 			if v.date.After(targetTime) && v.date.Before(targetTime.Add(time.Minute*10)) {
// 				hourlyValues[targetTime] = v.value
// 				// DEBUGGER
// 				// fmt.Printf("target time: %v, hourly value: %v, timevalue: %v\n", targetTime, v, k)
// 				break
// 			}
// 		}
// 	}
// 	return hourlyValues
// }

func (c *csvDatas) interpretHourly(date time.Time) hValues {
	var hourlyValues hValues
	// fmt.Printf("DEBUGGER: Name %v, lenght %v,sample value date: %v	|	%v\n", (*c)[0].trend, len(*c), (*c)[565].value, (*c)[565].date)
	for i := 0; i < 24; i++ {
		targetTime := date.Add((time.Hour * time.Duration(i)) + (time.Minute * 50))
		// fmt.Printf("DEBUGGER: target time: %v\n", targetTime)
		for k, v := range *c {
			// if value is between allotted time of H:50 to H:60, save in array and break loop
			// if v.date.Before(targetTime.Add(time.Hour * 1)) {
			// 	fmt.Printf("DEBUGGER: list\ntarget time %vcsvTimeValue time: %vbefore time: %v\n\n", targetTime, v.date, targetTime.Add(time.Minute*10))
			// }
			if v.date.After(targetTime) && v.date.Before(targetTime.Add(time.Minute*12)) {
				h := hValue{
					time: targetTime.Add(time.Minute * 10),
					// time:  targetTime,
					value: v.value,
				}
				hourlyValues = append(hourlyValues, h)
				// fmt.Printf("DEBUGGER: values per hour %v, time: %v\n", v.value, v.date)
				// DEBUGGER
				// fmt.Printf("target time: %v, hourly value: %v, timevalue: %v\n", targetTime, v, k)
				break
			}
			if k == (len(*c) - 1) {
				h := hValue{
					time: targetTime.Add(time.Minute * 10),
				}
				hourlyValues = append(hourlyValues, h)
			}
		}
	}
	// fmt.Printf("DEBUGGER: hourlyValues[0].value time	%v	%v\n", hourlyValues[0].value, hourlyValues[0].time)
	return hourlyValues
}

type hValue struct {
	time   time.Time
	value  float64
	dvalue float64
}

type hValues []hValue

type excelParams struct {
	name         string
	hasTotal     bool
	onlyTotal    bool
	totalHParams totalHParams
}
type totalHParams struct {
	firstReading     float64
	lastReading      float64
	firstReadingTime time.Time
	lastReadingTime  time.Time
	total            float64
}

func (h *hValues) calculateDValues() {
	// fmt.Printf("DEBUGGER: lenght hvalues during calculateDValues: %v\n", len(*h))
	for k, v := range *h {
		if k == 0 || v.value == 0 || (*h)[k-1].value == 0 {
			continue
		}
		// fmt.Printf("DEBUGGER:\nK: %v\nVbefore: %v\nVnow: %v", k, (*h)[k-1].value, v.value)
		(*h)[k].dvalue = v.value - (*h)[k-1].value
	}
}

func (h *hValues) printValues(printDiff bool) {
	for _, v := range *h {
		x := fmt.Sprintf("Time:%v|	value %.2f", v.time.Format("15:40"), v.value)
		if printDiff {
			x = x + fmt.Sprintf("	|	Difference: %.2f", v.dvalue)
		}
		fmt.Printf("%v\n", x)
		// fmt.Printf("Time:%v|	value %.2f	|	Difference: %.2f\n", v.time, v.value, v.dvalue)
	}
}

func (h *hValues) encodeExcel(e *excelize.File, p excelParams) {

	setDefaultHeaders(e, p.name)

	if p.hasTotal {
		p.totalHParams.setExcelTotalValues(e, p.name)
		if p.onlyTotal {
			return
		}
	}

	for k, v := range *h {
		cellLoc := fmt.Sprintf("%c%d", 'A', k+2)
		e.SetCellValue(p.name, cellLoc, v.time)
	}

	for k, v := range *h {
		cellLoc := fmt.Sprintf("%c%d", 'B', k+2)
		e.SetCellValue(p.name, cellLoc, v.value)
	}

	if (*h).hasDifference() {
		for k, v := range *h {
			e.SetCellValue(p.name, "C1", "Difference")
			cellLoc := fmt.Sprintf("%c%d", 'C', k+2)
			e.SetCellValue(p.name, cellLoc, v.dvalue)
		}
	}
	e.SetCellValue(p.name, "A40", p.name)
}

func setDefaultHeaders(e *excelize.File, name string) {
	e.SetColWidth(name, "A", "A", 20)

	e.SetCellValue(name, "A1", "Time")
	e.SetCellValue(name, "B1", "Value")
}

func (p *totalHParams) setExcelTotalValues(e *excelize.File, name string) {
	e.SetColWidth(name, "E", "F", 18)
	e.SetCellValue(name, "E1", "First Reading")
	e.SetCellValue(name, "F1", p.firstReadingTime)
	e.SetCellValue(name, "G1", p.firstReading)
	e.SetCellValue(name, "E2", "Last Reading")
	e.SetCellValue(name, "F2", p.lastReadingTime)
	e.SetCellValue(name, "G2", p.lastReading)
	// cellLoc := fmt.Sprintf("%c%d", 'E', 1)
	e.SetCellValue(name, "E3", "TOTAL")
	e.SetCellValue(name, "G3", p.total)
}

func (h *hValues) hasDifference() bool {
	hasDifference := false
	for _, v := range *h {
		if v.dvalue != 0.00 {
			hasDifference = true
		}
	}
	return hasDifference
}

// func isFloat64(v interface{}) bool {
// 	return reflect.TypeOf(v).Kind() == reflect.Float64
// }
