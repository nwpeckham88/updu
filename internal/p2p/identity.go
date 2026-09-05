package p2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/updu/updu/internal/storage"
)

// NodeIdentity represents the cryptographic identity of the local node.
type NodeIdentity struct {
	NodeID     string             `json:"node_id"`
	Name       string             `json:"name"`
	PublicKey  ed25519.PublicKey  `json:"-"`
	PrivateKey ed25519.PrivateKey `json:"-"`
	PubKeyHex  string             `json:"public_key"`
}

// DeriveNodeID derives a clean, human-readable 8-character ID (format: XXXX-XXXX)
// from an Ed25519 public key using Crockford/RFC4648 Base32.
func DeriveNodeID(pubKey ed25519.PublicKey) string {
	hash := sha256.Sum256(pubKey)
	// 5 bytes = 40 bits = exactly 8 base32 characters
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:5])
	enc = strings.ToUpper(enc)
	if len(enc) < 8 {
		enc = fmt.Sprintf("%-8s", enc)
	}
	return fmt.Sprintf("%s-%s", enc[:4], enc[4:8])
}

// EnsureIdentity loads the existing node identity from the settings table,
// or creates and stores a new Ed25519 keypair and derived Node ID.
func EnsureIdentity(ctx context.Context, db *storage.DB, customName string) (*NodeIdentity, error) {
	nodeID, _ := db.GetSetting(ctx, "p2p_node_id")
	pubHex, _ := db.GetSetting(ctx, "p2p_public_key")
	privHex, _ := db.GetSetting(ctx, "p2p_private_key")
	name, _ := db.GetSetting(ctx, "p2p_node_name")

	if nodeID != "" && pubHex != "" && privHex != "" {
		pubBytes, err := hex.DecodeString(pubHex)
		if err == nil && len(pubBytes) == ed25519.PublicKeySize {
			privBytes, err := hex.DecodeString(privHex)
			if err == nil && len(privBytes) == ed25519.PrivateKeySize {
				if customName != "" && customName != name {
					name = customName
					_ = db.SetSetting(ctx, "p2p_node_name", name)
				}
				if name == "" {
					hostname, _ := os.Hostname()
					if hostname == "" {
						hostname = "updu-" + nodeID
					}
					name = hostname
					_ = db.SetSetting(ctx, "p2p_node_name", name)
				}
				return &NodeIdentity{
					NodeID:     nodeID,
					Name:       name,
					PublicKey:  ed25519.PublicKey(pubBytes),
					PrivateKey: ed25519.PrivateKey(privBytes),
					PubKeyHex:  pubHex,
				}, nil
			}
		}
	}

	// Generate new Ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating ed25519 key: %w", err)
	}

	newNodeID := DeriveNodeID(pub)
	pubHex = hex.EncodeToString(pub)
	privHex = hex.EncodeToString(priv)

	if customName != "" {
		name = customName
	} else {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "updu-" + newNodeID
		}
		name = hostname
	}

	if err := db.SetSetting(ctx, "p2p_node_id", newNodeID); err != nil {
		return nil, fmt.Errorf("saving p2p_node_id: %w", err)
	}
	if err := db.SetSetting(ctx, "p2p_public_key", pubHex); err != nil {
		return nil, fmt.Errorf("saving p2p_public_key: %w", err)
	}
	if err := db.SetSetting(ctx, "p2p_private_key", privHex); err != nil {
		return nil, fmt.Errorf("saving p2p_private_key: %w", err)
	}
	if err := db.SetSetting(ctx, "p2p_node_name", name); err != nil {
		return nil, fmt.Errorf("saving p2p_node_name: %w", err)
	}

	slog.Info("initialized P2P node identity", "node_id", newNodeID, "name", name)

	return &NodeIdentity{
		NodeID:     newNodeID,
		Name:       name,
		PublicKey:  pub,
		PrivateKey: priv,
		PubKeyHex:  pubHex,
	}, nil
}

// Sign creates an Ed25519 signature over data and returns it as a hex string.
func (id *NodeIdentity) Sign(data []byte) string {
	sig := ed25519.Sign(id.PrivateKey, data)
	return hex.EncodeToString(sig)
}

// VerifySignature verifies that the signature matches the data using the provided public key hex.
func VerifySignature(pubKeyHex string, data []byte, sigHex string) bool {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(pubBytes, data, sigBytes)
}
