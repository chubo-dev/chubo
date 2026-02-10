package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func main() {
	var outDir string
	var payload string
	var payloadFile string

	flag.StringVar(&outDir, "out-dir", "", "output directory (created if missing)")
	flag.StringVar(&payload, "payload", "{\"hello\":\"world\"}\n", "bootstrap JSON payload (must be valid JSON)")
	flag.StringVar(&payloadFile, "payload-file", "", "path to JSON payload file (overrides -payload)")
	flag.Parse()

	if outDir == "" {
		fmt.Fprintln(os.Stderr, "-out-dir is required")
		os.Exit(2)
	}

	var payloadBytes []byte
	if payloadFile != "" {
		b, err := os.ReadFile(payloadFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read payload file: %v\n", err)
			os.Exit(1)
		}

		payloadBytes = b
	} else {
		payloadBytes = []byte(payload)
	}

	if !json.Valid(payloadBytes) {
		fmt.Fprintln(os.Stderr, "payload must be valid JSON")
		os.Exit(2)
	}

	if err := os.MkdirAll(outDir, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ed25519 keygen: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "chuboos-bootstrap-signer"},
		NotBefore:             now.Add(-1 * time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create signer cert: %v\n", err)
		os.Exit(1)
	}

	signerPath := filepath.Join(outDir, "signer.pem")
	if err := os.WriteFile(signerPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write signer cert: %v\n", err)
		os.Exit(1)
	}

	headerBytes, err := json.Marshal(map[string]string{"alg": "EdDSA"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal JWS header: %v\n", err)
		os.Exit(1)
	}

	protectedB64 := b64url(headerBytes)
	payloadB64 := b64url(payloadBytes)
	signingInput := protectedB64 + "." + payloadB64
	sig := ed25519.Sign(priv, []byte(signingInput))
	jws := signingInput + "." + b64url(sig)

	jwsPath := filepath.Join(outDir, "payload.jws")
	if err := os.WriteFile(jwsPath, []byte(jws), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write JWS payload: %v\n", err)
		os.Exit(1)
	}

	fp := sha256.Sum256(der)

	// Shell-friendly outputs for runbooks.
	fmt.Printf("OUT_DIR=%s\n", outDir)
	fmt.Printf("SIGNER_PEM=%s\n", signerPath)
	fmt.Printf("PAYLOAD_JWS=%s\n", jwsPath)
	fmt.Printf("SIGNER_SHA256=%s\n", hex.EncodeToString(fp[:]))
}
