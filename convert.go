package xes

import (
	"encoding/csv"
	"encoding/xml"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"
)

// XES represents the structure of the XES file
type XES struct {
	XMLName xml.Name `xml:"log"`
	Trace   []Trace  `xml:"trace"`
}

// Trace represents a trace in the XES file
type Trace struct {
	Event            []Event           `xml:"event"`
	StringAttributes []StringAttribute `xml:"string"`
}

// Event represents an event in the XES file
type Event struct {
	StringAttributes []StringAttribute `xml:"string"`
	DateAttributes   []DateAttribute   `xml:"date"`
}

// StringAttribute represents a string attribute in an event
type StringAttribute struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// DateAttribute represents a date attribute in an event
type DateAttribute struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// GetXESColumn get XES column name
func GetXESColumn(file *multipart.FileHeader) ([]string, []string, XES, error) {
	// Open the XES file
	xesFile, err := file.Open()
	if err != nil {
		return nil, nil, XES{}, err
	}
	defer func(xesFile multipart.File) {
		err := xesFile.Close()
		if err != nil {

		}
	}(xesFile)

	// Parse the XML data
	xes := XES{}
	decoder := xml.NewDecoder(xesFile)
	if err := decoder.Decode(&xes); err != nil {
		log.Println("Error decoding XML:", err)
		return nil, nil, XES{}, err
	}

	// Get all possible attribute keys
	keyMap := make(map[string]struct{})
	for _, trace := range xes.Trace {
		for _, event := range trace.Event {
			for _, stringAttr := range event.StringAttributes {
				keyMap[stringAttr.Key] = struct{}{}
			}
			for _, dateAttr := range event.DateAttributes {
				keyMap[dateAttr.Key] = struct{}{}
			}
		}
	}

	// Convert keys from map to slice
	var keys []string
	for key := range keyMap {
		keys = append(keys, key)
	}

	// Write CSV header
	var header []string
	for _, key := range keys {
		header = append(header, strings.TrimSpace(key))
	}
	header = append(header, "case:concept:name")
	return header, keys, xes, nil
}

// ConvertXESToCSV converts an XES file to a CSV file
func ConvertXESToCSV(header, keys []string, xes XES, csvFilePath string) error {
	// Create the CSV file
	csvFile, errs := os.Create(csvFilePath)
	if errs != nil {
		return errs
	}
	defer func(csvFile *os.File) {
		err := csvFile.Close()
		if err != nil {
		}
	}(csvFile)

	// Write UTF-8 BOM to ensure correct encoding
	_, errs = csvFile.WriteString("\xEF\xBB\xBF")
	if errs != nil {
		return errs
	}

	// Write CSV data
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		log.Println("Error writing CSV header:", err)
		return errs
	}

	// Write XES data to CSV file
	for _, trace := range xes.Trace {
		for _, event := range trace.Event {
			record := make([]string, len(keys)+1)
			for _, stringAttr := range event.StringAttributes {
				index := indexOf(keys, stringAttr.Key)
				if index != -1 {
					record[index] = strings.TrimSpace(stringAttr.Value)
				}
			}
			for _, dateAttr := range event.DateAttributes {
				index := indexOf(keys, dateAttr.Key)
				if index != -1 {
					record[index] = strings.TrimSpace(dateAttr.Value)
				}
			}
			record[len(keys)] = strings.TrimSpace(trace.StringAttributes[0].Value)
			if err := writer.Write(record); err != nil {
				log.Println("Error writing CSV record:", err)
				return errs
			}
		}
	}
	return nil
}

// indexOf finds the index of a string in a slice
func indexOf(slice []string, str string) int {
	for i, s := range slice {
		if s == str {
			return i
		}
	}
	return -1
}

// GetFileColumns gets the column names of a file.
func GetFileColumns(file io.Reader) ([]string, string, error) {
	// create CSV Reader
	reader := csv.NewReader(file)
	// set LazyQuotes to true to handle quotes in non-quoted fields
	reader.LazyQuotes = true
	// read the first line of the CSV file
	columns, errs := reader.Read()
	if errs != nil {
		return nil, "", errs
	}
	for i, col := range columns {
		columns[i] = strings.TrimLeft(col, "\uFEFF ") // remove BOM and spaces
	}
	delimiter := string(reader.Comma)
	return columns, delimiter, nil
}
