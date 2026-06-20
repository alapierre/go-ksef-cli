package qr

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go-ksef-cli/internal/config"
	"go-ksef-cli/pkg/bmp"

	"github.com/alapierre/go-ksef-client/ksef"
	"github.com/alapierre/go-ksef-client/ksef/keys"
	ksefqr "github.com/alapierre/go-ksef-client/ksef/qr"
	"github.com/alapierre/go-ksef-client/png"
	"golang.org/x/term"
)

type Cmd struct {
	Certificate CertificateCmd `cmd:"certificate" aliases:"cert,qr2" help:"Generate KSeF certificate verification QR Code II"`
}

type CertificateCmd struct {
	Cert       string `short:"c" required:"" type:"existingfile" help:"KSeF certificate file"`
	Key        string `short:"k" required:"" type:"existingfile" help:"Private key file"`
	SellerNIP  string `short:"s" name:"seller-nip" help:"Seller NIP, if different from the invoice issuer NIP"`
	ContextNIP string `short:"n" name:"context-nip" required:"" help:"Invoice issuer NIP (KSeF context)"`
	Out        string `short:"o" type:"path" default:"." help:"Base output path for QR Code file"`
	Redirect   string `short:"r" type:"path" help:"File for appending signed links line by line"`
	Format     string `short:"f" enum:"png,bmp" default:"png" help:"Output image format (png or bmp)"`
	InPath     string `short:"i" name:"in" type:"path" help:"Optional invoice XML file or any other file to sign. Use '-' to read from stdin"`
}

func (c *CertificateCmd) Run(cfg *config.Config) error {
	pass := os.Getenv("KSEF_KEY_PASSWORD")
	if pass == "" {
		var err error
		pass, err = readPassword("Enter private key password: ")
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
	}

	certBytes, err := os.ReadFile(c.Cert)
	if err != nil {
		return fmt.Errorf("read certificate: %w", err)
	}

	cert, err := ksefqr.LoadCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("load certificate: %w", err)
	}

	serial, err := ksefqr.ExtractCertSerial(cert)
	if err != nil {
		return fmt.Errorf("extract certificate serial number: %w", err)
	}

	keyBytes, err := os.ReadFile(c.Key)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}

	key, err := keys.LoadEncryptedPKCS8SignerFromPEM(keyBytes, []byte(pass))
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}

	var env ksef.Environment
	if err := env.UnmarshalText([]byte(cfg.Env)); err != nil {
		return fmt.Errorf("invalid KSeF environment: %w", err)
	}

	sellerNIP := c.SellerNIP
	if sellerNIP == "" {
		sellerNIP = c.ContextNIP
	}

	sum, err := invoiceHash(c.InPath)
	if err != nil {
		return err
	}

	url, err := ksefqr.GenerateCertificateVerificationLink(
		env,
		ksefqr.CtxNip,
		c.ContextNIP,
		sellerNIP,
		serial,
		key,
		sum[:],
	)
	if err != nil {
		return fmt.Errorf("generate link: %w", err)
	}

	fmt.Printf("Link generated for seller NIP %s, KSeF context (issuer) %s, environment: %s, certificate serial number %s\n", sellerNIP, c.ContextNIP, env.Name(), serial)
	fmt.Println(url)

	if c.Redirect != "" {
		line := fmt.Sprintf("seller NIP %s, KSeF context (issuer) %s, certificate serial number: %s, link: %s", sellerNIP, c.ContextNIP, serial, url)
		if err := appendLine(c.Redirect, line); err != nil {
			fmt.Printf("cannot write output information to file %s: %s\n", c.Redirect, err)
		}
	}

	img, ext, err := qrImage(url, c.Format)
	if err != nil {
		return err
	}

	outPath := filepath.Join(c.Out, fmt.Sprintf("%s_qr2.%s", c.ContextNIP, ext))
	if err := os.WriteFile(outPath, img, 0o644); err != nil {
		return fmt.Errorf("write QR code: %w", err)
	}

	fmt.Printf("QR code saved to %s. Your visualization will be formally correct. Substantively... maybe. Happy KSeFing!\n", outPath)
	return nil
}

func invoiceHash(inPath string) ([32]byte, error) {
	switch strings.TrimSpace(inPath) {
	case "":
		fmt.Println("Using default content instead of a real invoice. The QR code will still be valid.")
		return sha256.Sum256([]byte("KSeF is dead, baby, KSeF is dead...")), nil
	case "-":
		return sha(os.Stdin)
	default:
		f, err := os.Open(inPath)
		if err != nil {
			return [32]byte{}, fmt.Errorf("open file to sign: %w", err)
		}
		defer f.Close()

		return sha(f)
	}
}

func sha(r io.Reader) ([32]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return [32]byte{}, fmt.Errorf("calculate SHA from signed file content: %w", err)
	}

	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return strings.TrimSpace(string(bytePassword)), nil
}

func appendLine(path, line string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, line)
	return err
}

func qrImage(content, format string) ([]byte, string, error) {
	switch format {
	case "png":
		img, err := png.Qr(content)
		return img, "png", err
	case "bmp":
		img, err := bmp.Qr(content)
		return img, "bmp", err
	default:
		return nil, "", fmt.Errorf("unsupported image format: %s", format)
	}
}
