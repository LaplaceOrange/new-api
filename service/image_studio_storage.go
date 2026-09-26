package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

type imageStudioStoreSnapshot struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

func imageStudioObjectURL(snapshot imageStudioStoreSnapshot, key string) (string, error) {
	base, err := url.Parse(snapshot.Endpoint)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return "", errors.New("invalid S3 endpoint")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + url.PathEscape(snapshot.Bucket) + "/" + url.PathEscape(key)
	return base.String(), nil
}

func imageStudioS3Request(ctx context.Context, snapshot imageStudioStoreSnapshot, key, method string, data []byte) (*http.Response, error) {
	target, err := imageStudioObjectURL(snapshot, key)
	if err != nil {
		return nil, err
	}
	var body io.Reader
	if data != nil {
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	payloadHash := hex.EncodeToString(hash[:])
	req.Header.Set("x-amz-content-sha256", payloadHash)
	if data != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	credentials := aws.Credentials{AccessKeyID: snapshot.AccessKey, SecretAccessKey: snapshot.SecretKey}
	if err := v4.NewSigner().SignHTTP(ctx, credentials, req, payloadHash, "s3", snapshot.Region, time.Now()); err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout:       40 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("object storage returned %d", resp.StatusCode)
	}
	return resp, nil
}

func FetchImageStudioResult(ctx context.Context, originURL string) ([]byte, error) {
	if err := ValidateStrictSSRFProtectedFetchURL(originURL); err != nil {
		return nil, err
	}
	noProxy := func(*http.Request) (*url.URL, error) { return nil, nil }
	client := newProtectedFetchHTTPClientWithProxy(nil, nil, strictFetchProtection, noProxy)
	client.Transport.(*ssrfProtectedRoundTripper).maxBodyBytes = 50 << 20
	client.CheckRedirect = checkStrictProtectedFetchRedirect
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, originURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image URL returned status %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 50<<20+1))
	if err != nil || len(data) > 50<<20 {
		return nil, errors.New("image URL response exceeds 50 MB or could not be read")
	}
	return data, nil
}

func StoreImageStudioAsset(ctx context.Context, config model.ImageStudioConfig, recordID uint, kind, mime string, data []byte) (*model.ImageStudioAsset, error) {
	if len(data) == 0 || len(data) > 50<<20 {
		return nil, errors.New("image is empty or too large")
	}
	key := strconv.FormatUint(uint64(recordID), 10) + "-" + common.GetRandomString(24)
	asset := &model.ImageStudioAsset{RecordID: recordID, Kind: kind, MimeType: mime}
	if config.StorageMode == "s3" {
		snapshot := imageStudioStoreSnapshot{
			Endpoint: config.S3Endpoint, Bucket: config.S3Bucket, Region: config.S3Region,
			AccessKey: config.S3AccessKey, SecretKey: config.S3SecretKey,
		}
		snapshotJSON, err := common.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
		resp, err := imageStudioS3Request(ctx, snapshot, key, http.MethodPut, data)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		asset.Backend, asset.Location, asset.StoreConfig = "s3", key, string(snapshotJSON)
	} else {
		root, err := filepath.Abs(config.LocalDir)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(root, filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			return nil, err
		}
		asset.Backend, asset.Location = "local", target
	}
	if err := model.DB.Create(asset).Error; err != nil {
		_ = DeleteImageStudioAsset(ctx, asset)
		return nil, err
	}
	return asset, nil
}

func ReadImageStudioAsset(ctx context.Context, asset *model.ImageStudioAsset) ([]byte, error) {
	if asset.Backend == "local" {
		file, err := os.Open(asset.Location)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return io.ReadAll(io.LimitReader(file, 50<<20+1))
	}
	var snapshot imageStudioStoreSnapshot
	if err := common.Unmarshal([]byte(asset.StoreConfig), &snapshot); err != nil {
		return nil, err
	}
	resp, err := imageStudioS3Request(ctx, snapshot, asset.Location, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 50<<20+1))
}

func DeleteImageStudioAsset(ctx context.Context, asset *model.ImageStudioAsset) error {
	if asset.Backend == "local" {
		err := os.Remove(asset.Location)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var snapshot imageStudioStoreSnapshot
	if err := common.Unmarshal([]byte(asset.StoreConfig), &snapshot); err != nil {
		return err
	}
	resp, err := imageStudioS3Request(ctx, snapshot, asset.Location, http.MethodDelete, nil)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func PurgeImageStudioRecord(ctx context.Context, record *model.ImageStudioRecord, removeRecord bool) error {
	var assets []model.ImageStudioAsset
	if err := model.DB.Where("record_id = ?", record.ID).Find(&assets).Error; err != nil {
		return err
	}
	for i := range assets {
		if err := DeleteImageStudioAsset(ctx, &assets[i]); err != nil {
			return err
		}
		if err := model.DB.Delete(&assets[i]).Error; err != nil {
			return err
		}
	}
	if removeRecord {
		return model.DB.Delete(record).Error
	}
	return model.DB.Model(record).Updates(map[string]any{"prompt": "", "error": "", "status": "expired"}).Error
}
