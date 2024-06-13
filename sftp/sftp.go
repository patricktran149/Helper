package sftp

import (
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type SftpClient struct {
	Host, User, Password string
	Port                 int
	SSHClient            *ssh.Client
	IsConnected          bool
	*sftp.Client
}

func (sc *SftpClient) Connect() (err error) {
	switch {
	case `` == strings.TrimSpace(sc.Host),
		`` == strings.TrimSpace(sc.User),
		`` == strings.TrimSpace(sc.Password),
		0 >= sc.Port || sc.Port > 65535:
		return errors.New("Invalid parameters ")
	}

	configClient := &ssh.ClientConfig{
		User:            sc.User,
		Auth:            []ssh.AuthMethod{ssh.Password(sc.Password)},
		Timeout:         0, //30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// connect to ssh
	addr := fmt.Sprintf("%s:%d", sc.Host, sc.Port)
	conn, err := ssh.Dial("tcp", addr, configClient)
	if err != nil {
		return errors.New("SSH Dial ERROR - " + err.Error())
	}

	sc.SSHClient = conn
	// create sftp client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return errors.New("SFTP New client ERROR - " + err.Error())
	}
	sc.Client = client

	sc.IsConnected = true

	return nil
}

// Upload file to sftp server
func (sc *SftpClient) UploadFile(localFile, remoteFile string) (err error) {
	srcFile, err := os.Open(localFile)
	if err != nil {
		return errors.New("Open local file ERROR - " + err.Error())
	}
	defer srcFile.Close()

	dstFile, err := sc.Create(remoteFile)
	if err != nil {
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return
}

// Upload Data to sftp server
func (sc *SftpClient) UploadData(data string, remoteFile string) (err error) {
	dstFile, err := sc.Create(remoteFile)
	if err != nil {
		return
	}
	defer dstFile.Close()

	r := strings.NewReader(data)

	_, err = io.Copy(dstFile, r)
	return
}

// Download file from sftp server
func (sc *SftpClient) Get(remoteFile string) (f *sftp.File, err error) {
	f, err = sc.Open(remoteFile)
	if err != nil {
		return
	}
	return
}

// Read filename from sftp server
func (sc *SftpClient) GetFileNames(remotePath, validExt string) (fileNames []string, err error) {
	srcs, err := sc.ReadDir(remotePath)
	if err != nil {
		return nil, errors.New("Read directory ERROR - " + err.Error())
	}
	for _, src := range srcs {
		if (filepath.Ext(strings.ToLower(src.Name())) == fmt.Sprintf(".%s", strings.ToLower(validExt)) || validExt == "") && !src.IsDir() {
			fileNames = append(fileNames, src.Name())
		}
	}
	return
}

func (sc *SftpClient) GetFolderNames(remotePath string) (folder []string, err error) {
	srcs, err := sc.ReadDir(remotePath)
	if err != nil {
		return nil, errors.New("Read directory ERROR - " + err.Error())
	}
	for _, src := range srcs {
		if src.IsDir() {
			folder = append(folder, src.Name())
		}
	}
	return
}

func (sc *SftpClient) GetFile(remoteFile, localFile string) (err error) {
	//SFTP File
	srcFile, err := sc.Open(remoteFile)
	if err != nil {
		return errors.New("Client open file ERROR - " + err.Error())
	}
	defer srcFile.Close()

	//Local file
	dstFile, err := os.Create(localFile)
	if err != nil {
		return errors.New("Create local file ERROR - " + err.Error())
	}
	defer dstFile.Close()

	//Copy SFTP file to local File
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return errors.New("Copy Remote file to Local file ERROR - " + err.Error())
	}
	return nil
}

func (sc *SftpClient) CheckExistingFile(remoteFile string) (exists bool, err error) {
	_, err = sc.Stat(remoteFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, errors.New("Stat remote file ERROR - " + err.Error())
		}
	} else {
		return true, nil
	}
}

func (sc *SftpClient) RemoveEmptyFiles(remoteDirPath string) (err error) {
	srcs, err := sc.ReadDir(remoteDirPath)
	if err != nil {
		return errors.New("Read Directory ERROR - " + err.Error())
	}

	for _, f := range srcs {
		if f.Size() == 0 {
			if err = sc.Remove(fmt.Sprintf("%s/%s", remoteDirPath, f.Name())); err != nil {
				return errors.New(fmt.Sprintf("Remove 0kb file [%s] ERROR - %s", f.Name(), err.Error()))
			}
		}
	}

	return
}

func (sc *SftpClient) CloseConnection() (err error) {
	if sc.IsConnected {
		sc.IsConnected = false

		if err = sc.SSHClient.Close(); err != nil {
			return errors.New("SFTP SSH Client close connection ERROR - " + err.Error())
		}

	}
	return
}
