package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

func (s *syncer) getClient(server string, serverCert string) (*http.Client, error) {
	// Parse the server URL
	u, err := url.ParseRequestURI(server)
	if err != nil {
		return nil, err
	}

	var transport *http.Transport
	if u.Scheme == "http" {
		// Basic transport for clear-text HTTP
		transport = &http.Transport{
			DisableKeepAlives: true,
		}
	} else if u.Scheme == "https" {
		// Be picky on our cipher list
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
		}

		// If provided, pin the certificate
		if serverCert != "" {
			certBlock, _ := pem.Decode([]byte(serverCert))
			if certBlock == nil {
				return nil, fmt.Errorf("Failed to load pinned certificate")
			}

			cert, err := x509.ParseCertificate(certBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("Failed to parse pinned certificate: %v", err)
			}

			caCertPool := tlsConfig.RootCAs
			if caCertPool == nil {
				caCertPool = x509.NewCertPool()
			}

			caCertPool.AddCert(cert)
			tlsConfig.RootCAs = caCertPool
		}

		transport = &http.Transport{
			TLSClientConfig:   tlsConfig,
			DisableKeepAlives: true,
		}
	} else {
		return nil, fmt.Errorf("Unsupported server URL: %s", server)
	}

	// Create the new HTTP client
	client := http.Client{
		Transport: transport,
	}

	return &client, nil
}

func (s *syncer) websocket(server string, path string) (*websocket.Conn, error) {
	// Server-specific configuration
	var srv *http.Client
	var url string
	if server == "askgod" {
		srv = s.httpAskgod
		url = fmt.Sprintf("%s/1.0%s", s.config.AskgodURL, path)
	} else {
		return nil, fmt.Errorf("Unknown server: %s", server)
	}

	if strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("wss://%s", strings.TrimPrefix(url, "https://"))
	} else {
		url = fmt.Sprintf("ws://%s", strings.TrimPrefix(url, "http://"))
	}

	// Grab the http transport handler
	httpTransport := srv.Transport.(*http.Transport)

	// Setup a new websocket dialer based on it
	dialer := websocket.Dialer{
		TLSClientConfig: httpTransport.TLSClientConfig,
		Proxy:           httpTransport.Proxy,
	}

	// Establish the connection
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	return conn, err
}

type queryArgs struct {
	discourseUser string
	discourseKey  string
}

func (s *syncer) queryStruct(server string, method string, path string, data interface{}, target interface{}, args *queryArgs) error {
	var req *http.Request
	var err error

	// Server-specific configuration
	var srv *http.Client
	var url string
	if server == "askgod" {
		srv = s.httpAskgod
		url = fmt.Sprintf("%s/1.0%s", s.config.AskgodURL, path)
	} else if server == "discourse" {
		srv = s.httpDiscourse
		url = fmt.Sprintf("%s%s", s.config.DiscourseURL, path)
	} else {
		return fmt.Errorf("Unknown server: %s", server)
	}

	// Get a new HTTP request setup
	if data != nil {
		// Encode the provided data
		buf := bytes.Buffer{}
		err := json.NewEncoder(&buf).Encode(data)
		if err != nil {
			return err
		}

		// Some data to be sent along with the request
		req, err = http.NewRequest(method, url, &buf)
		if err != nil {
			return err
		}

		// Set the encoding accordingly
		req.Header.Set("Content-Type", "application/json")

		// Handle authentication
		if server == "discourse" {
			if args != nil && args.discourseUser != "" {
				req.Header.Set("Api-Username", args.discourseUser)
			} else {
				req.Header.Set("Api-Username", s.config.DiscourseAPIUser)
			}

			if args != nil && args.discourseKey != "" {
				req.Header.Set("Api-Key", args.discourseKey)
			} else {
				req.Header.Set("Api-Key", s.config.DiscourseAPIKey)
			}
		}
	} else {
		// No data to be sent along with the request
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return err
		}

		// Handle authentication
		if server == "discourse" {
			if args != nil && args.discourseUser != "" {
				req.Header.Set("Api-Username", args.discourseUser)
			} else {
				req.Header.Set("Api-Username", s.config.DiscourseAPIUser)
			}

			if args != nil && args.discourseKey != "" {
				req.Header.Set("Api-Key", args.discourseKey)
			} else {
				req.Header.Set("Api-Key", s.config.DiscourseAPIKey)
			}
		}
	}

	// Send the request
	resp, err := srv.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, err := ioutil.ReadAll(resp.Body)
		if err == nil && string(content) != "" {
			return fmt.Errorf("%s", strings.TrimSpace(string(content)))
		}

		return fmt.Errorf("%s: %s", url, resp.Status)
	}

	// Decode the response
	if target != nil {
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&target)
		if err != nil {
			return err
		}
	}

	return nil
}
