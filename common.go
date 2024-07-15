package helper

import (
	"bytes"
	"crypto/md5"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/maja42/goval"
	"github.com/mbordner/kazaam"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ConvertBitSql(request *http.Request, param string) (interface{}, error) {
	str := request.URL.Query().Get(param)
	if str != "" {
		b, err := strconv.ParseBool(str)
		if err != nil {
			return "", errors.New(fmt.Sprintf("Invalid [%s] parameter, Should be 0 or 1 - ERROR - %s ", param, err.Error()))
		}

		if b {
			return 1, nil
		} else {
			return 0, nil
		}
	} else {
		return nil, nil
	}
}

func IsItemExistsInArray(item, slice interface{}) bool {
	s := reflect.ValueOf(slice)

	if s.Kind() != reflect.Slice {
		return false //("Invalid data-type")
	}

	for i := 0; i < s.Len(); i++ {
		if s.Index(i).Interface() == item {
			return true
		}
	}

	return false
}

func XMLIndents(indent string, times int) string {
	strings.Repeat(indent, times)

	return ""
}

func PreprocessXMLData(xmlData []byte) []byte {
	// Replace &T with &amp;T
	xmlData = bytes.ReplaceAll(xmlData, []byte("&T"), []byte("&amp;T"))
	// Replace other entities as needed
	// ...

	return xmlData
}

// CompareDateOnly func
func CompareDateOnly(time1 time.Time, time2 time.Time) bool {
	if time1.Truncate(24 * time.Hour).Equal(time2.Truncate(24 * time.Hour)) {
		return true
	}
	return false
}

func InterfaceToSqlString(inter interface{}) string {
	b, _ := json.Marshal(&inter)

	return strings.ReplaceAll(string(b), "'", "''")
}

func JSONToString(j interface{}) string {
	b, _ := json.Marshal(&j)

	return string(b)
}

func ModelToMapStringInterface(j interface{}) (i map[string]interface{}) {
	b, _ := json.Marshal(&j)

	_ = json.Unmarshal(b, &i)

	return i
}

func EncodeMapStringToURLValues(i map[string]string) string {
	q := url.Values{}
	if i != nil && len(i) > 0 {
		for k, v := range i {
			q.Add(k, v)
		}
	}

	return q.Encode()
}

func RandomString(length int) string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[r.Intn(len(chars))])
	}

	return b.String()
}

func XMLToString(j interface{}) string {
	b, _ := xml.Marshal(&j)

	return string(b)
}

func XMLToStringWithPrefixIndent(j interface{}, prefix, indent string) string {
	b, _ := xml.MarshalIndent(&j, prefix, indent)

	return string(b)
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func GetSHA256Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func GetLimitAndPage(r *http.Request) (limit int64, page int64, err error) {
	limit, err = ParseParamStringToIntFromURLQuery(r, "limit", 300, 500)
	if err != nil {
		return 0, 0, err
	}

	page, err = ParseParamStringToIntFromURLQuery(r, "page", 1, 0)
	if err != nil {
		return 0, 0, err
	}

	return limit, page, nil
}

func GetLimitFromParams(params map[string]string, defaultValue, max int64) (limit int64, err error) {
	limit, err = ParseParamStringToIntFromParams(params, "limit", defaultValue, max)
	if err != nil {
		return 0, err
	}

	return
}

func GetFromAndToDateByTimeZone(fromDateParam, toDateParam string, r *http.Request) (fromDate int64, toDate int64, err error) {
	var (
		currentTime = time.Now()
		timeZone    int
	)

	paramInt := r.URL.Query().Get("timezone")
	if paramInt != "" {
		i, err := strconv.ParseInt(paramInt, 10, 64)
		if err != nil {
			return 0, 0, errors.New(fmt.Sprintf("Cannot parse timezone [%v]", paramInt))
		}
		if i < -12 || i > 12 {
			return 0, 0, errors.New(fmt.Sprintf("Invalid timezone [%v]", i))
		}
	}

	loc := time.FixedZone("FIXED", timeZone*3600)
	fromDateDefault := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, loc)
	toDateDefault := fromDateDefault.AddDate(0, 0, 1)

	fromDate, err = ParseParamStringToIntFromURL(r, fromDateParam, 0, currentTime.Unix())
	if err != nil {
		return 0, 0, err
	}

	toDate, err = ParseParamStringToIntFromURL(r, toDateParam, 0, toDateDefault.Unix())
	if err != nil {
		return 0, 0, err
	}

	if fromDate > toDate {
		return 0, 0, errors.New(fmt.Sprintf("ERROR : From Date '%s' CAN NOT greater than To Date '%s'", time.Unix(fromDate, 0), time.Unix(toDate, 0)))
	}

	return fromDate, toDate, nil
}

func GetFromAndToDateUnixByParams(fromDateParam, toDateParam string, params map[string]string) (fromDate int64, toDate int64, err error) {
	var currTime = time.Now().UTC()

	fromDateDefault := time.Date(currTime.Year(), currTime.Month(), currTime.Day(), 0, 0, 0, 0, time.UTC)
	toDateDefault := fromDateDefault.AddDate(0, 0, 1)

	fromDate, err = ParseParamStringToIntFromMap(params, fromDateParam, 0, currTime.Unix())
	if err != nil {
		return 0, 0, err
	}

	toDate, err = ParseParamStringToIntFromMap(params, toDateParam, toDateDefault.Unix(), 0)
	if err != nil {
		return 0, 0, err
	}

	if fromDate > toDate {
		return 0, 0, errors.New(fmt.Sprintf("ERROR : From Date '%s' CAN NOT greater than To Date '%s'", time.Unix(fromDate, 0), time.Unix(toDate, 0)))
	}

	return fromDate, toDate, nil
}

func GetFromAndToDateWithTimeZoneByParams(params map[string]string, fromDateParam, toDateParam string) (fromDate int64, toDate int64, err error) {
	var (
		currentTime = time.Now()
		timeZone    int
	)

	paramInt := params["timeZone"]
	if paramInt != "" {
		i, err := strconv.ParseInt(paramInt, 10, 64)
		if err != nil {
			return 0, 0, errors.New(fmt.Sprintf("Cannot parse timezone [%v]", paramInt))
		}
		if i < -12 || i > 12 {
			return 0, 0, errors.New(fmt.Sprintf("Invalid timezone [%v]", i))
		}
	}

	loc := time.FixedZone("FIXED", timeZone*3600)
	fromDateDefault := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, loc)
	toDateDefault := fromDateDefault.AddDate(0, 0, 1)

	fromDate, err = ParseParamStringToIntFromParams(params, fromDateParam, 0, currentTime.Unix())
	if err != nil {
		return 0, 0, err
	}

	toDate, err = ParseParamStringToIntFromParams(params, toDateParam, 0, toDateDefault.Unix())
	if err != nil {
		return 0, 0, err
	}

	if fromDate > toDate && fromDate != 0 && toDate != 0 {
		return 0, 0, errors.New(fmt.Sprintf("ERROR : From Date '%s' CAN NOT greater than To Date '%s'", time.Unix(fromDate, 0), time.Unix(toDate, 0)))
	}

	return fromDate, toDate, nil
}

func TransformJSON(rawData string, mapping []map[string]interface{}) (dataOut string, err error) {
	k, err := kazaam.NewKazaam(JSONToString(mapping))
	if err != nil {
		return "", errors.New(fmt.Sprintf("New Kazaam ERROR - %s", err.Error()))
	}

	dataOut, err = k.TransformJSONStringToString(rawData)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Transform ERROR - %s", err.Error()))
	}

	return
}

func ValuateCondition(condition string, data map[string]interface{}) (bool, error) {
	result, err := goval.NewEvaluator().Evaluate(condition, data, nil)
	if err != nil {
		return false, errors.New("Evaluate condition ERROR - " + err.Error())
	}

	switch i := result.(type) {
	case bool:
		return i, nil
	default:
		return false, errors.New("The Result is not boolean ")
	}
}

func GetFromAndToDateStr(fromDateParam, toDateParam string, r *http.Request) (fromDate, toDate time.Time, err error) {
	var layout = "2006/01/02"

	from := r.URL.Query().Get(fromDateParam)
	if from == "" {
		from = "1999/01/01"
	}

	to := r.URL.Query().Get(toDateParam)
	if to == "" {
		to = time.Now().AddDate(0, 0, 1).Format("2006/01/02")
	}

	fromDate, err = time.Parse(layout, from)
	if err != nil {
		err = errors.New("Parse From Date ERROR - " + err.Error())
		return
	}

	toDate, err = time.Parse(layout, to)
	if err != nil {
		err = errors.New("Parse To Date ERROR - " + err.Error())
		return
	}

	return
}

func FloatToThousandCommaString(numberOfDec int, f float64) string {
	format := fmt.Sprintf("%%.%df", numberOfDec)
	return message.NewPrinter(language.English).Sprintf(format, f)
}

func ParseTime(date, layout string) (t time.Time, err error) {
	date = strings.ReplaceAll(date, "Z", "")
	elements := strings.Split(date, ".")
	if len(elements) > 1 {
		layout += fmt.Sprintf(".%s", strings.Repeat("0", len(elements[1])))
	}

	t, err = time.Parse(layout, date)
	if err != nil {
		return
	}

	return
}

func ParseParamStringToIntFromURLQuery(r *http.Request, param string, defaultValue int64, max int64) (int64, error) {
	paramInt := r.URL.Query().Get(param)
	if paramInt == "" {
		return defaultValue, nil
	} else {
		i, err := strconv.ParseInt(paramInt, 10, 64)
		if err != nil {
			return 0, err
		}
		if max != 0 && i > max {
			i = max
		}
		return i, nil
	}
}

func ParseParamStringToIntFromParams(params map[string]string, param string, defaultValue int64, max int64) (int64, error) {
	paramInt := params[param]
	if paramInt == "" {
		return defaultValue, nil
	} else {
		i, err := strconv.ParseInt(paramInt, 10, 64)
		if err != nil {
			return 0, err
		}
		if max != 0 && i > max {
			i = max
		}
		return i, nil
	}
}

func ParseParamStringToIntFromURL(r *http.Request, param string, defaultValue int64, max int64) (int64, error) {
	paramInt := r.URL.Query().Get(param)
	if paramInt == "" {
		return defaultValue, nil
	} else {
		i, err := strconv.ParseInt(paramInt, 10, 64)
		if err != nil {
			return 0, err
		}
		if max != 0 && i > max {
			i = max
		}
		return i, nil
	}
}

func ParseParamStringToIntFromMap(mapping map[string]string, param string, defaultValue int64, max int64) (int64, error) {
	paramInt := mapping[param]
	if paramInt == "" {
		return defaultValue, nil
	} else {
		i, err := strconv.ParseInt(paramInt, 10, 64)
		if err != nil {
			return 0, err
		}
		if max != 0 && i > max {
			i = max
		}
		return i, nil
	}
}

func RecoverError(response http.ResponseWriter, funcName string) {
	if r := recover(); r != nil {
		responseData := bson.M{
			"status":   500,
			"msg":      r,
			"api_name": funcName,
		}
		response.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(response).Encode(responseData)
		return
	}
}

func BearerAuth(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	prefix := "Bearer "
	token := ""

	if auth != "" && strings.HasPrefix(auth, prefix) {
		token = auth[len(prefix):]
	} else {
		token = r.FormValue("token")
	}

	return token, token != ""
}

func RegexStringID(str string) (b bson.M) {
	return bson.M{"$regex": primitive.Regex{Pattern: "^" + EscapeToRegex(str) + "$", Options: "i"}}
}

func RegexStringLike(str string) (b bson.M) {
	return bson.M{"$regex": primitive.Regex{Pattern: EscapeToRegex(str), Options: "i"}}
}

func RegexStringLikeAnyCase(str string) bson.M {
	return bson.M{"$regex": primitive.Regex{Pattern: fmt.Sprintf("^%s$", EscapeToRegex(str)), Options: "i"}}
}

func IsZeroInterface(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

func CompareRequestFields(requestModel, model interface{}) bson.M {
	requestVal := reflect.ValueOf(requestModel)
	val := reflect.ValueOf(model)

	returnBson := make(bson.M, 0)

	if requestVal.Kind() == reflect.Map {
		for x := 0; x < val.Type().NumField(); x++ {
			compareModelFieldName := strings.Split(val.Type().Field(x).Tag.Get("bson"), ",")[0]
			compareModelFieldDataType := val.Type().Field(x).Type.Kind().String()

			for _, e := range reflect.ValueOf(requestModel).MapKeys() {
				v := reflect.ValueOf(requestModel).MapIndex(e)
				requestFieldName := e.Interface()
				compareModelFieldDataKind := val.Type().Field(x).Type.Kind().String()

				if requestFieldName == compareModelFieldName {
					switch t := v.Interface().(type) {
					case string:
						if compareModelFieldDataType == "string" || compareModelFieldDataType == "primitive.ObjectID" {
							returnBson[e.String()] = t
						}
					case bool:
						if compareModelFieldDataType == "bool" {
							returnBson[e.String()] = t
						}
					case float64:
						if IsItemExistsInArray(compareModelFieldDataType, []string{"float32", "float64", "int", "int32", "int64"}) {
							returnBson[e.String()] = t
						}
					case map[string]interface{}, bson.M:
						if IsItemExistsInArray(compareModelFieldDataKind, []string{"map", "struct"}) {
							returnBson[e.String()] = t
						}
					case []map[string]interface{}, []interface{}, []bson.M:
						if compareModelFieldDataKind == "slice" {
							returnBson[e.String()] = t
						}
					//case interface{}:
					//case []interface{}:
					case nil:
					default:
					}
				}
			}

		}
	}

	return returnBson
}

func IsEmptyString(str, replaceStr string) string {
	if str == "" {
		return replaceStr
	}

	return str
}

func AddFieldToMap(i interface{}, additionFields map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(&i)
	if err != nil {
		return nil, err
	}

	returnMap := make(map[string]interface{}, 0)

	if err = json.Unmarshal(b, &returnMap); err != nil {
		return nil, err
	}

	for key, value := range additionFields {
		if _, ok := returnMap[key]; ok {
			return nil, errors.New(fmt.Sprintf("Field [%s] exists in map", key))
		} else {
			returnMap[key] = value
		}
	}

	return returnMap, nil
}

func StringToCompareString(str string) string {
	return strings.ToUpper(strings.TrimSpace(str))
}

func StringCompare(str1, str2 string) bool {
	str1 = strings.ReplaceAll(strings.ToUpper(str1), " ", "")
	str2 = strings.ReplaceAll(strings.ToUpper(str2), " ", "")
	return str1 == str2
}

func IsStringInArray(str string, arr []string) bool {
	//str = strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(str)), " ", "")
	for _, str2 := range arr {
		//str2 = strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(str2)), " ", "")
		if StringCompare(str, str2) {
			return true
		}
	}
	return false
}

func IsStringInArrayAndGet(str string, arr []string) (string, bool) {
	str = strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(str)), " ", "")
	for _, str1 := range arr {
		str2 := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(str1)), " ", "")
		if str == str2 {
			return str1, true
		}
	}
	return "", false
}

func RoundNumberDecimalN(number float64, n int) float64 {
	return math.Round(number*math.Pow10(n)) / math.Pow10(n)
}

func InOutDateStr(date, in, out string) string {
	d, _ := time.Parse(in, date)
	return d.Format(out)
}

func IsZeroFloat(f, replaceF float64) float64 {
	if f == 0.0 {
		return replaceF
	}
	return f
}

func ReadCsvFile(csvfile *os.File, comma rune) *csv.Reader {
	r := csv.NewReader(csvfile)
	r.LazyQuotes = true
	r.Comma = comma
	r.Comment = '#'
	return r
}

// IsArrayIndexExists Check if array[n] has n + 1 length.
// Example : array[2] has 2 + 1 = 3 length
func IsArrayIndexExists(array interface{}, index int) bool {
	s := reflect.ValueOf(array)

	if s.Kind() != reflect.Slice && s.Kind() != reflect.Array {
		return false
	}

	return s.Len() > index
}

func BasicAuth(userName, password string) string {
	auth := userName + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func WriteJSONToCSV(fileName, folder string, arr interface{}) (string, error) {
	var (
		values = make([]reflect.Value, 0)
		row    []string
	)

	folder = fmt.Sprintf("%s/%s", folder, time.Now().Format("20060102"))

	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return "", errors.New("OS Make Directory ERROR - " + err.Error())
	}

	fileName = fmt.Sprintf("%s_%s.csv", fileName, time.Now().Format("20060102150405"))
	filePath := fmt.Sprintf("%s/%s", folder, fileName)

	csvFile, err := os.Create(filePath)
	if err != nil {
		return "", errors.New("OS Create CSV file ERROR - " + err.Error())
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)

	kind := reflect.TypeOf(arr).Kind().String()
	s := reflect.ValueOf(arr)
	switch kind {
	case "slice":
		{
			if s.Len() == 0 {
				return "", errors.New(fmt.Sprintf("Slice with no element"))
			}

			if s.Index(0).NumField() == 0 {
				return "", errors.New(fmt.Sprintf("Slice with Empty struct field"))
			}

			for i := 0; i < s.Len(); i++ {
				if s.Index(i).Type().Kind().String() == "struct" {
					values = append(values, s.Index(i))
				} else {
					return "", errors.New(fmt.Sprintf("Slice's Element [%v] is not a struct ", i+1))
				}
			}
		}
	case "struct":
		{
			values = append(values, s)
		}
	default:
		return "", errors.New(fmt.Sprintf("Invalid JSON Type - %s", kind))
	}

	//get Headers
	for y := 0; y < values[0].NumField(); y++ {
		row = append(row, values[0].Type().Field(y).Tag.Get("json"))
	}

	if err = writer.Write(row); err != nil {
		return "", errors.New("Write header to CSV ERROR - " + err.Error())
	}

	//Write lines
	for i, val := range values {
		row = []string{}

		for y := 0; y < val.NumField(); y++ {
			row = append(row, fmt.Sprintf("%v", val.Field(y)))
		}

		if err = writer.Write(row); err != nil {
			return "", errors.New(fmt.Sprintf("Write Line [%v] to CSV ERROR - %s", i, err.Error()))
		}
	}

	// remember to flush!
	writer.Flush()

	return filePath, nil
}

func GenerateOTP(n int) string {
	v := ""

	for i := 0; i < n; i++ {
		result, _ := cryptoRand.Int(cryptoRand.Reader, big.NewInt(10))
		v += fmt.Sprintf("%d", result.Int64())
	}

	return v
}

func DistinctStrings(arr []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, str := range arr {
		if _, ok := seen[str]; !ok {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}

func AddStringIfNotExist(arr []string, str string) []string {
	for _, s := range arr {
		if s == str {
			return arr // String already exists, return original array
		}
	}
	return append(arr, str) // String doesn't exist, append it to the array
}

func IsDriverRegistered(driverName string) bool {
	drivers := sql.Drivers()
	for _, driver := range drivers {
		if driver == driverName {
			return true
		}
	}
	return false
}

func EscapeToRegex(input string) string {
	return regexp.QuoteMeta(input)
}

func RemoveStringItem(slice []string, value string) []string {
	// Find the index of the element with the matching value
	index := -1
	for i, v := range slice {
		if v == value {
			index = i
			break
		}
	}

	// If the value was not found, return the original slice unchanged
	if index == -1 {
		return slice
	}

	// Create a new slice that excludes the element with the matching value
	return append(slice[:index], slice[index+1:]...)
}

func RemoveDuplicate[T string | float64](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func ReplaceIgnoreCase(input, old, new string) string {
	// Find all occurrences of the old substring in the input string
	index := strings.Index(strings.ToLower(input), strings.ToLower(old))

	// If the substring is found, replace it
	if index != -1 {
		// Extract the substring before the match
		before := input[:index]

		// Extract the substring after the match
		after := input[index+len(old):]

		// Replace the old substring with the new one
		replaced := before + new + after

		// Recursively call the function in case there are more occurrences
		return ReplaceIgnoreCase(replaced, old, new)
	}

	// If no more occurrences are found, return the modified string
	return input
}
