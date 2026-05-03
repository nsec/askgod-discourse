package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

func (*syncer) getClient(server string, serverCert string) (*http.Client, error) {
	// Parse the server URL
	u, err := url.ParseRequestURI(server)
	if err != nil {
		return nil, err
	}

	var transport *http.Transport

	switch u.Scheme {
	case "http":
		// Basic transport for clear-text HTTP
		transport = &http.Transport{
			DisableKeepAlives: true,
		}
	case "https":
		// Be picky on our cipher list
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
		}

		// If provided, pin the certificate
		if serverCert != "" {
			certBlock, _ := pem.Decode([]byte(serverCert))
			if certBlock == nil {
				return nil, errors.New("failed to load pinned certificate")
			}

			cert, err := x509.ParseCertificate(certBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse pinned certificate: %w", err)
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
	default:
		return nil, fmt.Errorf("unsupported server URL: %s", server)
	}

	// Create the new HTTP client
	client := http.Client{
		Transport: transport,
	}

	return &client, nil
}

func (s *syncer) websocket(server string, path string) (*websocket.Conn, error) {
	// Server-specific configuration
	if server != "askgod" {
		return nil, fmt.Errorf("unknown server: %s", server)
	}

	srv := s.httpAskgod
	requestURL := fmt.Sprintf("%s/1.0%s", s.config.AskgodURL, path)

	rest, ok := strings.CutPrefix(requestURL, "https://")
	if ok {
		requestURL = "wss://" + rest
	} else {
		requestURL = "ws://" + strings.TrimPrefix(requestURL, "http://")
	}

	// Grab the http transport handler
	httpTransport, ok := srv.Transport.(*http.Transport)
	if !ok {
		return nil, errors.New("unexpected http client transport type")
	}

	// Setup a new websocket dialer based on it
	dialer := websocket.Dialer{
		TLSClientConfig: httpTransport.TLSClientConfig,
		Proxy:           httpTransport.Proxy,
	}

	// Establish the connection
	conn, resp, err := dialer.Dial(requestURL, nil)
	if resp != nil {
		_ = resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	return conn, nil
}

type queryArgs struct {
	discourseUser string
	discourseKey  string
}

func (s *syncer) queryStruct(server string, method string, path string, data any, target any, args *queryArgs) error {
	var (
		req        *http.Request
		err        error
		srv        *http.Client
		requestURL string
	)

	// Server-specific configuration
	switch server {
	case "askgod":
		srv = s.httpAskgod
		requestURL = fmt.Sprintf("%s/1.0%s", s.config.AskgodURL, path)
	case "discourse":
		srv = s.httpDiscourse
		requestURL = fmt.Sprintf("%s%s", s.config.DiscourseURL, path)
	default:
		return fmt.Errorf("unknown server: %s", server)
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
		req, err = http.NewRequestWithContext(s.ctx, method, requestURL, &buf)
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
		req, err = http.NewRequestWithContext(s.ctx, method, requestURL, nil)
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
		content, err := io.ReadAll(resp.Body)
		if err == nil && string(content) != "" {
			return fmt.Errorf("%s", strings.TrimSpace(string(content)))
		}

		return fmt.Errorf("%s: %s", requestURL, resp.Status)
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
