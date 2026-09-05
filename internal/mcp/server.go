package mcp

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/updu/updu/internal/diagnostic"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// ToolDefinition defines an MCP tool schema.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Server provides Model Context Protocol (MCP) capabilities over stdio.
type Server struct {
	db     *storage.DB
	reader *bufio.Reader
	writer io.Writer
}

// NewServer creates a new MCP Server.
func NewServer(db *storage.DB, r io.Reader, w io.Writer) *Server {
	return &Server{
		db:     db,
		reader: bufio.NewReader(r),
		writer: w,
	}
}

// Run starts the stdio read loop for JSON-RPC 2.0 messages.
func (s *Server) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = []byte(strings.TrimSpace(string(line)))
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(ctx, &req)
	}
}

func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.sendResult(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "updu-mcp",
				"version": "2.0.0",
			},
			"capabilities": map[string]any{
				"tools": map[string]bool{"listChanged": false},
			},
		})

	case "notifications/initialized":
		// No response required for notifications

	case "tools/list":
		s.sendResult(req.ID, map[string]any{
			"tools": s.getTools(),
		})

	case "tools/call":
		s.handleToolCall(ctx, req)

	default:
		s.sendError(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func (s *Server) getTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "list_services",
			Description: "Returns all monitored services, their operational health status, primary latency, and root-cause diagnoses.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"zone": map[string]string{
						"type":        "string",
						"description": "Optional zone filter (e.g. 'default', 'homelab', 'vps')",
					},
				},
			},
		},
		{
			Name:        "get_service_details",
			Description: "Retrieves complete details for a specific service, including all vantage endpoints (LAN, Tailnet, WAN) and diagnostic autopsy.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"service_id": map[string]string{
						"type":        "string",
						"description": "The unique ID of the service to inspect",
					},
				},
				"required": []string{"service_id"},
			},
		},
		{
			Name:        "get_network_topology",
			Description: "Returns the network topology graph showing prober nodes, active zones, reachability fabrics (LAN, Tailnet, Public WAN), and probe edges.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_tls_certificates",
			Description: "Returns audited TLS/SSL certificates, expiration dates, days remaining, issuers, and Subject Alternative Names (SANs).",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "audit_tls_handshake",
			Description: "Performs a live on-demand TLS handshake audit against any hostname and port, validating trust chain and certificate expiration.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"host": map[string]string{
						"type":        "string",
						"description": "The domain name or FQDN to audit (e.g. 'vault.kn8design.com')",
					},
					"port": map[string]any{
						"type":        "integer",
						"description": "Port number (defaults to 443)",
					},
				},
				"required": []string{"host"},
			},
		},
	}
}

func (s *Server) handleToolCall(ctx context.Context, req *JSONRPCRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	var output any
	var err error

	switch params.Name {
	case "list_services":
		services, listErr := s.db.ListServices(ctx)
		if listErr != nil {
			err = listErr
		} else {
			for _, svc := range services {
				checksMap := make(map[string]*models.EndpointCheck)
				for _, ep := range svc.Endpoints {
					if ep.LastCheckedAt != nil {
						checksMap[ep.ID] = &models.EndpointCheck{
							ServiceID:  svc.ID,
							EndpointID: ep.ID,
							Status:     ep.Status,
							LatencyMs:  ep.LastLatency,
							StatusCode: ep.LastStatusCode,
							Message:    ep.LastMessage,
							CheckedAt:  *ep.LastCheckedAt,
						}
					}
					if ep.IsPrimary && ep.LastLatency != nil {
						svc.PrimaryLatency = ep.LastLatency
					}
				}
				diag := diagnostic.EvaluateServiceHealth(svc, checksMap)
				svc.Status = diag.Status
				svc.Diagnosis = diag.Summary
			}
			output = services
		}

	case "get_service_details":
		serviceID, _ := params.Arguments["service_id"].(string)
		if serviceID == "" {
			err = fmt.Errorf("service_id argument is required")
		} else {
			svc, getErr := s.db.GetService(ctx, serviceID)
			if getErr != nil {
				err = getErr
			} else if svc == nil {
				err = fmt.Errorf("service not found: %s", serviceID)
			} else {
				checksMap := make(map[string]*models.EndpointCheck)
				for _, ep := range svc.Endpoints {
					if ep.LastCheckedAt != nil {
						checksMap[ep.ID] = &models.EndpointCheck{
							ServiceID:  svc.ID,
							EndpointID: ep.ID,
							Status:     ep.Status,
							LatencyMs:  ep.LastLatency,
							StatusCode: ep.LastStatusCode,
							Message:    ep.LastMessage,
							CheckedAt:  *ep.LastCheckedAt,
						}
					}
					if ep.IsPrimary && ep.LastLatency != nil {
						svc.PrimaryLatency = ep.LastLatency
					}
				}
				diag := diagnostic.EvaluateServiceHealth(svc, checksMap)
				svc.Status = diag.Status
				svc.Diagnosis = diag.Summary
				output = map[string]any{
					"service":   svc,
					"diagnosis": diag,
				}
			}
		}

	case "get_network_topology":
		topo, topoErr := s.db.GetNetworkTopology(ctx, "local-node", "Primary Node")
		if topoErr != nil {
			err = topoErr
		} else {
			output = topo
		}

	case "get_tls_certificates":
		certs, certErr := s.db.ListTLSCertificates(ctx)
		if certErr != nil {
			err = certErr
		} else {
			output = certs
		}

	case "audit_tls_handshake":
		host, _ := params.Arguments["host"].(string)
		port := 443
		if pVal, ok := params.Arguments["port"].(float64); ok && pVal > 0 {
			port = int(pVal)
		}
		if host == "" {
			err = fmt.Errorf("host argument is required")
		} else {
			addr := net.JoinHostPort(host, strconv.Itoa(port))
			dialer := &net.Dialer{Timeout: 5 * time.Second}
			conn, dialErr := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
				ServerName: host,
			})
			if dialErr != nil {
				err = fmt.Errorf("TLS handshake failed: %w", dialErr)
			} else {
				defer conn.Close()
				state := conn.ConnectionState()
				if len(state.PeerCertificates) == 0 {
					err = fmt.Errorf("no peer certificates returned")
				} else {
					leaf := state.PeerCertificates[0]
					daysRemaining := int(time.Until(leaf.NotAfter).Hours() / 24)
					output = map[string]any{
						"domain":              host,
						"issuer":              leaf.Issuer.CommonName,
						"subject":             leaf.Subject.CommonName,
						"sans":                leaf.DNSNames,
						"valid_from":          leaf.NotBefore.Format(time.RFC3339),
						"valid_until":         leaf.NotAfter.Format(time.RFC3339),
						"days_remaining":      daysRemaining,
						"serial_number":       leaf.SerialNumber.String(),
						"signature_algorithm": leaf.SignatureAlgorithm.String(),
					}
				}
			}
		}

	default:
		s.sendError(req.ID, -32601, "Unknown tool: "+params.Name)
		return
	}

	if err != nil {
		s.sendResult(req.ID, map[string]any{
			"isError": true,
			"content": []map[string]string{
				{"type": "text", "text": "Error: " + err.Error()},
			},
		})
		return
	}

	outputBytes, _ := json.MarshalIndent(output, "", "  ")
	s.sendResult(req.ID, map[string]any{
		"content": []map[string]string{
			{"type": "text", "text": string(outputBytes)},
		},
	})
}

func (s *Server) sendResult(id any, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	bytes, _ := json.Marshal(resp)
	_, _ = s.writer.Write(append(bytes, '\n'))
}

func (s *Server) sendError(id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: map[string]any{
			"code":    code,
			"message": message,
		},
	}
	bytes, _ := json.Marshal(resp)
	_, _ = s.writer.Write(append(bytes, '\n'))
}
