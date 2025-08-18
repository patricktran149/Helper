package excel

import (
	"errors"
	"fmt"
	helper "github.com/patricktran149/Helper"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson"
)

// ConvertFlatJSONToExcel converts either a JSON flat object or array of flat objects into Excel
// and forces all cells to be stored as text
func ConvertFlatJSONToExcel(jsonByte []byte, filePath string) error {
	// Try to unmarshal into an array first
	var arr = make([]bson.D, 0)
	if err := bson.UnmarshalExtJSON(jsonByte, true, &arr); err == nil {
		if len(arr) > 0 {
			return writeToExcel(arr, filePath)
		}
	}

	//Try single object
	var obj = make(bson.D, 0)
	if err := bson.UnmarshalExtJSON(jsonByte, true, &obj); err == nil {
		return writeToExcel([]bson.D{obj}, filePath)
	} else {
		return errors.New("Parse JSON ERROR - " + err.Error())
	}
}

// Helper: write array of maps into Excel
func writeToExcel(data []bson.D, filePath string) error {
	// Collect all keys
	seen := make(map[string]struct{})
	var keys []string
	for _, doc := range data {
		for _, elem := range doc {
			if _, ok := seen[elem.Key]; !ok {
				seen[elem.Key] = struct{}{}
				keys = append(keys, elem.Key)
			}
		}
	}

	// Create Excel file
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)

	// Define text style (@ means text format in Excel)
	textStyle, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		return err
	}

	// Header row
	for col, key := range keys {
		colLetter, _ := excelize.ColumnNumberToName(col + 1)
		cell := fmt.Sprintf("%s1", colLetter)
		f.SetCellValue(sheet, cell, key)
		f.SetCellStyle(sheet, cell, cell, textStyle)
	}

	// Data rows
	for row, obj := range data {
		for col, key := range keys {
			var val interface{}

			for _, field := range obj {
				if field.Key == key {
					val = field.Value
				}
			}

			colLetter, _ := excelize.ColumnNumberToName(col + 1)
			cell := fmt.Sprintf("%s%d", colLetter, row+2)
			f.SetCellStyle(sheet, cell, cell, textStyle)

			if val == nil {
				continue
			}

			var strVal string

			switch v := val.(type) {
			case string:
				strVal = v // keep as is, no quotes
			default:
				strVal = helper.JSONToString(v) // numbers, bool, etc.
			}

			f.SetCellStr(sheet, cell, fmt.Sprintf("%v", strVal)) // force as string
		}
	}

	// Save file
	if err = f.SaveAs(filePath); err != nil {
		return errors.New("Excel Save As ERROR - " + err.Error())
	}

	return nil
}
