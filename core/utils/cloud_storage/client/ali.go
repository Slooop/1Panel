package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-resty/resty/v2"
)

type aliClient struct {
	token   string
	driveID string
}

func NewALIClient(vars map[string]interface{}) (*aliClient, error) {
	drive_id := loadParamFromVars("drive_id", vars)

	token, err := RefreshALIToken(vars)
	if err != nil {
		return nil, err
	}
	return &aliClient{token: token, driveID: drive_id}, nil
}

func (a aliClient) ListBuckets() ([]interface{}, error) {
	return nil, nil
}

func (a aliClient) Upload(src, target string) (bool, error) {
	target = path.Join("/root", target)
	parentID := "root"
	var err error
	if path.Dir(target) != "/root" {
		parentID, err = a.mkdirWithPath(path.Dir(target))
		if err != nil {
			return false, err
		}
	}
	file, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return false, err
	}
	data := map[string]interface{}{
		"drive_id":        a.driveID,
		"part_info_list":  makePartInfoList(fileInfo.Size()),
		"parent_file_id":  parentID,
		"name":            path.Base(src),
		"type":            "file",
		"size":            fileInfo.Size(),
		"check_name_mode": "auto_rename",
	}
	client := resty.New()
	url := "https://api.alipan.com/v2/file/create"

	resp, err := client.R().
		SetHeader("Authorization", a.token).
		SetBody(data).
		Post(url)
	if err != nil {
		return false, err
	}

	var createResp createFileResp
	if err := json.Unmarshal(resp.Body(), &createResp); err != nil {
		return false, err
	}
	for _, part := range createResp.PartInfoList {
		err = a.uploadPart(part.UploadURL, io.LimitReader(file, 1024*1024*1024))
		if err != nil {
			return false, err
		}
	}

	if err := a.completeUpload(createResp.UploadID, createResp.FileID); err != nil {
		return false, err
	}
	return true, nil
}

func (a aliClient) loadFileWithParentID(parentID string) ([]fileInfo, error) {
	client := resty.New()
	data := map[string]interface{}{
		"drive_id":       a.driveID,
		"fields":         "*",
		"limit":          100,
		"parent_file_id": parentID,
	}
	url := "https://api.alipan.com/v2/file/list"
	resp, err := client.R().
		SetHeader("Authorization", a.token).
		SetBody(data).
		Post(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("load file list failed, code: %v, err: %v", resp.StatusCode(), string(resp.Body()))
	}
	var fileResp fileResp
	if err := json.Unmarshal(resp.Body(), &fileResp); err != nil {
		return nil, err
	}
	return fileResp.Items, nil
}

func (a aliClient) mkdirWithPath(target string) (string, error) {
	pathItems := strings.Split(target, "/")
	var (
		fileInfos []fileInfo
		err       error
	)
	parentID := "root"
	for i := 0; i < len(pathItems); i++ {
		if len(pathItems[i]) == 0 {
			continue
		}
		fileInfos, err = a.loadFileWithParentID(parentID)
		if err != nil {
			return "", err
		}
		isEnd := false
		if i == len(pathItems)-2 {
			isEnd = true
		}
		exist := false
		for _, item := range fileInfos {
			if item.Name == pathItems[i+1] {
				parentID = item.FileID
				if isEnd {
					return item.FileID, nil
				} else {
					exist = true
				}
			}
		}
		if !exist {
			parentID, err = a.mkdir(parentID, pathItems[i+1])
			if err != nil {
				return parentID, err
			}
			if isEnd {
				return parentID, nil
			}
		}
	}
	return "", errors.New("mkdir failed.")
}

func (a aliClient) mkdir(parentID, name string) (string, error) {
	client := resty.New()
	data := map[string]interface{}{
		"drive_id":       a.driveID,
		"name":           name,
		"type":           "folder",
		"limit":          100,
		"parent_file_id": parentID,
	}
	url := "https://api.alipan.com/v2/file/create"
	resp, err := client.R().
		SetHeader("Authorization", a.token).
		SetBody(data).
		Post(url)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != 201 {
		return "", fmt.Errorf("mkdir %s failed, code: %v, err: %v", name, resp.StatusCode(), string(resp.Body()))
	}
	var mkdirResp mkdirResp
	if err := json.Unmarshal(resp.Body(), &mkdirResp); err != nil {
		return "", err
	}
	return mkdirResp.FileID, nil
}

type fileResp struct {
	Items []fileInfo `json:"items"`
}
type fileInfo struct {
	FileID string `json:"file_id"`
	Name   string `json:"name"`
	Size   int    `json:"size"`
}

type mkdirResp struct {
	FileID string `json:"file_id"`
}

type partInfo struct {
	PartNumber        int    `json:"part_number"`
	UploadURL         string `json:"upload_url"`
	InternalUploadURL string `json:"internal_upload_url"`
	ContentType       string `json:"content_type"`
}

func makePartInfoList(size int64) []*partInfo {
	var res []*partInfo
	maxPartSize := int64(1024 * 1024 * 1024)
	partInfoNum := int(size / maxPartSize)
	if size%maxPartSize > 0 {
		partInfoNum += 1
	}

	for i := 0; i < partInfoNum; i++ {
		res = append(res, &partInfo{PartNumber: i + 1})
	}

	return res
}

type createFileResp struct {
	Type         string      `json:"type"`
	RapidUpload  bool        `json:"rapid_upload"`
	DomainId     string      `json:"domain_id"`
	DriveId      string      `json:"drive_id"`
	FileName     string      `json:"file_name"`
	EncryptMode  string      `json:"encrypt_mode"`
	Location     string      `json:"location"`
	UploadID     string      `json:"upload_id"`
	FileID       string      `json:"file_id"`
	PartInfoList []*partInfo `json:"part_info_list,omitempty"`
}

func (a aliClient) uploadPart(uri string, reader io.Reader) error {
	req, err := http.NewRequest(http.MethodPut, uri, reader)
	if err != nil {
		return err
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("handle upload park file with url failed, code: %v", response.StatusCode)
	}

	return nil
}

func (a *aliClient) completeUpload(uploadID, fileID string) error {
	client := resty.New()
	data := map[string]interface{}{
		"drive_id":  a.driveID,
		"upload_id": uploadID,
		"file_id":   fileID,
	}

	url := "https://api.aliyundrive.com/v2/file/complete"
	resp, err := client.R().
		SetHeader("Authorization", a.token).
		SetBody(data).
		Post(url)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("complete upload failed, err: %v", string(resp.Body()))
	}

	return nil
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
}

func RefreshALIToken(varMap map[string]interface{}) (string, error) {
	refresh_token := loadParamFromVars("refresh_token", varMap)
	if len(refresh_token) == 0 {
		return "", errors.New("no such refresh token find in db")
	}
	client := resty.New()
	data := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": refresh_token,
	}

	url := "https://api.aliyundrive.com/token/refresh"
	resp, err := client.R().
		SetBody(data).
		Post(url)

	if err != nil {
		return "", fmt.Errorf("load account token failed, err: %v", err)
	}
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("load account token failed, code: %v", resp.StatusCode())
	}
	var respItem tokenResp
	if err := json.Unmarshal(resp.Body(), &respItem); err != nil {
		return "", err
	}
	return respItem.AccessToken, nil
}