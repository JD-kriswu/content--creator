// backend/internal/service/feishu_api.go
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const feishuAPIBase = "https://open.feishu.cn/open-apis"

type FeishuAPI struct {
	AppID     string
	AppSecret string
	Token     string
	TokenExp  time.Time
}

func NewFeishuAPI(appID, appSecret string) *FeishuAPI {
	return &FeishuAPI{AppID: appID, AppSecret: appSecret}
}

func (a *FeishuAPI) GetToken() (string, error) {
	if a.Token != "" && time.Now().Before(a.TokenExp) {
		return a.Token, nil
	}

	url := feishuAPIBase + "/auth/v3/tenant_access_token/internal"
	body := map[string]string{"app_id": a.AppID, "app_secret": a.AppSecret}

	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Token string `json:"tenant_access_token"`
		Exp   int    `json:"expire"`
	}
	json.Unmarshal(respBody, &result)

	if result.Code != 0 {
		return "", fmt.Errorf("api error: %s", result.Msg)
	}

	a.Token = result.Token
	a.TokenExp = time.Now().Add(time.Duration(result.Exp-60) * time.Second)
	return a.Token, nil
}

func (a *FeishuAPI) CreateCard(chatID string, cardJSON string) (string, error) {
	token, err := a.GetToken()
	if err != nil {
		return "", err
	}

	url := feishuAPIBase + "/im/v1/messages?receive_id_type=chat_id"
	body := map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "interactive",
		"content":    json.RawMessage(cardJSON),
	}

	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	json.Unmarshal(respBody, &result)

	if result.Code != 0 {
		return "", fmt.Errorf("create card failed: code %d", result.Code)
	}
	return result.Data.MessageID, nil
}

func (a *FeishuAPI) UpdateCard(messageID string, cardJSON string) error {
	token, err := a.GetToken()
	if err != nil {
		return err
	}

	url := feishuAPIBase + "/im/v1/messages/" + messageID
	body := map[string]interface{}{
		"msg_type": "interactive",
		"content":  json.RawMessage(cardJSON),
	}

	reqBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}