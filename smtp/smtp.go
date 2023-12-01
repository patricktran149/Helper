package smtp

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/scorredoira/email"
	"html/template"
	"io/ioutil"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

type SMTP struct {
	Server    string `json:"server" bson:"server"`
	User      string `json:"user" bson:"user"`
	Password  string `json:"password" bson:"password"`
	Port      int    `json:"port" bson:"port"`
	HasSSL    bool   `json:"hasSSL" bson:"hasSSL"`
	FromEmail string `json:"fromEmail" bson:"fromEmail"`
	BccEmail  string `json:"bccEmail" bson:"bccEmail"`
}

type SendEmail struct {
	Subject   string                 `json:"subject"`
	Variables map[string]interface{} `json:"variables"`
	ToEmails  []string               `json:"toEmails"`
	Data      template.HTML          `json:"data"`
}

func ParseTemplate(smtpConf SMTP, payload SendEmail, t *template.Template, attachFiles []string, folder string, isSendEmail, isRemoveFile bool) error {
	var (
		body bytes.Buffer
		//folder = fmt.Sprintf("%s/temp", config.LocalPathFiles)
		err error
	)

	if err = t.Execute(&body, payload.Variables); err != nil {
		return errors.New("Execute Parse template ERROR -" + err.Error())
	}

	if err = os.MkdirAll(folder, os.ModePerm); err != nil {
		return errors.New("Make dir all ERROR -" + err.Error())
	}

	fileName := fmt.Sprintf("%s/%d.html", folder, time.Now().UnixMilli())

	if err = ioutil.WriteFile(fileName, body.Bytes(), os.ModePerm); err != nil {
		return errors.New("Write File ERROR -" + err.Error())
	}

	if isSendEmail {
		if err = SendEmailAttachment(smtpConf, attachFiles, payload.ToEmails, body.Bytes(), payload.Subject, true); err != nil {
			return errors.New("Send Email ERROR -" + err.Error())
		}
	}

	if isRemoveFile {
		_ = os.Remove(fileName)
	}

	return nil
}

func SendEmailAttachment(smtpConf SMTP, filenames []string, to []string, body []byte, subject string, bcc bool) (err error) {
	var (
		addr         = smtpConf.Server + ":" + strconv.Itoa(smtpConf.Port)
		auth         = LoginAuth(smtpConf.User, smtpConf.Password)
		sender       = mail.Address{Name: smtpConf.FromEmail, Address: smtpConf.User}
		To           = to
		Subject      = subject
		emailContent = email.NewHTMLMessage(Subject, string(body))
	)

	emailContent.From = sender
	emailContent.To = To
	emailContent.Subject = Subject
	if bcc == true && smtpConf.BccEmail != "" {
		var listEmail []string
		listEmail = strings.Split(smtpConf.BccEmail, ",")
		emailContent.Bcc = listEmail
	}

	for _, file := range filenames {
		err = emailContent.Attach(file)
		if err != nil {
			return errors.New(fmt.Sprintf("Attach file [%s] ERROR - %s", file, err.Error()))
		}
	}

	if err = email.Send(addr, auth, emailContent); err != nil {
		return errors.New("Send Email ERROR - " + err.Error())
	}
	return nil
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unknown from Server ")
		}
	}
	return nil, nil
}

type loginAuth struct {
	username, password string
}
