package microsoftAzureEmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
)

type EmailConfig struct {
	UserMail string
	ClientId string
	TenantId string
	Secret   string
}

// EmailAddress represents an email recipient's address.
type EmailAddress struct {
	Address string `json:"address"`
}

// Recipient represents a recipient with an email address.
type Recipient struct {
	EmailAddress EmailAddress `json:"emailAddress"`
}

// ItemBody represents the body of the email.
type ItemBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// FileAttachment represents a file attachment.
type FileAttachment struct {
	ODataType    string `json:"@odata.type"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	ContentBytes string `json:"contentBytes"`
}

// Message represents the email message with multiple recipients and attachments.
type Message struct {
	Subject       string           `json:"subject"`
	Body          ItemBody         `json:"body"`
	ToRecipients  []Recipient      `json:"toRecipients"`
	BccRecipients []Recipient      `json:"bccRecipients,omitempty"` // Added for BCC
	Attachments   []FileAttachment `json:"attachments,omitempty"`
}

// SendMailPayload represents the body of the sendMail request.
type SendMailPayload struct {
	Message         Message `json:"message"`
	SaveToSentItems bool    `json:"saveToSentItems"`
}

func SendEmailAttachment(emailConfig EmailConfig, filenames []string, to, bcc []string, message string, subject string) (err error) {
	var url = fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/sendMail", emailConfig.UserMail)

	accessToken, err := GetEmailOAuthTokenWithSecret(emailConfig)
	if err != nil {
		return errors.New(fmt.Sprintf("Get Email OAuth Token With Secret ERROR - %s", err.Error()))
	}

	payload, err := CreateEmailPayload(subject, message, to, filenames, bcc)
	if err != nil {
		return errors.New(fmt.Sprintf("Create Email Payload ERROR - %s", err.Error()))
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return errors.New(fmt.Sprintf("Create Email Request ERROR - %s", err.Error()))
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.New("Read body ERROR - " + err.Error())
		}

		return fmt.Errorf("email send failed: %s", string(body))
	}

	return nil
}

func GetEmailOAuthTokenWithSecret(emailConfig EmailConfig) (accessToken string, err error) {
	var (
		authority = fmt.Sprintf("https://login.microsoftonline.com/%s", emailConfig.TenantId)
		scopes    = []string{"https://graph.microsoft.com/.default"}
	)

	// confidential clients have a credential, such as a secret or a certificate
	cred, err := confidential.NewCredFromSecret(emailConfig.Secret)
	if err != nil {
		return "", errors.New("NewCredFromSecret ERROR - " + err.Error())
	}

	confidentialClient, err := confidential.New(authority, emailConfig.ClientId, cred)
	if err != nil {
		return "", errors.New("confidential.New ERROR - " + err.Error())
	}

	result, err := confidentialClient.AcquireTokenSilent(context.Background(), scopes)
	if err != nil {
		// cache miss, authenticate with another AcquireToken... method
		result, err = confidentialClient.AcquireTokenByCredential(context.Background(), scopes)
		if err != nil {
			return "", errors.New("Acquire Token By Credential ERROR - " + err.Error())
		}
	}

	return result.AccessToken, nil
}

// CreateEmailPayload constructs the JSON payload for the sendMail request with multiple attachments.
func CreateEmailPayload(subject, body string, toEmail, filePaths, bccEmails []string) ([]byte, error) {
	var (
		attachments   = make([]FileAttachment, len(filePaths))
		toRecipients  = make([]Recipient, len(toEmail))
		bccRecipients = make([]Recipient, len(bccEmails))
	)

	// Build the recipient slices
	for i, email := range toEmail {
		toRecipients[i] = Recipient{EmailAddress: EmailAddress{Address: email}}
	}

	for i, email := range bccEmails {
		bccRecipients[i] = Recipient{EmailAddress: EmailAddress{Address: email}}
	}

	// Prepare the slice of attachments (same as before)
	for _, filePath := range filePaths {
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
		}

		base64Content := base64.StdEncoding.EncodeToString(fileBytes)
		contentType := mime.TypeByExtension(filepath.Ext(filePath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		attachments = append(attachments, FileAttachment{
			ODataType:    "#microsoft.graph.fileAttachment",
			Name:         filepath.Base(filePath),
			ContentType:  contentType,
			ContentBytes: base64Content,
		})
	}

	// Build the complete email payload
	payload := SendMailPayload{
		Message: Message{
			Subject:       subject,
			Body:          ItemBody{ContentType: "Text", Content: body},
			ToRecipients:  toRecipients,
			BccRecipients: bccRecipients,
			Attachments:   attachments,
		},
		SaveToSentItems: true,
	}

	return json.Marshal(payload)
}
