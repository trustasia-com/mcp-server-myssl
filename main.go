package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"mcp-server-myssl",
		"1.0.0",
	)

	// Add tool
	tool := mcp.NewTool("domain_check",
		mcp.WithDescription("query the information of an IP or domain, ensure secure HTTPS deployments"),
		mcp.WithString("domain",
			mcp.Required(),
			mcp.Description("the domain to query"),
		),
		mcp.WithString("IP",
			mcp.Description("the ip to query"),
		),
	)

	healthCheck := mcp.NewTool("health_check",
		mcp.WithDescription("check the health of myssl server"),
	)

	// Add tool handler
	s.AddTool(tool, mysslReport)
	s.AddTool(healthCheck, mysslHealth)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func mysslHealth(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := MysslHealthCheck()
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(status), nil
}

func mysslReport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ak := os.Getenv("AccessKey")
	sk := os.Getenv("SecretKey")
	if ak == "" || sk == "" {
		return nil, errors.New("missing environment ak/sk variable")
	}

	domain, ok := request.Params.Arguments["domain"].(string)
	if !ok {
		return nil, errors.New("domain must be a string")
	}
	ip, _ := request.Params.Arguments["ip"].(string)
	result, err := MysslReport(domain, ip, ak, sk)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(result), nil
}

func MysslHealthCheck() (string, error) {
	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, "https://myssl.com/health", nil)

	if err != nil {
		fmt.Println(err)
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		fmt.Println("Error: ", res.StatusCode, res.Status)
		return "", fmt.Errorf("error: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return string(body), nil
}

func MysslReport(domain, ip, ak, sk string) (string, error) {
	currenttime := fmt.Sprintf("%d", time.Now().UTC().Unix())
	params := url.Values{}
	params.Add("partnerId", ak)
	params.Add("timestamp", currenttime)
	params.Add("expire", "120")
	params.Add("domain", domain)
	params.Add("port", "443")
	params.Add("ip", ip)

	// 固定格式
	var payload = "partnerId=" + ak + "&timestamp=" + currenttime + "&expire=" + "120" + "&domain=" + domain + "&port=" + "443" + "&ip=" + ip
	// payload = params.Encode()
	h := hmac.New(sha1.New, []byte(sk))
	h.Write([]byte(payload))
	sig := h.Sum(nil)
	params.Add("signature", hex.EncodeToString(sig))

	url := fmt.Sprintf("http://myssl.com/eeapi/v1/deep_analyze?%s", params.Encode())
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		fmt.Println("Error: ", res.StatusCode, res.Status)
		return "", fmt.Errorf("error: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return string(body), nil
}
