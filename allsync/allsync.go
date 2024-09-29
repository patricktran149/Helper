package allsync

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	_ "github.com/SAP/go-hdb/driver"
	xj "github.com/basgys/goxml2json"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/jinzhu/copier"
	allSyncModel "github.com/patricktran149/AllSync.Model"
	helper "github.com/patricktran149/Helper"
	_ "github.com/sijms/go-ora/v2"
	"go.elastic.co/apm/module/apmsql"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/pkcs12"
	"hash"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func RequestAllSync(asConfig allSyncModel.AllSyncConfig, path, method string, data interface{}, params map[string]interface{}) (status int, msg string, responseData []byte) {
	var (
		b   []byte
		err error
	)

	status = 500

	if data != nil {
		b, err = json.Marshal(data)
		if err != nil {
			msg = err.Error()
			return
		}
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", asConfig.SystemAPIURL, path), bytes.NewBuffer(b))
	if err != nil {
		msg = err.Error()
		return
	}

	// add query params
	query := url.Values{}
	if params != nil && len(params) > 0 {
		for k, v := range params {
			query.Add(k, fmt.Sprintf("%v", v))
		}
		req.URL.RawQuery = query.Encode()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", asConfig.Token))
	req.Header.Set("tenantID", asConfig.TenantID)
	req.Header.Set("Accept-Encoding", "gzip,deflate,br")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		msg = err.Error()
		return
	}
	if resp != nil {
		//if resp.Header.Get("Content-Encoding") == "gzip" {
		var reader *gzip.Reader
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			msg = fmt.Sprintf("Gzip Read ERROR - %s", err.Error())
			return
		}
		defer reader.Close()

		responseData, err = io.ReadAll(reader)
		if err != nil {
			msg = fmt.Sprintf("Read Uncompress data ERROR - %s", err.Error())
			return
		}

		var allSyncResp allSyncModel.ToAppResponse
		if err = json.Unmarshal(responseData, &allSyncResp); err != nil {
			msg = fmt.Sprintf("Json Unmarshal AllSync Response format ERROR - %s", err.Error())
			return
		}
		msg = allSyncResp.Message
		//} else {
		//	responseData, _ = io.ReadAll(resp.Body)
		//	var allSyncResp allSyncModel.ToAppResponse
		//	_ = json.Unmarshal(responseData, &allSyncResp)
		//	msg = allSyncResp.Message
		//}

		defer resp.Body.Close()
	}

	status = resp.StatusCode

	return
}

func RequestAllSyncSystem(asConfig allSyncModel.AllSyncConfig, path, method string, data interface{}, params map[string]interface{}) (status int, msg string, responseData []byte) {
	var (
		b   []byte
		err error
	)

	status = 500

	if data != nil {
		b, err = json.Marshal(data)
		if err != nil {
			msg = err.Error()
			return
		}
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", asConfig.SystemAPIURL, path), bytes.NewBuffer(b))
	if err != nil {
		msg = err.Error()
		return
	}

	// add query params
	query := url.Values{}
	if params != nil && len(params) > 0 {
		for k, v := range params {
			query.Add(k, fmt.Sprintf("%v", v))
		}
		req.URL.RawQuery = query.Encode()
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", asConfig.SystemSecretKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip,deflate,br")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		msg = err.Error()
		return
	}
	if resp != nil {
		//if resp.Header.Get("Content-Encoding") == "gzip" {
		var reader *gzip.Reader
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			msg = fmt.Sprintf("Gzip Read ERROR - %s", err.Error())
			return
		}
		defer reader.Close()

		responseData, err = io.ReadAll(reader)
		if err != nil {
			msg = fmt.Sprintf("Read Uncompress data ERROR - %s", err.Error())
			return
		}

		var allSyncResp allSyncModel.ToAppResponse
		if err = json.Unmarshal(responseData, &allSyncResp); err != nil {
			msg = fmt.Sprintf("Json Unmarshal AllSync Response format ERROR - %s", err.Error())
			return
		}
		msg = allSyncResp.Message
		//} else {
		//	responseData, _ = io.ReadAll(resp.Body)
		//	var allSyncResp allSyncModel.ToAppResponse
		//	_ = json.Unmarshal(responseData, &allSyncResp)
		//	msg = allSyncResp.Message
		//}

		defer resp.Body.Close()
	}

	status = resp.StatusCode

	return
}

func RequestOtherSystemAPIFromAllSyncFlow(asConfig allSyncModel.AllSyncConfig, flowAppID string, flowConfig allSyncModel.IntegrationFlowConfig, toMapData interface{}, postData []byte, timeout time.Duration) (sendData, respData string, qLogs []allSyncModel.QueueLog, err error) {
	var (
		params  = make(map[string]interface{}, 0)
		headers = make(map[string]interface{}, 0)
		mapData = make(map[string]interface{}, 0)
		//funcName = "RequestOtherSystemAPIFromAllSyncFlow"
	)

	if toMapData != nil {
		if reflect.TypeOf(toMapData).Kind() == reflect.Slice {
			m, ok := toMapData.([]interface{})
			if ok {
				mapData = map[string]interface{}{
					"array": m,
				}
			}

		} else if reflect.TypeOf(toMapData).Kind() == reflect.Map {
			m, ok := toMapData.(map[string]interface{})
			if ok {
				mapData = m
			}
		}
	}

	//Get API Method
	apiMethod, err := LiquidMapping(asConfig, flowConfig.API.Method, mapData)
	if err != nil {
		err = errors.New("Liquid Mapping API Method ERROR - " + err.Error())
		return
	}

	//Get API URL
	apiURL, err := LiquidMapping(asConfig, flowConfig.API.URL, mapData)
	if err != nil {
		err = errors.New("Liquid Mapping API URL ERROR - " + err.Error())
		return
	}

	if apiURL == "" {
		err = errors.New("API URL is empty ")
		return
	}

	authType := allSyncModel.AuthenticationType(strings.ToUpper(string(flowConfig.API.AuthenticationType)))
	switch authType {
	case allSyncModel.AuthenticationTypeHmac:
		{
			var headers = make(map[string]interface{}, 0)

			if flowConfig.API.HMAC.AppIDName != "" {
				headers[flowConfig.API.HMAC.AppIDName] = flowConfig.API.HMAC.AppIDValue
			}

			if flowConfig.API.HMAC.AppKeyName != "" {
				headers[flowConfig.API.HMAC.AppIDName] = flowConfig.API.HMAC.AppIDValue
			}

			currTime := time.Now().UTC().Add(8 * time.Hour).Format(HMACTimeStampFormatConvert(flowConfig.API.HMAC.TimestampFormat))
			data := map[string]interface{}{
				"appIDName":      flowConfig.API.HMAC.AppIDName,
				"appIDValue":     flowConfig.API.HMAC.AppIDValue,
				"appKeyName":     flowConfig.API.HMAC.AppKeyName,
				"appKeyValue":    flowConfig.API.HMAC.AppKeyValue,
				"timeStampName":  flowConfig.API.HMAC.TimestampName,
				"timeStampValue": currTime,
			}

			//{{api.hmac.appidname}}={{api.hmac.appidvalue}}&{{api.hmac.timestampname}}={{api.hmac.timestampformat}}&Token=98743f7f68a1fca5wp
			toHash, liquidErr := LiquidMapping(asConfig, flowConfig.API.HMAC.BaseFields, data)
			if liquidErr != nil {
				err = errors.New("Liquid Mapping API HMAC Base fields ERROR - " + err.Error())
				return
			}

			hashed := ComputeSHA(flowConfig.API.HMAC.Algorithm, toHash)

			postData = []byte(fmt.Sprintf(`%s=%s&%s=%s&%s=%s&%s=%s`,
				flowConfig.API.HMAC.AppIDName, flowConfig.API.HMAC.AppIDValue,
				flowConfig.API.HMAC.TimestampName, currTime,
				flowConfig.API.HMAC.SignatureName, hashed,
				flowConfig.API.HMAC.BodyName, string(postData), //`{"Channel": "Shopify","OrderNo": "9847","OrderId": "9847","OrderDate": "2022-04-27T23:36:07","OrderNotes": "","TotalNoOfLine": 1,"TotalOrderValue": 26.76,"Currency": "SGD","CurrencyConvertionRate": 0,"ShippingCharges": 0,"PromoCode": "","DeliveryMethod": "StandardDelivery","Customer": {"FirstName": "ថោងផលាពេជ្រសុម៉ារីតា","LastName": "#9847","Email": "ttgmail.com","MobileNo": "","ReceiverFirstName": "ថោងផលាពេជ្រសុម៉ារីតា","ReceiverLastName": "#9847","ReceiverMobileNo": "","ShippingAddress1": "Rose Condo Acacia Street, Phnom Penh","ShippingAddress2": "","Province": "","City": "Phnom Penh ","Country": "Cambodia","PostalCode": ""},"OrderItems": [{"LineNo": 1,"SkuCode": "1003527401205","RetailPrice": 26.76,"SoldValue": 26.76,"Unit": 1,"TaxRate": 0,"TaxValue": 0}],"Payments": []}`,
			))

		}
	case allSyncModel.AuthenticationTypeBearer:
		{
			if flowConfig.OAuth.URL != "" {

				var (
					oauthMethod  = flowConfig.OAuth.Method
					oauthURL     = flowConfig.OAuth.URL
					oauthBody    string
					oauthParams  = flowConfig.OAuth.Params
					oauthHeaders = flowConfig.OAuth.Headers
					r            string
					accessToken  string
				)

				bodyText, ok := flowConfig.OAuth.Body["bodyText"]
				if ok {
					bod, ok1 := bodyText.(string)
					if ok1 {
						oauthBody = bod
					}
				} else {
					oauthBody = helper.JSONToString(flowConfig.OAuth.Body)
				}

				_, r, err = RequestOtherSystemAPI(oauthMethod, oauthURL, []byte(oauthBody), oauthParams, oauthHeaders, timeout, flowConfig.OAuth.P12)
				if err != nil {
					err = errors.New("Request Access token ERROR - " + err.Error())
					return
				}

				respObject := make(map[string]interface{}, 0)

				if err = json.Unmarshal([]byte(r), &respObject); err != nil {
					err = errors.New("JSON Unmarshal Access token ERROR - " + err.Error())
					return
				}

				accessToken, err = LiquidMapping(asConfig, flowConfig.OAuth.Template, respObject)
				if err != nil {
					err = errors.New("Liquid Mapping API URL ERROR - " + err.Error())
					return
				}

				if flowConfig.API.Token == "" {
					headers["Authorization"] = fmt.Sprintf("Bearer %s", strings.TrimSpace(accessToken))
				} else {
					headers[flowConfig.API.Token] = fmt.Sprintf("%s", strings.TrimSpace(accessToken))
				}

			} else {
				headers["Authorization"] = fmt.Sprintf("Bearer %s", flowConfig.API.Token)
			}

			break
		}
	case allSyncModel.AuthenticationTypeBasic:
		{
			auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", flowConfig.API.BasicAuth.UserName, flowConfig.API.BasicAuth.Password)))
			headers["Authorization"] = fmt.Sprintf("Basic %s", auth)
			break
		}
	case allSyncModel.AuthenticationTypeOAuth1:
		{
			oauth1 := flowConfig.API.OAuth1

			if oauth1.Nonce == "" || (oauth1.Nonce != "" && len(oauth1.Nonce) != 11) {
				oauth1.Nonce = helper.RandomString(11)
			}

			timeStamp := strconv.FormatInt(time.Now().Unix(), 10)

			baseStrParts := url.Values{
				"oauth_consumer_key":     {oauth1.ConsumerKey},
				"oauth_nonce":            {oauth1.Nonce},
				"oauth_signature_method": {oauth1.SignatureMethod},
				"oauth_timestamp":        {timeStamp},
				"oauth_token":            {oauth1.AccessToken},
				"oauth_version":          {"1.0"},
			}

			for key, value := range flowConfig.API.Params {
				baseStrParts.Set(key, fmt.Sprintf("%v", value))
			}

			baseStr := fmt.Sprintf("%s&%s&%s", flowConfig.API.Method, url.QueryEscape(flowConfig.API.URL), url.QueryEscape(baseStrParts.Encode()))

			hmacHashAlg := GetSHAHash(allSyncModel.HMACAlgorithm(oauth1.SignatureMethod))

			signingKey := fmt.Sprintf("%s&%s", url.QueryEscape(oauth1.ConsumerSecret), url.QueryEscape(oauth1.TokenSecret))
			h := hmac.New(hmacHashAlg, []byte(signingKey))
			h.Write([]byte(baseStr))
			signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

			switch flowConfig.API.OAuth1.AddData {
			case "HEADER":
				{
					authHeader := fmt.Sprintf(
						`OAuth realm="%s",oauth_consumer_key="%s",oauth_token="%s",oauth_signature_method="%s",oauth_timestamp="%s",oauth_nonce="%s",oauth_version="1.0",oauth_signature="%s"`,
						oauth1.Realm,
						oauth1.ConsumerKey,
						oauth1.AccessToken,
						oauth1.SignatureMethod,
						timeStamp,
						oauth1.Nonce,
						url.QueryEscape(signature),
					)
					headers["Authorization"] = authHeader

					break
				}
			case "PARAM":
				{
					params["oauth_consumer_key"] = oauth1.ConsumerKey
					params["oauth_nonce"] = oauth1.Nonce
					params["oauth_signature"] = url.QueryEscape(signature)
					params["oauth_signature_method"] = oauth1.SignatureMethod
					params["oauth_timestamp"] = timeStamp
					params["oauth_token"] = oauth1.AccessToken
					params["oauth_version"] = "1.0"

					break
				}
			}
		}
	default:
		break
	}

	var app allSyncModel.Application
	//Get Application from flowAppID
	app, err = GetApplicationByID(asConfig, flowAppID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
		return
	}

	//Get last run config
	lastRun := app.GetConfigValue("last_run")

	for k, v := range flowConfig.API.Params {
		vStr, _ := v.(string)

		var liquidMappedValue string

		if vStr != "" {
			if strings.Contains(vStr, "lastrun") {
				//Assign temp value
				var currentTimeValue string

				currentTimeValue, err = LiquidMapping(asConfig, strings.ReplaceAll(vStr, "lastrun", "now"), nil)
				if err != nil {
					err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
					return
				}

				//If application don't have last_run config yet, set it to current time
				if lastRun != "" {
					vStr = lastRun
				} else {
					vStr = currentTimeValue
				}

				liquidMappedValue, err = LiquidMapping(asConfig, vStr, nil)
				if err != nil {
					err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
					return
				}

				//Update last_run config = currentTime unix
				app.UpdateConfigValue("GENERAL", "last_run", currentTimeValue)

				//Update Application
				if _, err = UpdateApplication(asConfig, app); err != nil {
					err = errors.New(fmt.Sprintf("Update Application [%s] ERROR - %s", app.ApplicationID, err.Error()))
					return
				}

				params[k] = liquidMappedValue
				continue

			} else if strings.Contains(vStr, "now") {
				liquidMappedValue, err = LiquidMapping(asConfig, vStr, nil)
				if err != nil {
					err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
					return
				}

				//Update last_run config = currentTime unix
				app.UpdateConfigValue("GENERAL", "last_run", liquidMappedValue)

				//Update Application
				if _, err = UpdateApplication(asConfig, app); err != nil {
					err = errors.New(fmt.Sprintf("Update Application [%s] ERROR - %s", app.ApplicationID, err.Error()))
					return
				}

				params[k] = liquidMappedValue
				continue
			}
		}

		liquidMappedValue, err = LiquidMapping(asConfig, vStr, mapData)
		if err != nil {
			err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
			return
		}

		params[k] = liquidMappedValue
	}

	for k, v := range flowConfig.API.Headers {
		vStr, _ := v.(string)
		var liquidMappedValue string
		liquidMappedValue, err = LiquidMapping(asConfig, vStr, mapData)
		if err != nil {
			err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
			return
		}

		headers[k] = liquidMappedValue
	}

	if len(flowConfig.API.Body) > 0 && string(postData) == "" {
		body := helper.JSONToString(flowConfig.API.Body)
		var liquidMappedValue string
		if strings.Contains(body, "lastrun") {
			//Assign temp value
			var (
				currentTimeValue string
				lastRunPart      string
			)

			re := regexp.MustCompile(`{{ 'lastrun[^}]* }}`)
			match := re.FindString(body)
			if match != "" {
				// Extract the value including "{{ 'lastrun" and "}}"
				lastRunPart = match[0:len(match)]
			}

			currentTimeValue, err = LiquidMapping(asConfig, strings.ReplaceAll(lastRunPart, "lastrun", "now"), nil)
			if err != nil {
				err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
				return
			}

			//If application don't have last_run config yet, set it to current time
			if lastRun != "" {
				body = strings.ReplaceAll(body, lastRunPart, lastRun)
			} else {
				body = strings.ReplaceAll(body, lastRunPart, currentTimeValue)
			}

			//Update last_run config = currentTime unix
			app.UpdateConfigValue("GENERAL", "last_run", currentTimeValue)

			//Update Application
			if _, err = UpdateApplication(asConfig, app); err != nil {
				err = errors.New(fmt.Sprintf("Update Application [%s] ERROR - %s", app.ApplicationID, err.Error()))
				return
			}

		}

		liquidMappedValue, err = LiquidMapping(asConfig, body, nil)
		if err != nil {
			err = errors.New(fmt.Sprintf("Get Application [%s] ERROR - %s", flowAppID, err.Error()))
			return
		}

		postData = []byte(liquidMappedValue)

	}

	qLogs = LogsAddLog(qLogs, "API Method", apiMethod, "", "")
	qLogs = LogsAddLog(qLogs, "API URL", apiURL, "", "")
	qLogs = LogsAddLog(qLogs, "Params", helper.JSONToString(params), "", "")
	qLogs = LogsAddLog(qLogs, "Headers", helper.JSONToString(headers), "", "")
	qLogs = LogsAddLog(qLogs, "Template", helper.JSONToString(toMapData), "", "")

	sendData, respData, err = RequestOtherSystemAPI(apiMethod, apiURL, postData, params, headers, timeout, flowConfig.API.P12)
	if err != nil {
		err = errors.New("Request Other System API ERROR - " + err.Error())
		return
	}

	return
}

func RequestOtherSystemAPI(method, apiUrl string, data []byte, params, headers map[string]interface{}, timeout time.Duration, p12 allSyncModel.P12) (sendData, respData string, err error) {
	var (
		client = &http.Client{}
		ctx    = context.Background()
	)

	if timeout > 0 {
		ctxTmp, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		ctx = ctxTmp
	}

	sendData = string(data)

	req, err := http.NewRequestWithContext(ctx, method, apiUrl, strings.NewReader(sendData))
	if err != nil {
		err = errors.New("Make request ERROR - " + err.Error())
		return
	}

	if len(params) > 0 {
		query := url.Values{}
		for k, v := range params {
			query.Add(k, fmt.Sprintf("%v", v))
		}

		req.URL.RawQuery = query.Encode()
	}

	for k, v := range headers {
		req.Header.Set(k, fmt.Sprintf("%v", v))
	}

	if p12.FilePath != "" {
		var tlss *tls.Config

		tlss, err = IncludeP12ToTLSConfig(p12)
		if err != nil {
			err = errors.New("Include P12 to TLS Config ERROR - " + err.Error())
			return
		}

		client.Transport = &http.Transport{
			TLSClientConfig: tlss,
		}
	}

	res, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = errors.New("Send Request ERROR - Timed out ")
			return
		}

		err = errors.New("Send Request ERROR - " + err.Error())
		return
	}

	if res != nil {
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)

		if len(string(body)) > 5*1024*1024 { // 5MB
			err = errors.New("Response length greater than 5MB ")
			return
		}

		respData = string(body)

		if res.StatusCode < 200 || res.StatusCode > 299 {
			err = errors.New(fmt.Sprintf("Request ERROR - Status [%v] - Code [%v] - Response [%s]", res.Status, res.StatusCode, string(body)))
			return
		}
	}

	return
}

func ErrorResponse(response http.ResponseWriter, statusCode, errorCode int, errMsg string) {
	response.WriteHeader(statusCode)
	json.NewEncoder(response).Encode(allSyncModel.ResponseResult{
		Success:   false,
		Message:   errMsg,
		ErrorCode: errorCode,
		Data:      nil,
	})
}

func SuccessResponse(response http.ResponseWriter, data interface{}, msg string) {

	json.NewEncoder(response).Encode(allSyncModel.ResponseResult{
		Success:   true,
		Message:   msg,
		ErrorCode: 0,
		Data:      data,
	})
}

func MapMongoWriteException(err error) (str string) {
	if we, ok := err.(mongo.WriteException); ok {
		for _, e := range we.WriteErrors {
			str = strings.Trim(str+fmt.Sprintf(";%s", allSyncModel.MapErrorType[e.Code]), ";")
			fmt.Println(e.Details)
		}
	}

	return str
}

func GetApplicationList(asConfig allSyncModel.AllSyncConfig) (appList []allSyncModel.Application, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, "Application", http.MethodGet, nil, nil)
	if status != 200 {
		return appList, errors.New("Get All Application ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return appList, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.ApplicationList, nil
}

func GetTenantList(asConfig allSyncModel.AllSyncConfig) (tenantList []allSyncModel.Tenant, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, "System/Tenant", http.MethodGet, nil, nil)
	if status != 200 {
		return tenantList, errors.New("Get All Tenant ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return tenantList, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.TenantList, nil
}

func GetTenantByID(asConfig allSyncModel.AllSyncConfig, tenantID string) (tenant allSyncModel.Tenant, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("System/Tenant/%s", tenantID), http.MethodGet, nil, nil)
	if status != 200 {
		return tenant, errors.New("Get Tenant ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return tenant, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.Tenant, nil
}

func GetApplicationByID(asConfig allSyncModel.AllSyncConfig, applicationID string) (app allSyncModel.Application, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("Application/%s", applicationID), http.MethodGet, nil, nil)
	if status != 200 {
		return app, errors.New("Get All Application ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return app, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.Application, nil
}

func UpdateApplication(asConfig allSyncModel.AllSyncConfig, app allSyncModel.Application) (appResp allSyncModel.Application, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("Application/%s", app.ApplicationID), http.MethodPut, app, nil)
	if status != 200 {
		return app, errors.New("Get All Application ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return app, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.Application, nil
}

func GetQueueIncomingList(asConfig allSyncModel.AllSyncConfig, applicationID string, params map[string]interface{}) (queues []allSyncModel.Queue, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("QueueIncoming/%s", applicationID), http.MethodGet, nil, params)
	if status != 200 {
		return queues, errors.New("Get Incoming Queue list ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return queues, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.QueueIncomingList, nil
}

func GetQueueOutgoingList(asConfig allSyncModel.AllSyncConfig, applicationID string, params map[string]interface{}) (queues []allSyncModel.Queue, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("QueueOutgoing/%s", applicationID), http.MethodGet, nil, params)
	if status != 200 {
		return queues, errors.New("Get Outgoing Queue list ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return queues, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.QueueOutgoingList, nil
}

func GetDataMapperList(asConfig allSyncModel.AllSyncConfig, params map[string]interface{}) (dataMappers []allSyncModel.DataMapper, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, "DataMapper", http.MethodGet, nil, params)
	if status != 200 {
		return dataMappers, errors.New("Get Data Mapper List ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return dataMappers, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.DataMapperList, nil
}

func GetDataMapperDirection(asConfig allSyncModel.AllSyncConfig, params map[string]interface{}) (dataMapper allSyncModel.DataMapper, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, "DataMapper", http.MethodGet, nil, params)
	if status != 200 {
		return dataMapper, errors.New("Get Data Mapper List ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return dataMapper, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	if len(allSyncResp.Data.DataMapperList) > 0 {
		dataMapper = allSyncResp.Data.DataMapperList[0]
	}

	return
}

func LiquidMapping(asConfig allSyncModel.AllSyncConfig, template string, data map[string]interface{}) (dataOut string, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	liquidMapping := allSyncModel.LiquidObjectOverView{
		Data:     data,
		Template: template,
	}

	status, msg, respData := RequestAllSync(asConfig, "Object/LiquidMapping", http.MethodPost, liquidMapping, nil)
	if status != 200 {
		return dataOut, errors.New("Liquid mapping ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return dataOut, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.DataOut, nil
}

func PostObject(asConfig allSyncModel.AllSyncConfig, appID, tableName string, payload interface{}) (respData []byte, err error) {
	var payloadMap = make(map[string]interface{})

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("Marshal Payload ERROR - " + err.Error())
	}

	if err = json.Unmarshal(b, &payloadMap); err != nil {
		return nil, errors.New("Unmarshal Payload ERROR - " + err.Error())
	}

	objectReq := allSyncModel.ObjectRequest{
		ApplicationID: appID,
		TableName:     tableName,
		Fields:        payloadMap,
	}

	status, msg, respData := RequestAllSync(asConfig, "Object", http.MethodPost, objectReq, nil)
	if status != 200 {
		return respData, errors.New("Post Object ERROR - " + msg)
	}

	return
}

func UpdateQueueStatus(asConfig allSyncModel.AllSyncConfig, queueType string, queue allSyncModel.Queue, sendData, respData, errMsg string, queueFlag bool, status allSyncModel.QueueStatus, maxRetryDays, maxRetryTimes int) error {
	var queueReq allSyncModel.QueueRequest

	retryTimes := queue.RetryTimes
	retryDays := queue.RetryDays

	if !queueFlag {
		if retryTimes <= maxRetryTimes {
			retryTimes++
		}

		if retryDays == 0 {
			retryDays = 1
		}
	}

	queue.RetryTimes = retryTimes
	queue.RetryDays = retryDays
	queue.Flag = queueFlag
	queue.Status = status
	queue.SendData = sendData
	queue.ResponseData = respData
	queue.Message = errMsg
	queue.UpdatedBy = "Integration API"

	if !queueFlag {
		queue.Message = fmt.Sprintf("%s - Retry times: %d. Retry days: %d", errMsg, retryTimes, retryDays)
	}

	if retryTimes >= maxRetryTimes && retryDays >= maxRetryDays {
		queue.Flag = true
		queue.IsSkip = true
		queue.Message = "SKIP - " + errMsg
	}

	_ = copier.Copy(&queueReq, &queue)

	statusCode, msg, _ := RequestAllSync(asConfig, fmt.Sprintf("Queue%s/%s/%s", queueType, queue.ApplicationID, queue.ID.Hex()), http.MethodPut, queueReq, nil)
	if statusCode != 200 {
		return errors.New(fmt.Sprintf("Update Queue ERROR - %v", msg))
	}

	return nil
}

func CreateQueue(asConfig allSyncModel.AllSyncConfig, queueType string, queueReq allSyncModel.QueueRequest) error {
	statusCode, msg, _ := RequestAllSync(asConfig, fmt.Sprintf("Queue%s", queueType), http.MethodPost, queueReq, nil)
	if statusCode != 200 {
		return errors.New(fmt.Sprintf("Create Queue ERROR - %v", msg))
	}

	return nil
}

func CreateMicroServiceLog(asConfig allSyncModel.AllSyncConfig, serviceID, version, funcName, msg string) error {
	micLog := allSyncModel.MicroServiceLogRequest{
		ServiceID:    serviceID,
		FunctionName: funcName,
		VersionID:    version,
		Message:      msg,
	}

	_, _, _ = RequestAllSync(asConfig, "MicroServiceLog", http.MethodPost, micLog, nil)
	//if statusCode != 200 {
	//	return errors.New(fmt.Sprintf("Create Micro Service log ERROR - %v", msg))
	//}

	return nil
}

func GetIntegrationFlowList(asConfig allSyncModel.AllSyncConfig, params map[string]interface{}) (intFlowList []allSyncModel.IntegrationFlow, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, "IntegrationFlow", http.MethodGet, nil, params)
	if status != 200 {
		return intFlowList, errors.New("Get Integration Flow list ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return intFlowList, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.IntegrationFlowList, nil
}

func GetObjectList(asConfig allSyncModel.AllSyncConfig, tableName string, params map[string]interface{}) (objs []bson.M, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("Object/%s", tableName), http.MethodGet, nil, params)
	if status != 200 {
		return objs, errors.New("Get Object list ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return objs, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.ObjectList, nil
}

func GetObjectByID(asConfig allSyncModel.AllSyncConfig, tableName string, id interface{}) (obj bson.M, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("Object/%s/%v", tableName, id), http.MethodGet, nil, nil)
	if status != 200 {
		return obj, errors.New("Get Object ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return obj, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.Object, nil
}

func GetUDTList(asConfig allSyncModel.AllSyncConfig) (udt []allSyncModel.UserDefinedTable, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, "UserDefinedTable", http.MethodGet, nil, nil)
	if status != 200 {
		return udt, errors.New("Get UDT by Name ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return udt, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.UserDefinedTableList, nil
}

func GetUDTByName(asConfig allSyncModel.AllSyncConfig, tableName string) (udt allSyncModel.UserDefinedTable, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("UserDefinedTable/%s", tableName), http.MethodGet, nil, nil)
	if status != 200 {
		return udt, errors.New("Get UDT by Name ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return udt, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.UserDefinedTable, nil
}

func UDTDataWarehouse(asConfig allSyncModel.AllSyncConfig, tableName string) (err error) {
	status, msg, _ := RequestAllSync(asConfig, fmt.Sprintf("UserDefinedTable/%s/DataWarehouse", tableName), http.MethodPut, nil, nil)
	if status != 200 {
		return errors.New("UDT Update DataWarehouse ERROR - " + msg)
	}

	return
}

func GetIntegrationFlowByID(asConfig allSyncModel.AllSyncConfig, id string) (intFlow allSyncModel.IntegrationFlow, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("IntegrationFlow/%s", id), http.MethodGet, nil, nil)
	if status != 200 {
		return intFlow, errors.New("Get Integration Flow list ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return intFlow, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.IntegrationFlow, nil
}

func GetMicroServiceList(asConfig allSyncModel.AllSyncConfig, filter bson.M) (msList []allSyncModel.MicroServiceResponse, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	statusCode, msg, respData := RequestAllSync(asConfig, "MicroService", http.MethodGet, nil, filter)
	if statusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Get Micro Service ERROR - %v", msg))
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return nil, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.MicroServiceList, nil
}

func GetMicroServiceByID(asConfig allSyncModel.AllSyncConfig, serviceID string) (ms allSyncModel.MicroServiceResponse, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	statusCode, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("MicroService/%s", serviceID), http.MethodGet, nil, nil)
	if statusCode != 200 {
		return ms, errors.New(fmt.Sprintf("Get Micro Service ERROR - %v", msg))
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return ms, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.MicroService, nil
}

func UpdateMicroService(asConfig allSyncModel.AllSyncConfig, request allSyncModel.MicroServiceUpdateRequest) (err error) {
	req := helper.ModelToMapStringInterface(request)

	delete(req, "description")
	delete(req, "fileURL")
	delete(req, "languageCustomCode")
	delete(req, "message")
	delete(req, "customCode")

	statusCode, msg, _ := RequestAllSync(asConfig, fmt.Sprintf("MicroService/%s", request.ServiceID), http.MethodPut, req, nil)
	if statusCode != 200 {
		return errors.New(fmt.Sprintf("Update Micro Service ERROR - %v", msg))
	}

	return
}

func GetMicroServiceStatus(asConfig allSyncModel.AllSyncConfig, filter bson.M) (mssList []allSyncModel.MicroServiceStatusResponse, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	statusCode, msg, respData := RequestAllSync(asConfig, "MicroServiceStatus", http.MethodGet, nil, filter)
	if statusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Get Micro Service Status ERROR - %v", msg))
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return nil, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.MicroServiceStatusList, nil
}

func GetMicroServiceStatusByID(asConfig allSyncModel.AllSyncConfig, id string) (mss allSyncModel.MicroServiceStatusResponse, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	statusCode, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("MicroServiceStatus/%s", id), http.MethodGet, nil, nil)
	if statusCode != 200 {
		return mss, errors.New(fmt.Sprintf("Get Micro Service Status ERROR - %v", msg))
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return mss, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.MicroServiceStatus, nil
}

func CreateMicroServiceStatus(asConfig allSyncModel.AllSyncConfig, mssReq allSyncModel.MicroServiceStatusRequest) (mss allSyncModel.MicroServiceStatusResponse, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	statusCode, msg, respData := RequestAllSync(asConfig, "MicroServiceStatus", http.MethodPost, mssReq, nil)
	if statusCode != 200 {
		return mss, errors.New(fmt.Sprintf("Create Micro Service Status ERROR - %v", msg))
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return mss, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.MicroServiceStatus, nil
}

func UpdateMicroServiceStatus(asConfig allSyncModel.AllSyncConfig, id string, mssReq bson.M) (mss allSyncModel.MicroServiceStatusResponse, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	statusCode, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("MicroServiceStatus/%s", id), http.MethodPut, mssReq, nil)
	if statusCode != 200 {
		return mss, errors.New(fmt.Sprintf("Update Micro Service Status ERROR - %v", msg))
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return mss, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.MicroServiceStatus, nil
}

func GetOAuth2TokenByAppID(asConfig allSyncModel.AllSyncConfig, appID string) (token allSyncModel.OAuth2Token, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("OAuth2Token/applicationID/%s", appID), http.MethodGet, nil, nil)
	if status != 200 {
		return token, errors.New("Get OAuth2Token by App ID ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return token, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.OAuth2Token, nil
}

func CreateOAuth2Token(asConfig allSyncModel.AllSyncConfig, tokenReq allSyncModel.OAuth2TokenRequest) (tokenResp allSyncModel.OAuth2Token, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, "OAuth2Token", http.MethodPost, tokenReq, nil)
	if status != 200 {
		return tokenResp, errors.New("Get All Application ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return tokenResp, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.OAuth2Token, nil
}

func UpdateOAuth2Token(asConfig allSyncModel.AllSyncConfig, id string, tokenReq bson.M) (tokenResp allSyncModel.OAuth2Token, err error) {
	var allSyncResp allSyncModel.ToAppResponse

	status, msg, respData := RequestAllSync(asConfig, fmt.Sprintf("OAuth2Token/%s", id), http.MethodPut, tokenReq, nil)
	if status != 200 {
		return tokenResp, errors.New("Get All Application ERROR - " + msg)
	}

	if err := json.Unmarshal(respData, &allSyncResp); err != nil {
		return tokenResp, errors.New("JSON Unmarshal ERROR - " + err.Error())
	}

	return allSyncResp.Data.OAuth2Token, nil
}

func HMACTimeStampFormatConvert(dateStr string) string {
	r := strings.NewReplacer("YYYY", "2006",
		"MM", "01",
		"DD", "02",
		"hh", "15",
		"mm", "04",
		"ss", "05",
	)

	return r.Replace(dateStr)
}

func ComputeSHA(algorithm allSyncModel.HMACAlgorithm, dataToSign string) string {
	var h = GetSHAHash(algorithm)()

	h.Write([]byte(dataToSign))
	return hex.EncodeToString(h.Sum(nil))
}

func GetSHAHash(algorithm allSyncModel.HMACAlgorithm) func() hash.Hash {
	switch algorithm {
	case allSyncModel.HMACAlgorithmSHA1, allSyncModel.HMACAlgorithmHMACSHA1:
		return sha1.New
	case allSyncModel.HMACAlgorithmSHA256, allSyncModel.HMACAlgorithmHMACSHA256:
		return sha256.New
	case allSyncModel.HMACAlgorithmSHA512, allSyncModel.HMACAlgorithmHMACSHA512:
		return sha512.New
	default:
		return sha1.New
	}
}

func GetJSONDataMapping(data string) (m map[string]interface{}, err error) {
	// xml is an io.Reader
	xmls := strings.NewReader(data)
	jsons, err := xj.Convert(xmls)
	if err != nil {
		return m, errors.New("XML To JSON ERROR - " + err.Error())
	}

	// Attempt to unmarshal data as XML
	if json.Unmarshal(jsons.Bytes(), &m) == nil {
		return m, nil
	}

	// Attempt to unmarshal data as JSON
	if json.Unmarshal([]byte(data), &m) == nil {
		return m, nil
	}

	// If neither attempt succeeds, return empty string
	return nil, errors.New("Not match JSON or XML type ")
}

func LogsAddLog(logs []allSyncModel.QueueLog, funcName, log1, log2, log3 string) []allSyncModel.QueueLog {
	logs = append(logs, allSyncModel.QueueLog{
		FunctionName: funcName,
		Log1:         log1,
		Log2:         log2,
		Log3:         log3,
		LogDate:      time.Now().Unix(),
	})

	return logs
}

func SQLConnect(sqlConf allSyncModel.SQLConfig) (db *sql.DB, err error) {
	var connStr string

	if sqlConf.User == "" {
		// Windows authentication
		connStr = fmt.Sprintf("sqlserver://%s?database=%s&encrypt=disable&connection+timeout=300&trusted_connection=yes", sqlConf.Server, sqlConf.DatabaseName)
	} else {
		// SQL Server authentication
		connStr = fmt.Sprintf("sqlserver://%s:%s@%s?database=%s&encrypt=disable&connection+timeout=300", sqlConf.User, url.QueryEscape(sqlConf.Password), sqlConf.Server, sqlConf.DatabaseName)
	}

	if !helper.IsDriverRegistered("sqlserver") {
		apmsql.Register("sqlserver", &mssql.Driver{})
	}

	db, err = sql.Open("sqlserver", connStr)
	if err != nil {
		return db, errors.New("Open SQL DB ERROR - " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return db, errors.New("Ping Database ERROR - " + err.Error())
	}

	return
}

func SQLExecuteQuery(db *sql.DB, query string) (objects []map[string]interface{}, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db.PingContext(ctx)

	rows, err := db.Query(query)
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Excecute Query ERROR - %s", err.Error()))
	}
	defer rows.Close()

	// Get column names
	columnNames, err := rows.Columns()
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Get collumn names ERROR - %s", err.Error()))
	}

	var hasJsonText = false
	if len(columnNames) == 1 && columnNames[0] == "json" {
		hasJsonText = true
	}

	// Get column types
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Get collumn types ERROR - %s", err.Error()))
	}

	// Iterate through rows
	for rows.Next() {
		// Create a slice to hold column values
		columns := make([]interface{}, len(columnNames))

		// Create a slice to hold pointers to each column value
		columnPointers := make([]interface{}, len(columnNames))

		// Initialize pointers to each column value
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the row into the columns slice
		if err := rows.Scan(columnPointers...); err != nil {
			return objects, errors.New(fmt.Sprintf("Row scan ERROR - %s", err.Error()))
		}

		if hasJsonText {
			val := columns[0]

			// Convert column values to appropriate JSON types
			switch v := val.(type) {
			case string:
				// Handle string data type explicitly
				if err = json.Unmarshal([]byte(v), &objects); err != nil {
					return objects, errors.New("Unmarshal string to JSON ERROR - " + err.Error())
				}
				break
			default:
				return objects, errors.New("json field is not a text to marshal to JSON format")
			}

			return
		}

		// Create a map to hold the row data
		rowMap := make(map[string]interface{})

		// Iterate through columns and retrieve values
		for i, colName := range columnNames {
			val := columns[i]

			if strings.Contains(strings.ToLower(colName), "_query") {
				q, ok := val.(string)
				if !ok {
					return objects, errors.New("Query is not a string ERROR - " + err.Error())
				}

				objs, err := SQLExecuteQuery(db, q)
				if err != nil {
					return objects, errors.New("Exec SQL Query ERROR - " + err.Error())
				}

				//remove "_query" from colName
				colName = helper.ReplaceIgnoreCase(colName, "_query", "")

				rowMap[colName] = objs

				continue
			}

			// Convert column values to appropriate JSON types
			switch v := val.(type) {
			case nil:
				rowMap[colName] = nil
			case int, int32, int64:
				rowMap[colName] = v
			case float64:
				rowMap[colName] = v
			case []byte:
				// Convert []byte to string for text-like data types
				colType := columnTypes[i].DatabaseTypeName()
				switch colType {
				case "VARCHAR", "TEXT", "NVARCHAR":
					rowMap[colName] = string(v)
				case "DECIMAL":
					{
						d, err := strconv.ParseFloat(string(v), 64)
						if err != nil {
							return objects, errors.New(fmt.Sprintf("Parse Decimal [%v] ERROR - %s", v, err.Error()))
						}

						rowMap[colName] = d
					}
				default:
					rowMap[colName] = v // Keep []byte for other binary-like data types
				}
			case time.Time:
				// Convert time.Time to string in a specific format
				rowMap[colName] = v.Format("2006-01-02 15:04:05")
			case string:
				// Handle string data type explicitly
				rowMap[colName] = v
			default:
				rowMap[colName] = fmt.Sprintf("%v", v)
			}
		}

		// Append the row map to the objects slice
		objects = append(objects, rowMap)
	}

	return
}

func SQLExecuteRawQuery(db *sql.DB, query string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db.PingContext(ctx)

	rows, err := db.Query(query)
	if err != nil {
		return errors.New(fmt.Sprintf("Excecute Raw Query ERROR - %s", err.Error()))
	}
	defer rows.Close()

	return
}

func OracleConnect(oracleConf allSyncModel.OracleConfig) (db *sql.DB, err error) {
	var connStr string

	if oracleConf.User == "" {
		// Windows authentication
		connStr = fmt.Sprintf("oracle://%s:1521/%s", oracleConf.Server, oracleConf.DatabaseName)
	} else {
		// SQL Server authentication
		connStr = fmt.Sprintf("oracle://%s:%s@%s:1521/%s", oracleConf.User, url.QueryEscape(oracleConf.Password), oracleConf.Server, oracleConf.DatabaseName)
	}

	if !helper.IsDriverRegistered("oracle") {
		apmsql.Register("oracle", &mssql.Driver{})
	}

	db, err = sql.Open("oracle", connStr)
	if err != nil {
		return db, errors.New("Open Oracle DB ERROR - " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return db, errors.New("Ping Database ERROR - " + err.Error())
	}

	return
}

func OracleExecuteQuery(db *sql.DB, query string) (objects []map[string]interface{}, err error) {
	// Defer a function that recovers from panic and sets the error
	defer func() {
		if r := recover(); r != nil {
			// Convert the panic into an error and assign it to the named error return variable
			err = fmt.Errorf("panic occurred: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db.PingContext(ctx)

	rows, err := db.Query(query)
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Excecute Query ERROR - %s", err.Error()))
	}
	defer rows.Close()

	// Get column names
	columnNames, err := rows.Columns()
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Get collumn names ERROR - %s", err.Error()))
	}

	var hasJsonText = false
	if len(columnNames) == 1 && columnNames[0] == "json" {
		hasJsonText = true
	}

	// Get column types
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Get collumn types ERROR - %s", err.Error()))
	}

	// Iterate through rows
	for rows.Next() {
		// Create a slice to hold column values
		columns := make([]interface{}, len(columnNames))

		// Create a slice to hold pointers to each column value
		columnPointers := make([]interface{}, len(columnNames))

		// Initialize pointers to each column value
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the row into the columns slice
		if err := rows.Scan(columnPointers...); err != nil {
			return objects, errors.New(fmt.Sprintf("Row scan ERROR - %s", err.Error()))
		}

		if hasJsonText {
			val := columns[0]

			// Convert column values to appropriate JSON types
			switch v := val.(type) {
			case string:
				// Handle string data type explicitly
				if err = json.Unmarshal([]byte(v), &objects); err != nil {
					return objects, errors.New("Unmarshal string to JSON ERROR - " + err.Error())
				}
				break
			default:
				return objects, errors.New("json field is not a text to marshal to JSON format")
			}

			return
		}

		// Create a map to hold the row data
		rowMap := make(map[string]interface{})

		// Iterate through columns and retrieve values
		for i, colName := range columnNames {
			val := columns[i]

			if strings.Contains(strings.ToLower(colName), "_query") {
				q, ok := val.(string)
				if !ok {
					return objects, errors.New("Query is not a string ERROR - " + err.Error())
				}

				objs, err := OracleExecuteQuery(db, q)
				if err != nil {
					return objects, errors.New("Exec Oracle Query ERROR - " + err.Error())
				}

				//remove _query from colName
				colName = helper.ReplaceIgnoreCase(colName, "_query", "")

				rowMap[colName] = objs

				continue
			}

			// Convert column values to appropriate JSON types
			switch v := val.(type) {
			case nil:
				rowMap[colName] = nil
			case int, int32, int64:
				rowMap[colName] = v
			case float64:
				rowMap[colName] = v
			case []byte:
				// Convert []byte to string for text-like data types
				colType := columnTypes[i].DatabaseTypeName()
				switch colType {
				case "VARCHAR", "TEXT", "NVARCHAR":
					rowMap[colName] = string(v)
				case "DECIMAL":
					{
						d, err := strconv.ParseFloat(string(v), 64)
						if err != nil {
							return objects, errors.New(fmt.Sprintf("Parse Decimal [%v] ERROR - %s", v, err.Error()))
						}

						rowMap[colName] = d
					}
				default:
					rowMap[colName] = v // Keep []byte for other binary-like data types
				}
			case time.Time:
				// Convert time.Time to string in a specific format
				rowMap[colName] = v.Format("2006-01-02 15:04:05")
			case string:
				// Handle string data type explicitly
				rowMap[colName] = v
			default:
				rowMap[colName] = fmt.Sprintf("%v", v)
			}
		}

		// Append the row map to the objects slice
		objects = append(objects, rowMap)
	}

	return
}

func OracleExecuteRawQuery(db *sql.DB, query string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db.PingContext(ctx)

	rows, err := db.Query(query)
	if err != nil {
		return errors.New(fmt.Sprintf("Excecute Query ERROR - %s", err.Error()))
	}
	defer rows.Close()

	return
}

func SAPHanaConnect(conf allSyncModel.SAPHanaConfig) (db *sql.DB, err error) {
	var connStr string

	if conf.User == "" {
		// Windows authentication
		connStr = fmt.Sprintf("hdb://%s?databaseName=%s", conf.Server, conf.DatabaseName)
	} else {
		// SAP Hana Server authentication
		connStr = fmt.Sprintf("hdb://%s:%s@%s?databaseName=%s", conf.User, conf.Password, conf.Server, conf.DatabaseName)
	}

	if !helper.IsDriverRegistered("hdb") {
		apmsql.Register("hdb", &mssql.Driver{})
	}

	db, err = sql.Open("hdb", connStr)
	if err != nil {
		return db, errors.New("Open SAP Hana DB ERROR - " + err.Error())
	}

	_, err = db.ExecContext(context.Background(), fmt.Sprintf("SET SCHEMA %s", conf.Schema))
	if err != nil {
		return db, errors.New("SET Schema ERROR - " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return db, errors.New("Ping Database ERROR - " + err.Error())
	}

	return
}

func SAPHanaExecuteQuery(db *sql.DB, query string) (objects []map[string]interface{}, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db.PingContext(ctx)

	rows, err := db.Query(query)
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Excecute Query ERROR - %s", err.Error()))
	}
	defer rows.Close()

	// Get column names
	columnNames, err := rows.Columns()
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Get collumn names ERROR - %s", err.Error()))
	}

	var hasJsonText = false
	if len(columnNames) == 1 && columnNames[0] == "json" {
		hasJsonText = true
	}

	// Get column types
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return objects, errors.New(fmt.Sprintf("Get collumn types ERROR - %s", err.Error()))
	}

	// Iterate through rows
	for rows.Next() {
		// Create a slice to hold column values
		columns := make([]interface{}, len(columnNames))

		// Create a slice to hold pointers to each column value
		columnPointers := make([]interface{}, len(columnNames))

		// Initialize pointers to each column value
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the row into the columns slice
		if err := rows.Scan(columnPointers...); err != nil {
			return objects, errors.New(fmt.Sprintf("Row scan ERROR - %s", err.Error()))
		}

		if hasJsonText {
			val := columns[0]

			// Convert column values to appropriate JSON types
			switch v := val.(type) {
			case string:
				// Handle string data type explicitly
				if err = json.Unmarshal([]byte(v), &objects); err != nil {
					return objects, errors.New("Unmarshal string to JSON ERROR - " + err.Error())
				}
				break
			default:
				return objects, errors.New("json field is not a text to marshal to JSON format")
			}

			return
		}

		// Create a map to hold the row data
		rowMap := make(map[string]interface{})

		// Iterate through columns and retrieve values
		for i, colName := range columnNames {
			val := columns[i]

			if strings.Contains(strings.ToLower(colName), "_query") {
				q, ok := val.(string)
				if !ok {
					return objects, errors.New("Query is not a string ERROR - " + err.Error())
				}

				objs, err := SAPHanaExecuteQuery(db, q)
				if err != nil {
					return objects, errors.New("Exec SAP Hana Query ERROR - " + err.Error())
				}

				//remove _query from colName
				colName = helper.ReplaceIgnoreCase(colName, "_query", "")

				rowMap[colName] = objs

				continue
			}

			// Convert column values to appropriate JSON types
			switch v := val.(type) {
			case nil:
				rowMap[colName] = nil
			case int, int32, int64:
				rowMap[colName] = v
			case float64:
				rowMap[colName] = v
			case []byte:
				// Convert []byte to string for text-like data types
				colType := columnTypes[i].DatabaseTypeName()
				switch colType {
				case "VARCHAR", "TEXT", "NVARCHAR":
					rowMap[colName] = string(v)
				case "DECIMAL":
					{
						d, err := strconv.ParseFloat(string(v), 64)
						if err != nil {
							return objects, errors.New(fmt.Sprintf("Parse Decimal [%v] ERROR - %s", v, err.Error()))
						}

						rowMap[colName] = d
					}
				default:
					rowMap[colName] = v // Keep []byte for other binary-like data types
				}
			case time.Time:
				// Convert time.Time to string in a specific format
				rowMap[colName] = v.Format("2006-01-02 15:04:05")
			case string:
				// Handle string data type explicitly
				rowMap[colName] = v
			default:
				rowMap[colName] = fmt.Sprintf("%v", v)
			}
		}

		// Append the row map to the objects slice
		objects = append(objects, rowMap)
	}

	return
}

func SAPHanaExecuteRawQuery(db *sql.DB, query string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db.PingContext(ctx)

	rows, err := db.Query(query)
	if err != nil {
		return errors.New(fmt.Sprintf("Excecute Query ERROR - %s", err.Error()))
	}
	defer rows.Close()

	return
}

func FunctionsDenomination(denoList []float64, payment float64) (amountList []float64) {
	// Author Mẫn
	sort.Slice(denoList, func(i, j int) bool {
		return denoList[i] > denoList[j]
	})

	for i, bigMoney := range denoList {
		amountList = append(amountList, math.Ceil(payment/bigMoney)*bigMoney)

		if i != len(denoList)-1 {
			if payment-bigMoney > denoList[i+1] {
				if math.Ceil((payment-bigMoney)/denoList[i+1])*denoList[i+1]+bigMoney <= denoList[0] {
					amountList = append(amountList, math.Ceil((payment-bigMoney)/denoList[i+1])*denoList[i+1]+bigMoney)
				}
			}
		}
	}

	amountList = helper.RemoveDuplicate(amountList)

	sort.Slice(amountList, func(i, j int) bool {
		return amountList[i] < amountList[j]
	})

	return
}

func IncludeP12ToTLSConfig(p12 allSyncModel.P12) (t *tls.Config, err error) {
	client := &http.Client{}

	resp, err := client.Get(p12.FilePath)
	if err != nil {
		return t, errors.New("Get File ERROR - " + err.Error())
	}
	defer resp.Body.Close()

	pfxData, err := io.ReadAll(resp.Body)
	if err != nil {
		return t, errors.New("Read body ERROR - " + err.Error())
	}

	// Decode the PFX data to extract the private key, certificate, and CA certificates
	blocks, err := pkcs12.ToPEM(pfxData, p12.Password)
	if err != nil {
		return t, errors.New("Decoding PFX file ERROR - " + err.Error())
	}

	var privateKeyPEM, certPEM []byte
	caCertPool := x509.NewCertPool()

	for _, b := range blocks {
		if b.Type == "PRIVATE KEY" {
			privateKeyPEM = pem.EncodeToMemory(b)
		} else if b.Type == "CERTIFICATE" {
			if certPEM == nil {
				certPEM = pem.EncodeToMemory(b)
			} else {
				caCertPool.AppendCertsFromPEM(pem.EncodeToMemory(b))
			}
		}
	}

	// Load the certificate and private key into a tls.Certificate
	tlsCert, err := tls.X509KeyPair(certPEM, privateKeyPEM)
	if err != nil {
		return t, errors.New("Loading certificate and key ERROR - " + err.Error())
	}

	// Load system root CAs
	systemCertPool, err := x509.SystemCertPool()
	if err != nil {
		return t, errors.New("Loading system cert pool ERROR - " + err.Error())
	}
	if systemCertPool == nil {
		systemCertPool = x509.NewCertPool()
	}

	// Extract CA certificates from the PFX file
	var caCerts []*x509.Certificate
	for _, b := range blocks {
		if b.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(b.Bytes)
			if err != nil {
				continue
			}
			caCerts = append(caCerts, cert)
		}
	}

	// Add the CA certificates to the system CA pool
	for _, cert := range caCerts {
		systemCertPool.AddCert(cert)
	}

	// Create a TLS config with the certificate and combined CA certificate pool
	t = &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		RootCAs:      systemCertPool,
	}

	return
}

func PostTenantS3File(asConfig allSyncModel.AllSyncConfig, filePath string) (s3FilePath string, msg string, err error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		err = errors.New(fmt.Sprintf("Open file [%s] ERROR - %s", filePath, err.Error()))
		return
	}
	defer file.Close()

	// Create a buffer to hold the multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create the form file field
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		err = errors.New(fmt.Sprintf("Creating form file ERROR - %s", err.Error()))
		return
	}

	// Copy the file content to the form file field
	if _, err = io.Copy(part, file); err != nil {
		err = errors.New(fmt.Sprintf("Copying file content ERROR - %s", err.Error()))
		return
	}

	// Close the writer to finalize the multipart form data
	if err = writer.Close(); err != nil {
		err = errors.New(fmt.Sprintf("Closing form file ERROR - %s", err.Error()))
		return
	}

	// Create a new HTTP request with the multipart form data
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/Tenant/UploadS3", asConfig.SystemAPIURL), &requestBody)
	if err != nil {
		err = errors.New(fmt.Sprintf("Creating request ERROR - %s", err.Error()))
		return
	}

	// Set the content type header
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("tenantID", asConfig.TenantID)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", asConfig.Token))
	req.Header.Set("Accept-Encoding", "gzip,deflate,br")

	// Send the request using the default HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		err = errors.New(fmt.Sprintf("Sending request ERROR - %s", err.Error()))
		return
	}

	if resp != nil {
		//if resp.Header.Get("Content-Encoding") == "gzip" {
		var (
			reader       *gzip.Reader
			responseData []byte
		)
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			err = errors.New(fmt.Sprintf("Gzip Read ERROR - %s", err.Error()))
			return
		}
		defer reader.Close()

		responseData, err = io.ReadAll(reader)
		if err != nil {
			err = errors.New(fmt.Sprintf("Read Uncompress data ERROR - %s", err.Error()))
			return
		}

		var allSyncResp allSyncModel.ToAppResponse
		if err = json.Unmarshal(responseData, &allSyncResp); err != nil {
			err = errors.New(fmt.Sprintf("Json Unmarshal AllSync Response format ERROR - %s", err.Error()))
			return
		}
		msg = allSyncResp.Message

		s3FilePath = allSyncResp.Data.FileURL

		defer resp.Body.Close()
	}

	return
}
